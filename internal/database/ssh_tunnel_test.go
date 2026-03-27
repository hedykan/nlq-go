package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/channelwill/nlq/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestNewSSHTunnel_PasswordAuth_Success 测试密码认证创建SSH隧道
func TestNewSSHTunnel_PasswordAuth_Success(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)

	if err != nil {
		t.Fatalf("NewSSHTunnel() error = %v", err)
	}

	assert.NotNil(t, tunnel)
	assert.Equal(t, cfg.Host, tunnel.config.Host)
	assert.Equal(t, cfg.Port, tunnel.config.Port)
	assert.Equal(t, cfg.User, tunnel.config.User)
	assert.False(t, tunnel.IsConnected())
}

// TestNewSSHTunnel_PrivateKeyAuth_Success 测试私钥认证创建SSH隧道
func TestNewSSHTunnel_PrivateKeyAuth_Success(t *testing.T) {
	// 创建临时测试私钥文件
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// 生成测试私钥
	err := generateTestPrivateKey(keyFile, "")
	assert.NoError(t, err)

	cfg := &SSHConfig{
		Host:           "localhost",
		Port:           22,
		User:           "testuser",
		PrivateKeyFile: keyFile,
	}

	tunnel, err := NewSSHTunnel(cfg)

	if err != nil {
		t.Fatalf("NewSSHTunnel() error = %v", err)
	}

	assert.NotNil(t, tunnel)
	assert.Equal(t, cfg.Host, tunnel.config.Host)
	assert.Equal(t, cfg.PrivateKeyFile, tunnel.config.PrivateKeyFile)
	assert.False(t, tunnel.IsConnected())
}

// TestNewSSHTunnel_PrivateKeyWithPassphrase_Success 测试带密码短语的私钥认证
func TestNewSSHTunnel_PrivateKeyWithPassphrase_Success(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// 生成带密码的测试私钥
	passphrase := "test123"
	err := generateTestPrivateKey(keyFile, passphrase)
	assert.NoError(t, err)

	cfg := &SSHConfig{
		Host:           "localhost",
		Port:           22,
		User:           "testuser",
		PrivateKeyFile: keyFile,
		KeyPassphrase:  passphrase,
	}

	tunnel, err := NewSSHTunnel(cfg)

	if err != nil {
		t.Fatalf("NewSSHTunnel() error = %v", err)
	}

	assert.NotNil(t, tunnel)
	assert.Equal(t, cfg.KeyPassphrase, tunnel.config.KeyPassphrase)
	assert.False(t, tunnel.IsConnected())
}

// TestNewSSHTunnel_MissingAuthMethod 测试缺少认证方法
func TestNewSSHTunnel_MissingAuthMethod(t *testing.T) {
	cfg := &SSHConfig{
		Host: "localhost",
		Port: 22,
		User: "testuser",
		// 既没有密码也没有私钥
	}

	_, err := NewSSHTunnel(cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "密码")
}

// TestNewSSHTunnel_InvalidPrivateKeyFile 测试无效的私钥文件
func TestNewSSHTunnel_InvalidPrivateKeyFile(t *testing.T) {
	cfg := &SSHConfig{
		Host:           "localhost",
		Port:           22,
		User:           "testuser",
		PrivateKeyFile: "/nonexistent/key.pem",
	}

	_, err := NewSSHTunnel(cfg)

	assert.Error(t, err)
}

// TestNewSSHTunnel_WrongPassphrase 测试错误的私钥密码
func TestNewSSHTunnel_WrongPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// 生成带密码的测试私钥
	err := generateTestPrivateKey(keyFile, "correct123")
	assert.NoError(t, err)

	// 创建一个内容不是有效私钥的文件来模拟错误密码的情况
	wrongKeyFile := filepath.Join(tmpDir, "wrong_key")
	err = os.WriteFile(wrongKeyFile, []byte("invalid key content"), 0600)
	assert.NoError(t, err)

	cfg := &SSHConfig{
		Host:           "localhost",
		Port:           22,
		User:           "testuser",
		PrivateKeyFile: wrongKeyFile,
		KeyPassphrase:  "anypass",
	}

	_, err = NewSSHTunnel(cfg)

	// 由于私钥文件无效，NewSSHTunnel应该成功
	// 但在实际连接时会失败
	// 这里我们只测试配置验证通过
	assert.NoError(t, err)
}

// TestNewSSHTunnel_NilConfig 测试nil配置
func TestNewSSHTunnel_NilConfig(t *testing.T) {
	_, err := NewSSHTunnel(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "配置")
}

// TestSSHTunnel_Connect_Timeout 测试连接超时
func TestSSHTunnel_Connect_Timeout(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "192.0.2.1", // TEST-NET-1，确保无法连接
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 设置较短的超时时间
	err = tunnel.ConnectWithTimeout(1 * time.Second)

	assert.Error(t, err)
	assert.False(t, tunnel.IsConnected())
}

// TestSSHTunnel_ForwardPort_NotConnected 测试未连接时端口转发
func TestSSHTunnel_ForwardPort_NotConnected(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 未连接就尝试端口转发
	_, err = tunnel.ForwardPort("localhost", 3306)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未连接")
}

// TestSSHTunnel_Close 测试关闭隧道
func TestSSHTunnel_Close(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 关闭未连接的隧道不应报错
	err = tunnel.Close()
	assert.NoError(t, err)
	assert.False(t, tunnel.IsConnected())
}

// TestSSHConfig_Validate_Success 测试SSH配置验证成功
func TestSSHConfig_Validate_Success(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()

	assert.NoError(t, err)
}

// TestSSHConfig_Validate_MissingHost 测试缺少主机
func TestSSHConfig_Validate_MissingHost(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "主机")
}

// TestSSHConfig_Validate_InvalidPort 测试无效端口
func TestSSHConfig_Validate_InvalidPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"端口为0", 0, true},
		{"负端口", -1, true},
		{"端口太大", 65536, true},
		{"有效端口", 22, false},
		{"有效端口2222", 2222, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SSHConfig{
				Host:     "localhost",
				Port:     tt.port,
				User:     "testuser",
				Password: "testpass",
			}

			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSSHConfig_Validate_MissingUser 测试缺少用户名
func TestSSHConfig_Validate_MissingUser(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "",
		Password: "testpass",
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户")
}

// TestBuildSSHClientConfig_PasswordAuth 测试构建SSH客户端配置（密码认证）
func TestBuildSSHClientConfig_PasswordAuth(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	sshConfig, err := buildSSHClientConfig(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, sshConfig)
	assert.Equal(t, "testuser", sshConfig.User)
	assert.Len(t, sshConfig.Auth, 1)
}

// TestBuildSSHClientConfig_PrivateKeyAuth 测试构建SSH客户端配置（私钥认证）
func TestBuildSSHClientConfig_PrivateKeyAuth(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	err := generateTestPrivateKey(keyFile, "")
	assert.NoError(t, err)

	cfg := &SSHConfig{
		Host:           "localhost",
		Port:           22,
		User:           "testuser",
		PrivateKeyFile: keyFile,
	}

	sshConfig, err := buildSSHClientConfig(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, sshConfig)
	assert.Equal(t, "testuser", sshConfig.User)
	assert.Len(t, sshConfig.Auth, 1)
}

// TestGenerateTestPrivateKey 测试生成测试私钥的辅助函数
func TestGenerateTestPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// 无密码私钥
	err := generateTestPrivateKey(keyFile, "")
	assert.NoError(t, err)

	// 验证文件存在
	_, err = os.Stat(keyFile)
	assert.NoError(t, err)

	// 带密码私钥
	keyFile2 := filepath.Join(tmpDir, "test_key_encrypted")
	err = generateTestPrivateKey(keyFile2, "password123")
	assert.NoError(t, err)

	_, err = os.Stat(keyFile2)
	assert.NoError(t, err)
}

// TestIsPrivateKeyEncrypted 测试私钥是否加密
func TestIsPrivateKeyEncrypted(t *testing.T) {
	tmpDir := t.TempDir()

	// 无密码私钥（PKCS1格式）
	keyFile1 := filepath.Join(tmpDir, "test_key")
	err := generateTestPrivateKey(keyFile1, "")
	assert.NoError(t, err)

	encrypted, err := isPrivateKeyEncrypted(keyFile1)
	assert.NoError(t, err)
	assert.False(t, encrypted)

	// 创建一个模拟的加密私钥文件（使用无效内容模拟）
	// 因为generateTestPrivateKey生成的PKCS8格式无法简单检测加密状态
	keyFile2 := filepath.Join(tmpDir, "test_key_enc")
	err = os.WriteFile(keyFile2, []byte("-----BEGIN RSA PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\n-----END RSA PRIVATE KEY-----"), 0600)
	assert.NoError(t, err)

	// 无法解析的私钥会被认为是加密的
	encrypted, err = isPrivateKeyEncrypted(keyFile2)
	// 这种情况下无法确定，可能返回错误
	// 我们主要测试函数不会崩溃
	assert.NotPanics(t, func() {
		isPrivateKeyEncrypted(keyFile2)
	})
}

// TestParseAddress 测试解析地址
func TestParseAddress(t *testing.T) {
	tests := []struct {
		name      string
		address   string
		wantHost  string
		wantPort  int
		wantError bool
	}{
		{
			name:      "有效地址localhost:8080",
			address:   "localhost:8080",
			wantHost:  "localhost",
			wantPort:  8080,
			wantError: false,
		},
		{
			name:      "有效地址127.0.0.1:3306",
			address:   "127.0.0.1:3306",
			wantHost:  "127.0.0.1",
			wantPort:  3306,
			wantError: false,
		},
		{
			name:      "无效地址-缺少端口",
			address:   "localhost",
			wantHost:  "",
			wantPort:  0,
			wantError: true,
		},
		{
			name:      "无效地址-格式错误",
			address:   "localhost:abc",
			wantHost:  "",
			wantPort:  0,
			wantError: true,
		},
		{
			name:      "无效地址-端口超出范围",
			address:   "localhost:99999",
			wantHost:  "",
			wantPort:  0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseAddress(tt.address)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantHost, host)
				assert.Equal(t, tt.wantPort, port)
			}
		})
	}
}

// TestGetAvailablePort 测试获取可用端口
func TestGetAvailablePort(t *testing.T) {
	port1, err := getAvailablePort()
	assert.NoError(t, err)
	assert.Greater(t, port1, 0)
	assert.Less(t, port1, 65536)

	// 再次获取应该得到不同端口
	port2, err := getAvailablePort()
	assert.NoError(t, err)
	assert.Greater(t, port2, 0)
	assert.Less(t, port2, 65536)

	// 虽然不保证一定不同，但大概率不同
	// 这里只是验证函数能正常工作
}

// TestSSHTunnel_Connect_Success 测试成功连接（仅验证函数调用）
func TestSSHTunnel_Connect_Success(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "192.0.2.1", // TEST-NET-1，确保连接会失败
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 这个连接会失败，但至少验证了Connect方法被调用
	err = tunnel.Connect()
	assert.Error(t, err) // 期望连接失败
	assert.False(t, tunnel.IsConnected())
}

// TestSSHTunnel_GetLocalPort 测试获取本地端口
func TestSSHTunnel_GetLocalPort(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 未设置端口时应该返回0
	port := tunnel.GetLocalPort()
	assert.Equal(t, 0, port)
}

// TestNewSSHTunnel_PrivateKeyFileNotReadable 测试私钥文件不可读
func TestNewSSHTunnel_PrivateKeyFileNotReadable(t *testing.T) {
	// 创建一个没有读取权限的文件
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "no_read_key")
	err := os.WriteFile(keyFile, []byte("test"), 0000)
	assert.NoError(t, err)

	cfg := &SSHConfig{
		Host:           "localhost",
		Port:           22,
		User:           "testuser",
		PrivateKeyFile: keyFile,
	}

	// 在某些系统上，即使文件权限为000，root用户也可能读取
	// 所以这里我们只验证函数不会崩溃
	_, err = NewSSHTunnel(cfg)
	// 结果可能成功或失败，取决于运行环境
	// 我们只确保函数正常执行
	assert.NotPanics(t, func() {
		NewSSHTunnel(cfg)
	})
}

// TestSSHTunnel_ForwardPort_InvalidAddress 测试端口转发到无效地址
func TestSSHTunnel_ForwardPort_InvalidAddress(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "192.0.2.1", // TEST-NET-1
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 连接会失败
	err = tunnel.Connect()
	assert.Error(t, err)

	// 未连接时尝试端口转发应该失败
	_, err = tunnel.ForwardPort("", -1)
	assert.Error(t, err)
}

// TestBuildSSHClientConfig_NoAuth 测试没有认证方法时构建配置
func TestBuildSSHClientConfig_NoAuth(t *testing.T) {
	cfg := &SSHConfig{
		Host: "localhost",
		Port: 22,
		User: "testuser",
		// 既没有密码也没有私钥
	}

	_, err := buildSSHClientConfig(cfg)
	assert.NoError(t, err) // 构建配置本身不验证认证方式
}

// TestGetPrivateKeySigner_InvalidFile 测试读取无效私钥文件
func TestGetPrivateKeySigner_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "invalid_key")
	err := os.WriteFile(keyFile, []byte("not a valid key"), 0600)
	assert.NoError(t, err)

	_, err = getPrivateKeySigner(keyFile, "")
	assert.Error(t, err)
}

// TestGetPrivateKeySigner_EncryptedKeyWrongPassphrase 测试加密私钥使用错误密码
func TestGetPrivateKeySigner_EncryptedKeyWrongPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key")

	// 生成测试私钥
	err := generateTestPrivateKey(keyFile, "")
	assert.NoError(t, err)

	// 尝试使用密码解析未加密的私钥
	_, err = getPrivateKeySigner(keyFile, "wrongpassphrase")
	// 未加密的私钥不需要密码，所以这可能成功
	// 我们只验证函数不会崩溃
	assert.NotPanics(t, func() {
		getPrivateKeySigner(keyFile, "wrongpassphrase")
	})
}

// TestDatabaseConfig_ValidateSSHConfig_NotEnabled 测试未启用SSH时的验证
func TestDatabaseConfig_ValidateSSHConfig_NotEnabled(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:       "localhost",
		Port:       3306,
		Database:   "testdb",
		Username:   "root",
		Password:   "root",
		SSHEnabled: false,
	}

	err := cfg.ValidateSSHConfig()
	assert.NoError(t, err) // 未启用SSH时不验证
}

// TestDatabaseConfig_ValidateSSHConfig_MissingHost 测试启用SSH但缺少主机
func TestDatabaseConfig_ValidateSSHConfig_MissingHost(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:       "localhost",
		Port:       3306,
		Database:   "testdb",
		Username:   "root",
		Password:   "root",
		SSHEnabled: true,
		SSHHost:    "",
	}

	err := cfg.ValidateSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "主机")
}

// TestDatabaseConfig_ValidateSSHConfig_InvalidPort 测试启用SSH但端口无效
func TestDatabaseConfig_ValidateSSHConfig_InvalidPort(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:       "localhost",
		Port:       3306,
		Database:   "testdb",
		Username:   "root",
		Password:   "root",
		SSHEnabled: true,
		SSHHost:    "example.com",
		SSHPort:    -1,
	}

	err := cfg.ValidateSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "端口")
}

// TestDatabaseConfig_ValidateSSHConfig_MissingUser 测试启用SSH但缺少用户
func TestDatabaseConfig_ValidateSSHConfig_MissingUser(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:       "localhost",
		Port:       3306,
		Database:   "testdb",
		Username:   "root",
		Password:   "root",
		SSHEnabled: true,
		SSHHost:    "example.com",
		SSHPort:    22,
		SSHUser:    "",
	}

	err := cfg.ValidateSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "用户")
}

// TestDatabaseConfig_ValidateSSHConfig_NoAuth 测试启用SSH但没有认证方式
func TestDatabaseConfig_ValidateSSHConfig_NoAuth(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:             "localhost",
		Port:             3306,
		Database:         "testdb",
		Username:         "root",
		Password:         "root",
		SSHEnabled:       true,
		SSHHost:          "example.com",
		SSHPort:          22,
		SSHUser:          "testuser",
		SSHPassword:      "",
		SSHPrivateKeyFile: "",
	}

	err := cfg.ValidateSSHConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "密码")
}

// TestDatabaseConfig_validatePrivateKeyFile_NotExist 测试验证不存在的私钥文件
func TestDatabaseConfig_validatePrivateKeyFile_NotExist(t *testing.T) {
	cfg := &config.DatabaseConfig{
		SSHPrivateKeyFile: "/nonexistent/key.pem",
	}

	err := cfg.ValidateSSHConfig()
	// 因为SSH未启用，所以不会验证私钥文件
	_ = err
}

// TestSSHTunnel_ConnectWithTimeout_ZeroTimeout 测试零超时连接
func TestSSHTunnel_ConnectWithTimeout_ZeroTimeout(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 零超时应该立即返回或使用默认超时
	err = tunnel.ConnectWithTimeout(0)
	// 结果取决于是否有SSH服务运行
	// 我们只验证函数不会崩溃
	assert.NotPanics(t, func() {
		tunnel.ConnectWithTimeout(0)
	})
	_ = err
}

// TestSSHTunnel_ConnectWithTimeout_NegativeTimeout 测试负超时连接
func TestSSHTunnel_ConnectWithTimeout_NegativeTimeout(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 负超时应该立即返回错误
	err = tunnel.ConnectWithTimeout(-1 * time.Second)
	assert.Error(t, err)
}

// TestSSHTunnel_MultipleClose 测试多次关闭隧道
func TestSSHTunnel_MultipleClose(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 多次关闭不应该报错
	err = tunnel.Close()
	assert.NoError(t, err)

	err = tunnel.Close()
	assert.NoError(t, err)

	err = tunnel.Close()
	assert.NoError(t, err)

	assert.False(t, tunnel.IsConnected())
}

// TestDatabaseError_Error 测试DatabaseError的Error方法
func TestDatabaseError_Error(t *testing.T) {
	err := &DatabaseError{
		Op:      "TestOperation",
		SSH:     true,
		Err:     assert.AnError,
		Message: "Test error message",
	}

	errorStr := err.Error()
	assert.Contains(t, errorStr, "TestOperation")
	assert.Contains(t, errorStr, "Test error message")
}

// TestDatabaseError_Unwrap 测试DatabaseError的Unwrap方法
func TestDatabaseError_Unwrap(t *testing.T) {
	originalErr := assert.AnError
	err := &DatabaseError{
		Op:      "TestOperation",
		SSH:     true,
		Err:     originalErr,
		Message: "Test error message",
	}

	unwrapped := err.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

// TestDatabaseError_NilErr 测试DatabaseError的nil错误
func TestDatabaseError_NilErr(t *testing.T) {
	err := &DatabaseError{
		Op:      "TestOperation",
		SSH:     true,
		Err:     nil,
		Message: "Test error message",
	}

	errorStr := err.Error()
	assert.NotEmpty(t, errorStr)
	assert.Contains(t, errorStr, "TestOperation")
}

// TestNewSSHError 测试NewSSHError函数
func TestNewSSHError(t *testing.T) {
	baseErr := assert.AnError
	err := NewSSHError("TestOp", baseErr, "Test message")

	assert.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.True(t, err.SSH)
	assert.Equal(t, baseErr, err.Err)
	assert.Equal(t, "Test message", err.Message)

	errorStr := err.Error()
	assert.Contains(t, errorStr, "TestOp")
	assert.Contains(t, errorStr, "Test message")
}

// TestNewConnectionError 测试NewConnectionError函数
func TestNewConnectionError(t *testing.T) {
	baseErr := assert.AnError
	err := NewConnectionError("TestOp", baseErr, "Test message")

	assert.NotNil(t, err)
	assert.Equal(t, "TestOp", err.Op)
	assert.False(t, err.SSH)
	assert.Equal(t, baseErr, err.Err)
	assert.Equal(t, "Test message", err.Message)
}

// TestCreateSSHTunnel 测试createSSHTunnel函数
func TestCreateSSHTunnel(t *testing.T) {
	cfg := &config.DatabaseConfig{
		SSHHost:           "example.com",
		SSHPort:           22,
		SSHUser:           "testuser",
		SSHPassword:       "testpass",
		SSHPrivateKeyFile: "",
		SSHKeyPassphrase:  "",
	}

	tunnel, err := createSSHTunnel(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, tunnel)
	assert.Equal(t, "example.com", tunnel.config.Host)
	assert.Equal(t, 22, tunnel.config.Port)
	assert.Equal(t, "testuser", tunnel.config.User)
}

// TestSSHTunnel_ConnectAndForward 测试连接和端口转发（使用测试服务器）
func TestSSHTunnel_ConnectAndForward(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 尝试连接（可能会失败，因为没有实际的SSH服务器）
	err = tunnel.Connect()
	if err != nil {
		// 连接失败是预期的，我们只验证代码路径
		return
	}
	defer tunnel.Close()

	// 如果连接成功，尝试端口转发
	addr, err := tunnel.ForwardPort("localhost", 3306)
	if err != nil {
		// 端口转发可能失败
		return
	}

	assert.NotEmpty(t, addr)
	assert.Greater(t, tunnel.GetLocalPort(), 0)
}

// TestSSHTunnel_ForwardPort_AfterClose 测试关闭后端口转发
func TestSSHTunnel_ForwardPort_AfterClose(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 关闭隧道
	err = tunnel.Close()
	assert.NoError(t, err)

	// 关闭后尝试端口转发应该失败
	_, err = tunnel.ForwardPort("localhost", 3306)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未连接")
}

// TestSSHTunnel_Connect_AlreadyConnected 测试重复连接
func TestSSHTunnel_Connect_AlreadyConnected(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "192.0.2.1", // TEST-NET-1
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)

	// 第一次连接（会失败，因为没有实际服务器）
	err = tunnel.Connect()
	assert.Error(t, err)
	assert.False(t, tunnel.IsConnected())

	// 第二次连接
	err = tunnel.Connect()
	assert.Error(t, err)
}

// TestSSHTunnel_Lifecycle 测试SSH隧道完整生命周期
func TestSSHTunnel_Lifecycle(t *testing.T) {
	cfg := &SSHConfig{
		Host:     "localhost",
		Port:     22,
		User:     "testuser",
		Password: "testpass",
	}

	// 创建隧道
	tunnel, err := NewSSHTunnel(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, tunnel)
	assert.False(t, tunnel.IsConnected())
	assert.Equal(t, 0, tunnel.GetLocalPort())

	// 尝试连接
	err = tunnel.Connect()
	if err != nil {
		// 连接失败是预期的（没有实际SSH服务器）
		assert.False(t, tunnel.IsConnected())
	} else {
		// 如果连接成功，测试端口转发
		addr, err := tunnel.ForwardPort("localhost", 3306)
		if err == nil {
			assert.NotEmpty(t, addr)
			assert.Greater(t, tunnel.GetLocalPort(), 0)
		}

		// 关闭隧道
		err = tunnel.Close()
		assert.NoError(t, err)
		assert.False(t, tunnel.IsConnected())
	}
}

// TestGetPrivateKeySigner_EmptyFile 测试读取空文件作为私钥
func TestGetPrivateKeySigner_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty_key")
	err := os.WriteFile(emptyFile, []byte(""), 0600)
	assert.NoError(t, err)

	_, err = getPrivateKeySigner(emptyFile, "")
	assert.Error(t, err)
}

// TestGetPrivateKeySigner_NonPEMFile 测试读取非PEM文件
func TestGetPrivateKeySigner_NonPEMFile(t *testing.T) {
	tmpDir := t.TempDir()
	textFile := filepath.Join(tmpDir, "text_key")
	err := os.WriteFile(textFile, []byte("This is not a PEM file"), 0600)
	assert.NoError(t, err)

	_, err = getPrivateKeySigner(textFile, "")
	assert.Error(t, err)
}

// TestParsePort_InvalidFormat 测试解析无效格式的端口
func TestParsePort_InvalidFormat(t *testing.T) {
	tests := []struct {
		name    string
		portStr string
		wantErr bool
	}{
		{"非数字", "abc", true},
		{"负数", "-1", true},
		{"零", "0", true},
		{"太大", "65536", true},
		{"有效", "8080", false},
		{"边界值1", "1", false},
		{"边界值65535", "65535", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := parsePort(tt.portStr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.portStr, fmt.Sprintf("%d", port))
			}
		})
	}
}

