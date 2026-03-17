package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/llm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 数据库连接配置
	dsn := "root:root@tcp(127.0.0.1:3306)/loloyal?charset=utf8mb4&parseTime=True&loc=Local"

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("数据库连接失败: %v\n", err)
		return
	}

	// 创建Schema解析器
	parser := database.NewSchemaParser(db)

	// 创建示例仓库
	exampleRepo := llm.NewExampleRepository("./data")

	fmt.Println("=== NLQ两步法增强测试 ===\n")

	// 测试1：Few-shot示例检索
	fmt.Println("测试1：Few-shot示例检索")
	testExamples := []string{
		"查询100个最早的用户的username",
		"统计每个VIP等级的用户数量",
		"查询VIP用户的数量",
	}

	for _, question := range testExamples {
		examples := exampleRepo.RetrieveExamples(question, 2)
		fmt.Printf("\n问题: %s\n", question)
		fmt.Printf("检索到 %d 个示例\n", len(examples))
		for i, example := range examples {
			fmt.Printf("  示例%d: %s (类型: %s)\n", i+1, example.Question, example.Type)
		}
	}

	// 测试2：增强的表摘要
	fmt.Println("\n=== 测试2：增强的表摘要 ===")
	tables, err := parser.GetTableSummariesEnhanced()
	if err != nil {
		fmt.Printf("获取表摘要失败: %v\n", err)
		return
	}

	for _, table := range tables {
		if len(table.KeyColumns) > 0 {
			fmt.Printf("表 %s: 关键字段 = %v\n", table.Name, table.KeyColumns)
		}
	}

	// 测试3：字段别名映射
	fmt.Println("\n=== 测试3：字段别名映射 ===")
	_ = handler.NewTableSelector(nil)
	fmt.Println("字段别名映射已初始化")

	// 测试4：两阶段查询（需要Mock LLM）
	fmt.Println("\n=== 测试4：两阶段查询 ===")
	mockLLM := &MockGLMClient{
		responses: make(map[string]string),
	}

	// 设置Mock响应
	mockLLM.SetResponse("100个最早的用户的shop_name", `{"primary_tables": ["boom_user"], "secondary_tables": [], "reasoning": "查询用户信息，需要boom_user表"}`)

	// 创建两阶段处理器
	twoPhaseHandler := handler.NewTwoPhaseQueryHandler(parser, db, mockLLM)

	// 测试查询
	testQueries := []string{
		"100个最早的用户的shop_name",
		"查询VIP用户的数量",
	}

	ctx := context.Background()
	for _, query := range testQueries {
		fmt.Printf("\n查询: %s\n", query)

		// 注意：由于我们使用Mock LLM，这里只会返回基本的选择结果
		// 在实际使用中，需要配置真实的GLM API Key
		result, err := twoPhaseHandler.Handle(ctx, query)
		if err != nil {
			fmt.Printf("  错误: %v\n", err)
			continue
		}

		// 打印结果
		resultJSON, _ := json.MarshalIndent(result, "  ", "  ")
		fmt.Printf("  结果: %s\n", string(resultJSON))
	}

	fmt.Println("\n=== 测试完成 ===")
}

// MockGLMClient Mock LLM客户端
type MockGLMClient struct {
	responses map[string]string
}

func (m *MockGLMClient) SetResponse(question, response string) {
	m.responses[question] = response
}

func (m *MockGLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	// 返回预设的响应
	if response, ok := m.responses[question]; ok {
		return response, nil
	}
	return "SELECT * FROM boom_user LIMIT 100", nil
}

func (m *MockGLMClient) IsAvailable() bool {
	return true
}

func (m *MockGLMClient) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// 返回预设的响应
	if response, ok := m.responses[userPrompt]; ok {
		return response, nil
	}
	return `{"primary_tables": ["boom_user"], "secondary_tables": [], "reasoning": "默认选择"}`, nil
}
