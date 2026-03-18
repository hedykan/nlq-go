package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Database     DatabaseConfig     `mapstructure:"database"`
	LLM          LLMConfig          `mapstructure:"llm"`
	Security     SecurityConfig     `mapstructure:"security"`
	Server       ServerConfig       `mapstructure:"server"`
	FieldAliases FieldAliasConfig   `mapstructure:"field_aliases"` // 字段别名配置
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver         string        `mapstructure:"driver"`
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	Database       string        `mapstructure:"database"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	Readonly       bool          `mapstructure:"readonly"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`    // 读取超时
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"` // 连接超时
}

// LLMConfig 大语言模型配置
type LLMConfig struct {
	Provider     string        `mapstructure:"provider"`
	Model        string        `mapstructure:"model"`
	DefaultModel string        `mapstructure:"default_model"` // 默认模型（用于兼容）
	APIKey       string        `mapstructure:"api_key"`
	BaseURL      string        `mapstructure:"base_url"`
	MaxRetries   int           `mapstructure:"max_retries"`
	Timeout      time.Duration `mapstructure:"timeout"`
	Temperature  float64       `mapstructure:"temperature"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	Mode            string   `mapstructure:"mode"`
	CheckComments   bool     `mapstructure:"check_comments"`
	CheckSemicolon  bool     `mapstructure:"check_semicolon"`
	AllowedPrefixes []string `mapstructure:"allowed_prefixes"`
	BlockedKeywords []string `mapstructure:"blocked_keywords"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host          string        `mapstructure:"host"`
	Port          int           `mapstructure:"port"`
	QueryTimeout  time.Duration `mapstructure:"query_timeout"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	EnableCORS    bool          `mapstructure:"enable_cors"`
}

// FieldAliasConfig 字段别名配置
type FieldAliasConfig struct {
	Name     []string `mapstructure:"name"`     // 名称相关字段别名
	User     []string `mapstructure:"user"`     // 用户相关字段别名
	Customer []string `mapstructure:"customer"` // 客户相关字段别名
	Email    []string `mapstructure:"email"`    // 邮箱相关字段别名
	Phone    []string `mapstructure:"phone"`    // 电话相关字段别名
	Time     []string `mapstructure:"time"`     // 时间相关字段别名
	Status   []string `mapstructure:"status"`   // 状态相关字段别名
	Price    []string `mapstructure:"price"`    // 价格相关字段别名
}

// LoadConfig 使用viper加载配置
// 支持从配置文件、环境变量、命令行参数加载
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// 1. 设置默认值
	setDefaults(v)

	// 2. 配置文件读取
	if configPath != "" {
		// 显式指定配置文件路径
		v.SetConfigFile(configPath)
	} else {
		// 自动搜索配置文件
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// 尝试读取配置文件（如果文件不存在也不会报错）
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用默认值和环境变量
			fmt.Println("⚠️  配置文件未找到，将使用默认值和环境变量")
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 3. 环境变量支持
	// 自动将环境变量映射到配置字段
	// 例如: DATABASE_HOST -> database.host, LLM_API_KEY -> llm.api_key
	v.SetEnvPrefix("NLQ") // 环境变量前缀：NLQ_DATABASE_HOST
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 4. 手动设置环境变量映射（向后兼容）
	bindEnvVars(v)

	// 5. 解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 6. 设置默认值（确保未配置的字段有合理默认值）
	cfg.SetDefaults()

	return &cfg, nil
}

// LoadFromFile 从指定文件加载配置（向后兼容）
func LoadFromFile(filepath string) (*Config, error) {
	return LoadConfig(filepath)
}

// LoadFromEnv 从环境变量加载配置（向后兼容）
func LoadFromEnv() (*Config, error) {
	return LoadConfig("")
}

// setDefaults 设置viper默认值
func setDefaults(v *viper.Viper) {
	// 数据库默认值
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.readonly", true)
	v.SetDefault("database.read_timeout", 30*time.Second)   // 默认读取超时30秒
	v.SetDefault("database.connect_timeout", 10*time.Second) // 默认连接超时10秒

	// LLM默认值
	v.SetDefault("llm.provider", "zhipuai")
	v.SetDefault("llm.model", "glm-4.7")
	v.SetDefault("llm.default_model", "glm-4.7")
	v.SetDefault("llm.max_retries", 3)
	v.SetDefault("llm.timeout", 90*time.Second) // GLM-4.7响应较慢
	v.SetDefault("llm.temperature", 0.1)

	// 安全默认值
	v.SetDefault("security.mode", "strict")
	v.SetDefault("security.check_comments", true)
	v.SetDefault("security.check_semicolon", true)

	// 服务器默认值
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.query_timeout", 10*time.Second)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.enable_cors", true)
}

// bindEnvVars 绑定环境变量（向后兼容，支持无前缀的环境变量）
func bindEnvVars(v *viper.Viper) {
	// 数据库环境变量（向后兼容）
	v.BindEnv("database.host", "DATABASE_HOST")
	v.BindEnv("database.port", "DATABASE_PORT")
	v.BindEnv("database.database", "DATABASE_NAME")
	v.BindEnv("database.username", "DATABASE_USER")
	v.BindEnv("database.password", "DATABASE_PASSWORD")

	// LLM环境变量（向后兼容）
	v.BindEnv("llm.api_key", "LLM_API_KEY")
	v.BindEnv("llm.model", "LLM_MODEL")
	v.BindEnv("llm.base_url", "LLM_BASE_URL")

	// 服务器环境变量
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.host", "SERVER_HOST")
}

// SetDefaults 设置默认值（保证向后兼容）
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
	if c.Database.ReadTimeout == 0 {
		c.Database.ReadTimeout = 30 * time.Second
	}
	if c.Database.ConnectTimeout == 0 {
		c.Database.ConnectTimeout = 10 * time.Second
	}

	// LLM默认值
	if c.LLM.Provider == "" {
		c.LLM.Provider = "zhipuai"
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "glm-4.7"
	}
	if c.LLM.DefaultModel == "" {
		c.LLM.DefaultModel = "glm-4.7"
	}
	if c.LLM.MaxRetries == 0 {
		c.LLM.MaxRetries = 3
	}
	if c.LLM.Timeout == 0 {
		c.LLM.Timeout = 90 * time.Second // GLM-4.7响应较慢
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
		return fmt.Errorf("数据库名称不能为空")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("数据库用户名不能为空")
	}

	// 验证LLM配置
	if c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API密钥不能为空")
	}
	if c.LLM.BaseURL == "" {
		return fmt.Errorf("LLM BaseURL不能为空")
	}

	return nil
}

// GetDSN 获取数据库连接字符串
func (c *Config) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
	)
}
