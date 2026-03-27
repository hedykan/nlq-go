package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/channelwill/nlq/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 全局SSH隧道实例
var (
	sshTunnel *SSHTunnel
	sshOnce   sync.Once
)

// NewConnection 创建数据库连接
func NewConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 验证配置
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为nil")
	}

	// 验证SSH配置
	if err := cfg.ValidateSSHConfig(); err != nil {
		return nil, fmt.Errorf("SSH配置验证失败: %w", err)
	}

	// 保存原始配置
	originalHost := cfg.Host
	originalPort := cfg.Port

	// 如果启用SSH隧道，先建立隧道
	if cfg.SSHEnabled {
		tunnel, err := createSSHTunnel(cfg)
		if err != nil {
			return nil, fmt.Errorf("创建SSH隧道失败: %w", err)
		}

		// 使用sync.Once确保只设置一次全局隧道
		sshOnce.Do(func() {
			sshTunnel = tunnel
		})

		// 连接SSH服务器
		if err := tunnel.Connect(); err != nil {
			return nil, fmt.Errorf("SSH隧道连接失败: %w", err)
		}

		// 端口转发
		tunnelAddr, err := tunnel.ForwardPort(cfg.Host, cfg.Port)
		if err != nil {
			tunnel.Close()
			return nil, fmt.Errorf("SSH端口转发失败: %w", err)
		}

		// 修改配置以使用本地隧道端口
		host, port, err := parseAddress(tunnelAddr)
		if err != nil {
			tunnel.Close()
			return nil, fmt.Errorf("解析隧道地址失败: %w", err)
		}
		cfg.Host = host
		cfg.Port = port
	}

	// 构建DSN
	dsn := buildDSN(cfg)

	// GORM配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// 创建连接
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		// 如果SSH隧道已创建，连接失败时需要关闭
		if cfg.SSHEnabled && sshTunnel != nil {
			sshTunnel.Close()
		}
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		if cfg.SSHEnabled && sshTunnel != nil {
			sshTunnel.Close()
		}
		return nil, fmt.Errorf("获取SQL数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(time.Minute * 10) // 最大空闲时间

	// 恢复原始配置
	cfg.Host = originalHost
	cfg.Port = originalPort

	return db, nil
}

// buildDSN 构建MySQL DSN
func buildDSN(cfg *config.DatabaseConfig) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	// 使用配置的超时时间，如果没有配置则使用默认值
	params := "charset=utf8mb4&parseTime=True&loc=Local"

	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 30 * time.Second // 默认30秒
	}
	params += fmt.Sprintf("&readTimeout=%s", readTimeout)

	connectTimeout := cfg.ConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = 10 * time.Second // 默认10秒
	}
	params += fmt.Sprintf("&timeout=%s", connectTimeout)

	return dsn + params
}

// ValidateConnection 验证数据库连接
func ValidateConnection(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接为nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取SQL数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	return nil
}

// createSSHTunnel 创建SSH隧道
func createSSHTunnel(cfg *config.DatabaseConfig) (*SSHTunnel, error) {
	sshConfig := &SSHConfig{
		Host:           cfg.SSHHost,
		Port:           cfg.SSHPort,
		User:           cfg.SSHUser,
		Password:       cfg.SSHPassword,
		PrivateKeyFile: cfg.SSHPrivateKeyFile,
		KeyPassphrase:  cfg.SSHKeyPassphrase,
	}

	return NewSSHTunnel(sshConfig)
}

// CloseConnection 关闭数据库连接和SSH隧道
func CloseConnection(db *gorm.DB) error {
	// 关闭数据库连接
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}

	// 关闭SSH隧道
	if sshTunnel != nil {
		err := sshTunnel.Close()
		sshTunnel = nil
		sshOnce = sync.Once{} // 重置sync.Once
		return err
	}

	return nil
}

// GetSSHTunnel 获取当前SSH隧道实例（用于测试和监控）
func GetSSHTunnel() *SSHTunnel {
	return sshTunnel
}
