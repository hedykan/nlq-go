package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Database     DatabaseConfig   `mapstructure:"database"`
	LLM          LLMConfig        `mapstructure:"llm"`
	Security     SecurityConfig   `mapstructure:"security"`
	Server       ServerConfig     `mapstructure:"server"`
	FieldAliases FieldAliasConfig `mapstructure:"field_aliases"` // 字段别名配置
	Query        QueryConfig      `mapstructure:"query"`         // 查询处理器配置
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

	// SSH隧道配置（新增）
	SSHEnabled        bool   `mapstructure:"ssh_enabled"`          // 是否启用SSH隧道
	SSHHost           string `mapstructure:"ssh_host"`             // SSH服务器地址
	SSHPort           int    `mapstructure:"ssh_port"`             // SSH服务器端口（默认22）
	SSHUser           string `mapstructure:"ssh_user"`             // SSH用户名
	SSHPassword       string `mapstructure:"ssh_password"`         // SSH密码（与私钥二选一）
	SSHPrivateKeyFile string `mapstructure:"ssh_private_key_file"` // SSH私钥文件路径
	SSHKeyPassphrase  string `mapstructure:"ssh_key_passphrase"`   // 私钥密码短语
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
	MaxTokens    int           `mapstructure:"max_tokens"`
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
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	QueryTimeout time.Duration `mapstructure:"query_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	EnableCORS   bool          `mapstructure:"enable_cors"`
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

// QueryConfig 查询处理器配置
type QueryConfig struct {
	Agent AgentConfig `mapstructure:"agent"` // Agent模式配置
}

// AgentConfig Agent查询处理器配置
type AgentConfig struct {
	MaxSelfCorrect int  `mapstructure:"max_self_correct"` // SQL自检修正最大次数，默认3
	MaxTurns       int  `mapstructure:"max_turns"`        // 最大总轮次，默认5
	Verbose        bool `mapstructure:"verbose"`         // 返回中间推理过程，默认false
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
	v.SetDefault("database.read_timeout", 30*time.Second)    // 默认读取超时30秒
	v.SetDefault("database.connect_timeout", 10*time.Second) // 默认连接超时10秒

	// LLM默认值
	v.SetDefault("llm.provider", "zhipuai")
	v.SetDefault("llm.model", "glm-4.7")
	v.SetDefault("llm.default_model", "glm-4.7")
	v.SetDefault("llm.max_retries", 3)
	v.SetDefault("llm.timeout", 90*time.Second) // GLM-4.7响应较慢
	v.SetDefault("llm.temperature", 0.0)
	v.SetDefault("llm.max_tokens", 2048)

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

	// 查询处理器默认值
	v.SetDefault("query.agent.max_self_correct", 3) // 默认自检修正3次
	v.SetDefault("query.agent.max_turns", 5)        // 默认最大5轮
	v.SetDefault("query.agent.verbose", false)       // 默认不返回中间过程
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
	if c.LLM.Temperature == 0 {
		c.LLM.Temperature = 0.0
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = 2048
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

// ValidateSSHConfig 验证SSH配置的有效性
func (c *DatabaseConfig) ValidateSSHConfig() error {
	// 如果未启用SSH隧道，直接返回
	if !c.SSHEnabled {
		return nil
	}

	// 验证SSH配置完整性
	if c.SSHHost == "" {
		return fmt.Errorf("SSH隧道已启用，但SSH主机地址未配置")
	}

	if c.SSHPort <= 0 || c.SSHPort > 65535 {
		return fmt.Errorf("SSH端口必须在1-65535之间")
	}

	if c.SSHUser == "" {
		return fmt.Errorf("SSH隧道已启用，但SSH用户名未配置")
	}

	// 验证认证方式
	if c.SSHPassword == "" && c.SSHPrivateKeyFile == "" {
		return fmt.Errorf("SSH隧道已启用，必须配置SSH密码或私钥文件")
	}

	// 如果使用私钥认证，验证私钥文件
	if c.SSHPrivateKeyFile != "" {
		if err := c.validatePrivateKeyFile(); err != nil {
			return err
		}
	}

	return nil
}

// validatePrivateKeyFile 验证私钥文件的有效性
func (c *DatabaseConfig) validatePrivateKeyFile() error {
	// 检查文件是否存在
	if _, err := os.Stat(c.SSHPrivateKeyFile); os.IsNotExist(err) {
		return fmt.Errorf("SSH私钥文件不存在: %s", c.SSHPrivateKeyFile)
	}

	// 检查文件权限（私钥文件应该只有所有者可读）
	info, err := os.Stat(c.SSHPrivateKeyFile)
	if err != nil {
		return fmt.Errorf("无法读取SSH私钥文件: %w", err)
	}

	// 在Unix系统上检查文件权限
	// 私钥文件应该权限为600（只有所有者可读写）
	if info.Mode().Perm()&0077 != 0 {
		// 警告：私钥文件权限过于宽松
		// 这里返回警告而不是错误，允许用户继续
		fmt.Printf("⚠️  警告: SSH私钥文件权限过于宽松，建议设置为600: %s\n", c.SSHPrivateKeyFile)
	}

	return nil
}
