package config

import (
	"os"
	"testing"
	"time"
)

// TestConfig_LoadFromFile 测试从文件加载配置
func TestConfig_LoadFromFile(t *testing.T) {
	// 创建临时配置文件
	configContent := `
database:
  driver: mysql
  host: localhost
  port: 3306
  database: loloyal
  username: root
  password: root
  readonly: true

llm:
  provider: zhipuai
  model: glm-4-plus
  api_key: test-api-key
  base_url: https://open.bigmodel.cn/api/paas/v4/
  max_retries: 3
  timeout: 30s

security:
  mode: strict
  check_comments: true
  check_semicolon: true

server:
  port: 8080
  query_timeout: 10s
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("写入配置内容失败: %v", err)
	}
	tmpFile.Close()

	// 测试加载配置
	cfg, err := LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置文件失败: %v", err)
	}

	// 验证数据库配置
	if cfg.Database.Driver != "mysql" {
		t.Errorf("期望 Driver 为 mysql, 实际为 %s", cfg.Database.Driver)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("期望 Host 为 localhost, 实际为 %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("期望 Port 为 3306, 实际为 %d", cfg.Database.Port)
	}
	if cfg.Database.Database != "loloyal" {
		t.Errorf("期望 Database 为 loloyal, 实际为 %s", cfg.Database.Database)
	}
	if cfg.Database.Username != "root" {
		t.Errorf("期望 Username 为 root, 实际为 %s", cfg.Database.Username)
	}
	if cfg.Database.Password != "root" {
		t.Errorf("期望 Password 为 root, 实际为 %s", cfg.Database.Password)
	}
	if !cfg.Database.Readonly {
		t.Errorf("期望 Readonly 为 true")
	}

	// 验证LLM配置
	if cfg.LLM.Provider != "zhipuai" {
		t.Errorf("期望 Provider 为 zhipuai, 实际为 %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "glm-4-plus" {
		t.Errorf("期望 Model 为 glm-4-plus, 实际为 %s", cfg.LLM.Model)
	}
	if cfg.LLM.APIKey != "test-api-key" {
		t.Errorf("期望 APIKey 为 test-api-key, 实际为 %s", cfg.LLM.APIKey)
	}
	if cfg.LLM.BaseURL != "https://open.bigmodel.cn/api/paas/v4/" {
		t.Errorf("期望 BaseURL 为 https://open.bigmodel.cn/api/paas/v4/, 实际为 %s", cfg.LLM.BaseURL)
	}
	if cfg.LLM.MaxRetries != 3 {
		t.Errorf("期望 MaxRetries 为 3, 实际为 %d", cfg.LLM.MaxRetries)
	}
	if cfg.LLM.Timeout != 30*time.Second {
		t.Errorf("期望 Timeout 为 30s, 实际为 %v", cfg.LLM.Timeout)
	}

	// 验证安全配置
	if cfg.Security.Mode != "strict" {
		t.Errorf("期望 Mode 为 strict, 实际为 %s", cfg.Security.Mode)
	}
	if !cfg.Security.CheckComments {
		t.Errorf("期望 CheckComments 为 true")
	}
	if !cfg.Security.CheckSemicolon {
		t.Errorf("期望 CheckSemicolon 为 true")
	}

	// 验证服务器配置
	if cfg.Server.Port != 8080 {
		t.Errorf("期望 Port 为 8080, 实际为 %d", cfg.Server.Port)
	}
	if cfg.Server.QueryTimeout != 10*time.Second {
		t.Errorf("期望 QueryTimeout 为 10s, 实际为 %v", cfg.Server.QueryTimeout)
	}
}

// TestConfig_LoadFromFile_NotFound 测试加载不存在的配置文件
func TestConfig_LoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/non/existent/config.yaml")
	if err == nil {
		t.Error("期望返回错误，但返回了 nil")
	}
}

// TestConfig_LoadFromFile_InvalidYAML 测试加载无效的YAML格式
func TestConfig_LoadFromFile_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid: yaml: content: ["); err != nil {
		t.Fatalf("写入配置内容失败: %v", err)
	}
	tmpFile.Close()

	_, err = LoadFromFile(tmpFile.Name())
	if err == nil {
		t.Error("期望返回错误，但返回了 nil")
	}
}

// TestConfig_LoadFromEnv 测试从环境变量加载配置
func TestConfig_LoadFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("DATABASE_HOST", "env-host")
	os.Setenv("DATABASE_PORT", "3307")
	os.Setenv("LLM_API_KEY", "env-api-key")
	defer func() {
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
		os.Unsetenv("LLM_API_KEY")
	}()

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("从环境变量加载配置失败: %v", err)
	}

	// 验证环境变量覆盖
	if cfg.Database.Host != "env-host" {
		t.Errorf("期望 Host 为 env-host, 实际为 %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 3307 {
		t.Errorf("期望 Port 为 3307, 实际为 %d", cfg.Database.Port)
	}
	if cfg.LLM.APIKey != "env-api-key" {
		t.Errorf("期望 APIKey 为 env-api-key, 实际为 %s", cfg.LLM.APIKey)
	}
}

// TestConfig_DefaultValues 测试默认值
func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{}

	// 设置默认值
	cfg.SetDefaults()

	if cfg.Database.Driver != "mysql" {
		t.Errorf("期望默认 Driver 为 mysql, 实际为 %s", cfg.Database.Driver)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("期望默认 Host 为 localhost, 实际为 %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("期望默认 Port 为 3306, 实际为 %d", cfg.Database.Port)
	}
	if cfg.Database.Readonly != true {
		t.Errorf("期望默认 Readonly 为 true")
	}

	if cfg.LLM.Provider != "zhipuai" {
		t.Errorf("期望默认 Provider 为 zhipuai, 实际为 %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "glm-4-plus" {
		t.Errorf("期望默认 Model 为 glm-4-plus, 实际为 %s", cfg.LLM.Model)
	}
	if cfg.LLM.MaxRetries != 3 {
		t.Errorf("期望默认 MaxRetries 为 3, 实际为 %d", cfg.LLM.MaxRetries)
	}

	if cfg.Security.Mode != "strict" {
		t.Errorf("期望默认 Mode 为 strict, 实际为 %s", cfg.Security.Mode)
	}
	if cfg.Security.CheckComments != true {
		t.Errorf("期望默认 CheckComments 为 true")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("期望默认 Port 为 8080, 实际为 %d", cfg.Server.Port)
	}
}

// TestConfig_Validate 测试配置验证
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "有效配置",
			config: &Config{
				Database: DatabaseConfig{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     3306,
					Database: "testdb",
					Username: "root",
					Password: "password",
				},
				LLM: LLMConfig{
					APIKey:  "test-key",
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "缺少数据库名称",
			config: &Config{
				Database: DatabaseConfig{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     3306,
					Username: "root",
					Password: "password",
				},
			},
			wantErr: true,
		},
		{
			name: "缺少API密钥",
			config: &Config{
				Database: DatabaseConfig{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     3306,
					Database: "testdb",
					Username: "root",
					Password: "password",
				},
				LLM: LLMConfig{
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "无效的端口号",
			config: &Config{
				Database: DatabaseConfig{
					Driver:   "mysql",
					Host:     "localhost",
					Port:     -1,
					Database: "testdb",
					Username: "root",
					Password: "password",
				},
				LLM: LLMConfig{
					APIKey:  "test-key",
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
