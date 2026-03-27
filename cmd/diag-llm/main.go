package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/llm"
	"github.com/channelwill/nlq/pkg/utils"
)

func main() {
	fmt.Println("🔍 NLQ LLM诊断工具")
	fmt.Println("════════════════════════════════════════════════════════════════")

	// 设置日志级别为DEBUG以显示详细信息
	utils.SetLogLevel(utils.DEBUG)

	// 加载配置
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 配置文件加载成功")
	fmt.Printf("📋 LLM配置:\n")
	fmt.Printf("   - Provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("   - Model: %s\n", cfg.LLM.Model)
	fmt.Printf("   - Base URL: %s\n", cfg.LLM.BaseURL)
	fmt.Printf("   - API Key: %s...%s\n",
		cfg.LLM.APIKey[:min(5, len(cfg.LLM.APIKey))],
		cfg.LLM.APIKey[max(0, len(cfg.LLM.APIKey)-5):])
	fmt.Printf("   - Timeout: %s\n", cfg.LLM.Timeout)
	fmt.Println()

	// 检查API Key
	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" || cfg.LLM.APIKey == "${GLM_API_KEY}" {
		fmt.Println("❌ API Key未配置！")
		fmt.Println("请设置环境变量 GLM_API_KEY 或在配置文件中设置有效的API Key")
		os.Exit(1)
	}

	// 创建LLM客户端
	fmt.Println("🔧 创建GLM客户端...")
	client, err := llm.NewGLMClient(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
	if err != nil {
		fmt.Printf("❌ GLM客户端创建失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ GLM客户端创建成功")
	fmt.Println()

	// 测试API连接
	fmt.Println("🌐 测试API连接...")
	testAPIConnection(client, cfg.LLM.APIKey)
	fmt.Println()

	// 测试SQL生成
	fmt.Println("🤖 测试SQL生成...")
	testSQLGeneration(client)
	fmt.Println()

	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("✅ 诊断完成！")
}

// testAPIConnection 测试API连接
func testAPIConnection(client *llm.GLMClient, apiKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 构建简单的测试请求
	testRequest := map[string]any{
		"model": client.GetModel(),
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个测试助手"},
			{"role": "user", "content": "请回复'测试成功'"},
		},
		"max_tokens": 10,
	}

	reqBody, _ := json.Marshal(testRequest)
	endpoint := fmt.Sprintf("%s/chat/completions", "https://open.bigmodel.cn/api/coding/paas/v4")

	fmt.Printf("🔍 发送测试请求到: %s\n", endpoint)
	fmt.Printf("🔍 使用API Key: %s...%s\n",
		apiKey[:min(8, len(apiKey))],
		apiKey[max(0, len(apiKey)-4):])

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint,
		nil)
	if err != nil {
		fmt.Printf("❌ 创建请求失败: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	fmt.Printf("🔍 请求头: Content-Type=%s\n", req.Header.Get("Content-Type"))
	fmt.Printf("🔍 请求体: %s\n", string(reqBody))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("🔍 响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("🔍 响应头:\n")
	for key, values := range resp.Header {
		fmt.Printf("   %s: %v\n", key, values)
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("🔍 响应体: %s\n", string(body))

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ API连接成功！")
	} else {
		fmt.Printf("❌ API返回错误状态码: %d\n", resp.StatusCode)
	}
}

// testSQLGeneration 测试SQL生成
func testSQLGeneration(client *llm.GLMClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 简单的测试Schema
	schema := `数据库Schema:

表: users
  - id: INT NOT NULL // 用户ID
  - name: VARCHAR(255) NULL // 用户名
  - email: VARCHAR(255) NULL // 邮箱
  - created_at: TIMESTAMP NULL // 创建时间

`

	question := "查询所有用户"

	fmt.Printf("🔍 测试问题: %s\n", question)
	fmt.Printf("🔍 Schema: %d字符\n", len(schema))

	sql, err := client.GenerateSQL(ctx, schema, question)
	if err != nil {
		fmt.Printf("❌ SQL生成失败: %v\n", err)
		return
	}

	fmt.Printf("✅ SQL生成成功: %s\n", sql)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
