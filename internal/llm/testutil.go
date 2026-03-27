package llm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/channelwill/nlq/internal/config"
)

// LoadTestConfig 加载测试配置
// 优先从config/config.yaml读取，如果失败则使用环境变量
func LoadTestConfig(t *testing.T) *config.Config {
	// 尝试从config/config.yaml读取（相对于internal/llm目录）
	configPath := filepath.Join("..", "..", "config", "config.yaml")
	cfg, err := config.LoadConfig(configPath)

	// 如果配置文件加载失败或API Key无效，使用环境变量
	if err != nil || cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" {
		cfg = &config.Config{}
		cfg.LLM.APIKey = os.Getenv("GLM_API_KEY")
		cfg.LLM.BaseURL = getEnvWithDefault("LLM_BASE_URL", "https://open.bigmodel.cn/api/coding/paas/v4/")
		cfg.LLM.Model = getEnvWithDefault("LLM_MODEL", "glm-4.7")
	}

	// 验证必需配置
	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" {
		t.Skip("需要设置GLM_API_KEY环境变量或配置文件")
	}

	// 确保BaseURL不为空
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = "https://open.bigmodel.cn/api/coding/paas/v4/"
	}

	// 确保Model不为空
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "glm-4.7"
	}

	return cfg
}

// CreateTestClient 创建测试客户端（从config读取配置）
// 用于需要真实API调用的测试
func CreateTestClient(t *testing.T) *GLMClient {
	cfg := LoadTestConfig(t)
	client, err := NewGLMClient(
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		cfg.LLM.Model,
	)
	if err != nil {
		t.Skipf("创建测试客户端失败: %v", err)
		return nil
	}
	return client
}

// CreateMockTestClient 创建Mock测试客户端
// 用于不需要真实API调用的单元测试
func CreateMockTestClient() *GLMClient {
	client, err := NewGLMClient(
		"test-api-key",
		"https://api.example.com",
		"glm-4-plus",
	)
	if err != nil {
		return nil
	}
	return client
}

// getEnvWithDefault 获取环境变量，支持默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
