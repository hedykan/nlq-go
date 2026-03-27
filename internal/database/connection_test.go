package database

import (
	"context"
	"testing"
	"time"

	"github.com/channelwill/nlq/internal/config"
)

// TestNewConnection 测试创建数据库连接
func TestNewConnection(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}

	if db == nil {
		t.Fatal("期望返回非nil的数据库连接")
	}

	// 验证连接可用
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层SQL数据库连接失败: %v", err)
	}

	// 测试Ping
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("数据库Ping失败: %v", err)
	}

	// 关闭连接
	if err := sqlDB.Close(); err != nil {
		t.Errorf("关闭数据库连接失败: %v", err)
	}
}

// TestNewConnection_InvalidConfig 测试无效配置
func TestNewConnection_InvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.DatabaseConfig
		wantErr bool
	}{
		{
			name: "空配置",
			config: &config.DatabaseConfig{
				Driver:   "",
				Host:     "",
				Port:     0,
				Database: "",
				Username: "",
				Password: "",
			},
			wantErr: true,
		},
		{
			name: "缺少主机",
			config: &config.DatabaseConfig{
				Driver:   "mysql",
				Host:     "",
				Port:     3306,
				Database: "testdb",
				Username: "root",
				Password: "root",
			},
			wantErr: true,
		},
		{
			name: "无效的驱动",
			config: &config.DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				Username: "root",
				Password: "root",
			},
			wantErr: true,
		},
		{
			name: "无效端口",
			config: &config.DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     -1,
				Database: "testdb",
				Username: "root",
				Password: "root",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConnection(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewConnection_BuildDSN 测试DSN构建
func TestNewConnection_BuildDSN(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	expectedDSN := "root:root@tcp(localhost:3306)/loloyal?charset=utf8mb4&parseTime=True&loc=Local&readTimeout=30s&timeout=10s"
	actualDSN := buildDSN(cfg)

	if actualDSN != expectedDSN {
		t.Errorf("DSN不匹配\n期望: %s\n实际: %s", expectedDSN, actualDSN)
	}
}

// TestConnection_WithReadonly 测试只读连接
func TestConnection_WithReadonly(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 验证连接池设置
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层SQL数据库连接失败: %v", err)
	}

	// 测试连接设置
	if sqlDB.Ping() != nil {
		t.Error("数据库Ping失败")
	}
}

// TestConnection_ConnectionPool 测试连接池设置
func TestConnection_ConnectionPool(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层SQL数据库连接失败: %v", err)
	}

	// 验证连接池设置
	stats := sqlDB.Stats()

	// 验证最大打开连接数
	if stats.MaxOpenConnections == 0 {
		t.Error("期望设置最大打开连接数")
	}

	// 验证连接可用
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("数据库Ping失败: %v", err)
	}
}

// TestNewConnection_WithTimeout 测试超时设置
func TestNewConnection_WithTimeout(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层SQL数据库连接失败: %v", err)
	}

	// 验证连接设置
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("数据库Ping失败: %v", err)
	}

	// 设置较短的超时时间以测试
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 这个查询应该超时
	start := time.Now()
	err = db.WithContext(ctx).Exec("SELECT SLEEP(1)").Error
	elapsed := time.Since(start)

	if err == nil {
		t.Error("期望查询超时，但成功了")
	}

	if elapsed < 1*time.Millisecond {
		t.Errorf("查询超时时间不正确，期望约1ms，实际: %v", elapsed)
	}
}

// TestConnection_Close 测试关闭连接
func TestConnection_Close(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层SQL数据库连接失败: %v", err)
	}

	// 关闭连接
	if err := sqlDB.Close(); err != nil {
		t.Errorf("关闭数据库连接失败: %v", err)
	}

	// 验证连接已关闭
	if err := sqlDB.Ping(); err == nil {
		t.Error("期望关闭后Ping失败，但成功了")
	}
}

// TestValidateConnection_Valid 测试验证有效连接
func TestValidateConnection_Valid(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	err = ValidateConnection(db)
	if err != nil {
		t.Errorf("验证有效连接失败: %v", err)
	}
}

// TestValidateConnection_Nil 测试验证nil连接
func TestValidateConnection_Nil(t *testing.T) {
	err := ValidateConnection(nil)
	if err == nil {
		t.Error("期望验证nil连接返回错误")
	}
}

// TestCloseConnection_Valid 测试关闭数据库连接
func TestCloseConnection_Valid(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}

	err = CloseConnection(db)
	if err != nil {
		t.Errorf("关闭连接失败: %v", err)
	}

	// 验证SSH隧道已重置
	tunnel := GetSSHTunnel()
	if tunnel != nil {
		t.Error("期望关闭后SSH隧道为nil")
	}
}

// TestCloseConnection_Nil 测试关闭nil连接
func TestCloseConnection_Nil(t *testing.T) {
	err := CloseConnection(nil)
	if err != nil {
		t.Errorf("关闭nil连接应该不返回错误: %v", err)
	}
}

// TestGetSSHTunnel 测试获取SSH隧道
func TestGetSSHTunnel(t *testing.T) {
	// 没有SSH隧道时应该返回nil
	tunnel := GetSSHTunnel()
	if tunnel != nil {
		t.Error("期望没有SSH隧道时返回nil")
	}
}

// TestNewConnection_SSHEnabled_ValidationFailure 测试SSH配置验证失败
func TestNewConnection_SSHEnabled_ValidationFailure(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.DatabaseConfig
		wantErr bool
	}{
		{
			name: "SSH启用但缺少主机",
			config: &config.DatabaseConfig{
				Driver:     "mysql",
				Host:       "localhost",
				Port:       3306,
				Database:   "testdb",
				Username:   "root",
				Password:   "root",
				SSHEnabled: true,
				SSHHost:    "",
			},
			wantErr: true,
		},
		{
			name: "SSH启用但缺少用户",
			config: &config.DatabaseConfig{
				Driver:     "mysql",
				Host:       "localhost",
				Port:       3306,
				Database:   "testdb",
				Username:   "root",
				Password:   "root",
				SSHEnabled: true,
				SSHHost:    "example.com",
				SSHUser:    "",
			},
			wantErr: true,
		},
		{
			name: "SSH启用但缺少认证",
			config: &config.DatabaseConfig{
				Driver:           "mysql",
				Host:             "localhost",
				Port:             3306,
				Database:         "testdb",
				Username:         "root",
				Password:         "root",
				SSHEnabled:       true,
				SSHHost:          "example.com",
				SSHUser:          "testuser",
				SSHPassword:      "",
				SSHPrivateKeyFile: "",
			},
			wantErr: true,
		},
		{
			name: "SSH启用但私钥文件不存在",
			config: &config.DatabaseConfig{
				Driver:           "mysql",
				Host:             "localhost",
				Port:             3306,
				Database:         "testdb",
				Username:         "root",
				Password:         "root",
				SSHEnabled:       true,
				SSHHost:          "example.com",
				SSHUser:          "testuser",
				SSHPrivateKeyFile: "/nonexistent/key.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConnection(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewConnection_SSHDisabled_Compatibility 测试不使用SSH时的向后兼容性
func TestNewConnection_SSHDisabled_Compatibility(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver:     "mysql",
		Host:       "localhost",
		Port:       3306,
		Database:   "loloyal",
		Username:   "root",
		Password:   "root",
		Readonly:   true,
		SSHEnabled: false, // 显式禁用
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer CloseConnection(db)

	// 验证连接可用
	if err := ValidateConnection(db); err != nil {
		t.Errorf("验证连接失败: %v", err)
	}
}
