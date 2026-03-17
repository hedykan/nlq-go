package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Security SecurityConfig `yaml:"security"`
	Server   ServerConfig   `yaml:"server"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Readonly bool   `yaml:"readonly"`
}

// LLMConfig 大语言模型配置
type LLMConfig struct {
	Provider   string        `yaml:"provider"`
	Model      string        `yaml:"model"`
	APIKey     string        `yaml:"api_key"`
	BaseURL    string        `yaml:"base_url"`
	MaxRetries int           `yaml:"max_retries"`
	Timeout    time.Duration `yaml:"timeout"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	Mode            string `yaml:"mode"`
	CheckComments   bool   `yaml:"check_comments"`
	CheckSemicolon  bool   `yaml:"check_semicolon"`
	AllowedPrefixes []string
	BlockedKeywords []string
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host          string        `yaml:"host"`
	Port          int           `yaml:"port"`
	QueryTimeout  time.Duration `yaml:"query_timeout"`
	ReadTimeout   time.Duration `yaml:"read_timeout"`
	WriteTimeout  time.Duration `yaml:"write_timeout"`
	EnableCORS    bool          `yaml:"enable_cors"`
}

// LoadFromFile 从YAML文件加载配置
func LoadFromFile(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析YAML配置文件失败: %w", err)
	}

	// 设置默认值
	cfg.SetDefaults()

	return &cfg, nil
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() (*Config, error) {
	cfg := &Config{}
	cfg.SetDefaults()

	// 从环境变量覆盖配置
	if host := os.Getenv("DATABASE_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if portStr := os.Getenv("DATABASE_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Database.Port = port
		}
	}
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}

	return cfg, nil
}

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	// 数据库默认值
	if c.Database.Driver == "" {
		c.Database.Driver = "mysql"
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 3306
	}
	// 默认只读模式
	c.Database.Readonly = true

	// LLM默认值
	if c.LLM.Provider == "" {
		c.LLM.Provider = "zhipuai"
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "glm-4-plus"
	}
	if c.LLM.MaxRetries == 0 {
		c.LLM.MaxRetries = 3
	}
	if c.LLM.Timeout == 0 {
		c.LLM.Timeout = 30 * time.Second
	}

	// 安全默认值
	if c.Security.Mode == "" {
		c.Security.Mode = "strict"
	}
	// 默认启用安全检查
	c.Security.CheckComments = true
	c.Security.CheckSemicolon = true

	// 服务器默认值
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.QueryTimeout == 0 {
		c.Server.QueryTimeout = 10 * time.Second
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// 验证数据库配置
	if c.Database.Database == "" {
		return errors.New("数据库名称不能为空")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return errors.New("端口号必须在1-65535之间")
	}
	if c.Database.Username == "" {
		return errors.New("数据库用户名不能为空")
	}

	// 验证LLM配置
	if c.LLM.APIKey == "" {
		return errors.New("LLM API密钥不能为空")
	}
	if c.LLM.BaseURL == "" {
		return errors.New("LLM BaseURL不能为空")
	}

	return nil
}
