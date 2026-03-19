package llm

import (
	"context"
	"testing"
	"time"
)

// TestGLMClient_NewGLMClient 测试创建GLM客户端
func TestGLMClient_NewGLMClient(t *testing.T) {
	client := NewGLMClient("test-api-key", "https://api.example.com", "glm-4-plus")
	if client == nil {
		t.Fatal("期望返回非nil的客户端")
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("期望apiKey为test-api-key，实际为%s", client.apiKey)
	}
}

// TestGLMClient_GenerateSQL 测试SQL生成（需要真实API Key）
func TestGLMClient_GenerateSQL(t *testing.T) {
	// 使用testutil创建测试客户端（从config读取配置）
	client := CreateTestClient(t)

	schema := "数据库Schema:\n表: users\n  - id: int\n  - name: varchar\n  - email: varchar"
	question := "查询所有用户的数量"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sql, err := client.GenerateSQL(ctx, schema, question)
	if err != nil {
		t.Logf("生成SQL失败（可能是API Key无效）: %v", err)
		return
	}

	if sql == "" {
		t.Error("期望返回非空的SQL")
	}

	t.Logf("生成的SQL: %s", sql)
}

// TestGLMClient_GenerateSQL_WithRealAPI 测试真实API调用（手动测试）
func TestGLMClient_GenerateSQL_WithRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要真实API的测试")
	}

	// 使用testutil创建测试客户端（从config读取配置）
	client := CreateTestClient(t)

	schema := `数据库Schema:

表: boom_user
  - id: bigint
  - name: varchar
  - email: varchar
  - created_at: bigint`

	tests := []struct {
		name     string
		question string
	}{
		{
			name:     "查询用户数量",
			question: "boom_user表有多少条数据？",
		},
		{
			name:     "查询特定用户",
			question: "查询名字为'测试'的用户",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			sql, err := client.GenerateSQL(ctx, schema, tt.question)
			if err != nil {
				t.Logf("API调用失败: %v", err)
				return
			}

			if sql == "" {
				t.Error("期望返回非空的SQL")
				return
			}

			// 验证SQL是否安全
			if !ValidateSQLQuery(sql) {
				t.Errorf("生成的SQL不安全: %s", sql)
			}

			t.Logf("问题: %s", tt.question)
			t.Logf("SQL: %s", sql)
		})
	}
}

// TestBuildChatRequest 测试构建聊天请求
func TestBuildChatRequest(t *testing.T) {
	systemPrompt := "你是一个SQL专家"
	userPrompt := "查询所有用户"

	messages := BuildChatMessages(systemPrompt, userPrompt)

	if len(messages) != 2 {
		t.Errorf("期望2条消息，实际为%d", len(messages))
	}

	if messages[0]["role"] != "system" {
		t.Errorf("期望第一条消息的role为system")
	}

	if messages[1]["role"] != "user" {
		t.Errorf("期望第二条消息的role为user")
	}
}

// TestExtractSQLFromLLMResponse 测试从LLM响应中提取SQL
func TestExtractSQLFromLLMResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{
			name:     "纯SQL",
			response: "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "带解释的响应",
			response: "根据您的问题，生成的SQL如下：\n```sql\nSELECT * FROM users\n```\n这个查询会返回所有用户。",
			expected: "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSQLFromResponse(tt.response)
			if err != nil {
				t.Errorf("解析失败: %v", err)
			}
			if result != tt.expected {
				t.Errorf("结果不匹配\n期望: %s\n实际: %s", tt.expected, result)
			}
		})
	}
}
