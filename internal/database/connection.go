package database

import (
	"fmt"
	"time"

	"github.com/channelwill/nlq/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewConnection 创建数据库连接
func NewConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 验证配置
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为nil")
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
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层SQL DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取SQL数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)                    // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)                   // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour)          // 连接最大生命周期
	sqlDB.SetConnMaxIdleTime(time.Minute * 10)   // 最大空闲时间

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
