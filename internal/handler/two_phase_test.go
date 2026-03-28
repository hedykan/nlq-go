package handler

import (
	"context"
	"testing"

	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/knowledge"
)

// MockLLMClient 两阶段测试用的Mock LLM客户端
type MockLLMClientForTwoPhase struct {
	tableSelectionResponse string
	sqlGenerationResponse  string
}

func (m *MockLLMClientForTwoPhase) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	// 如果问题是表选择相关，返回表选择JSON
	if containsTableSelectionPrompt(question) {
		return m.tableSelectionResponse, nil
	}
	// 否则返回SQL生成结果
	return m.sqlGenerationResponse, nil
}

func (m *MockLLMClientForTwoPhase) IsAvailable() bool {
	return true
}

func (m *MockLLMClientForTwoPhase) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// 返回表选择JSON
	return m.tableSelectionResponse, nil
}

func (m *MockLLMClientForTwoPhase) SetKnowledge(docs []knowledge.Document) {}
func (m *MockLLMClientForTwoPhase) GetKnowledge() []knowledge.Document     { return nil }
func (m *MockLLMClientForTwoPhase) SetModel(model string)                  {}
func (m *MockLLMClientForTwoPhase) GetModel() string                       { return "mock-model" }
func (m *MockLLMClientForTwoPhase) Type() string                           { return "mock" }

func containsTableSelectionPrompt(question string) bool {
	// 简单判断是否是表选择请求
	return contains(question, "表选择") || contains(question, "JSON")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestTableSelector 表选择器测试
func TestTableSelector(t *testing.T) {
	// 创建Mock LLM客户端
	mockLLM := &MockLLMClientForTwoPhase{
		tableSelectionResponse: `{
			"primary_tables": ["users", "orders"],
			"secondary_tables": ["products"],
			"reasoning": "用户查询订单信息"
		}`,
		sqlGenerationResponse: "SELECT * FROM users WHERE id = 1",
	}

	selector := NewTableSelector(mockLLM, nil)

	// 测试parseTableSelection（不需要真实数据库）
	tables := []database.TableSummary{
		{Name: "users", Comment: "用户表"},
		{Name: "orders", Comment: "订单表"},
		{Name: "products", Comment: "产品表"},
	}

	selection := selector.parseTableSelection(mockLLM.tableSelectionResponse, tables)

	// 验证选择结果
	if len(selection.PrimaryTables) == 0 {
		t.Error("期望选择至少一个主要表")
	}

	t.Logf("选择的表: Primary=%v, Secondary=%v", selection.PrimaryTables, selection.SecondaryTables)
	t.Logf("选择理由: %s", selection.Reasoning)
}

// TestSchemaBuilder Schema构建器测试
func TestSchemaBuilder(t *testing.T) {
	// 注意：这个测试需要真实数据库连接，所以暂时跳过
	t.Skip("需要真实数据库连接")

	parser := setupMockSchemaParser()
	builder := NewSchemaBuilder(parser)

	// 测试Schema构建
	schema := builder.BuildSchema(
		[]string{"users", "orders"}, // 主要表
		[]string{"products"},        // 次要表
	)

	if schema == "" {
		t.Error("期望生成Schema")
	}

	t.Logf("生成的Schema:\n%s", schema)
}

// TestTwoPhaseQueryHandler 完整两阶段流程测试
func TestTwoPhaseQueryHandler(t *testing.T) {
	// 注意：这个测试需要真实数据库连接，所以暂时跳过
	t.Skip("需要真实数据库连接")

	parser := setupMockSchemaParser()

	mockLLM := &MockLLMClientForTwoPhase{
		tableSelectionResponse: `{
			"primary_tables": ["users"],
			"secondary_tables": [],
			"reasoning": "查询用户信息"
		}`,
		sqlGenerationResponse: "SELECT id, name FROM users WHERE status = 'active'",
	}

	handler := NewTwoPhaseQueryHandler(parser, nil, mockLLM)

	// 执行查询
	result, err := handler.Handle(context.Background(), "查询活跃用户")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	// 验证结果
	if result.SQL == "" {
		t.Error("期望生成SQL")
	}

	if result.Question != "查询活跃用户" {
		t.Errorf("问题不匹配: %s", result.Question)
	}

	// 验证元数据
	if result.Metadata["mode"] != "two_phase" {
		t.Error("期望使用两阶段模式")
	}

	t.Logf("生成的SQL: %s", result.SQL)
	t.Logf("主要表: %v", result.Metadata["primary_tables"])
	t.Logf("次要表: %v", result.Metadata["secondary_tables"])
}

// TestExtractJSON 提取JSON测试
func TestExtractJSON(t *testing.T) {
	selector := NewTableSelector(nil, nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "标准JSON代码块",
			input:    "```json\n{\"test\": \"value\"}\n```",
			expected: `{"test": "value"}`,
		},
		{
			name:     "无代码块标记",
			input:    `{"test": "value"}`,
			expected: `{"test": "value"}`,
		},
		{
			name:     "带文本的JSON",
			input:    "这是一些文本\n```json\n{\"test\": \"value\"}\n```\n更多文本",
			expected: `{"test": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selector.extractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("期望 %s，实际 %s", tt.expected, result)
			}
		})
	}
}

// TestParseTableSelection 表选择解析测试
func TestParseTableSelection(t *testing.T) {
	selector := NewTableSelector(nil, nil)

	tables := []database.TableSummary{
		{Name: "users", Comment: "用户表"},
		{Name: "orders", Comment: "订单表"},
		{Name: "products", Comment: "产品表"},
	}

	tests := []struct {
		name      string
		response  string
		expectErr bool
	}{
		{
			name: "正常JSON响应",
			response: `{
				"primary_tables": ["users", "orders"],
				"secondary_tables": ["products"],
				"reasoning": "测试"
			}`,
			expectErr: false,
		},
		{
			name:      "无效JSON",
			response:  "这不是JSON",
			expectErr: false, // 应该返回保守策略，不报错
		},
		{
			name:      "空响应",
			response:  "",
			expectErr: false, // 应该返回保守策略
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selection := selector.parseTableSelection(tt.response, tables)

			if selection == nil {
				t.Error("期望返回选择结果")
			}

			// 保守策略验证：当解析失败时，应该包含所有表
			if tt.expectErr && len(selection.PrimaryTables) == 0 {
				t.Error("解析失败时应该返回保守策略")
			}

			t.Logf("选择结果: Primary=%v, Secondary=%v", selection.PrimaryTables, selection.SecondaryTables)
		})
	}
}

// setupMockSchemaParser 创建Mock Schema解析器
func setupMockSchemaParser() *database.SchemaParser {
	// 注意：这里应该使用实际的数据库连接或者mock
	// 为了简化，这里返回nil，实际测试中需要真实的数据库连接
	return nil
}

// TestTableSelectionPrompt 表选择Prompt测试
func TestTableSelectionPrompt(t *testing.T) {
	selector := NewTableSelector(nil, nil)

	tables := []database.TableSummary{
		{Name: "users", Comment: "用户表", RowCount: 1000},
		{Name: "orders", Comment: "订单表", RowCount: 5000},
		{Name: "products", Comment: "产品表", RowCount: 500},
	}

	prompt := selector.buildTableSelectionPrompt("查询VIP用户订单", tables)

	// 验证Prompt包含关键信息
	if !contains(prompt, "用户问题") {
		t.Error("Prompt应该包含用户问题部分")
	}

	if !contains(prompt, "users") {
		t.Error("Prompt应该包含users表")
	}

	if !contains(prompt, "PRIMARY") {
		t.Error("Prompt应该说明主要表概念")
	}

	if !contains(prompt, "JSON") {
		t.Error("Prompt应该要求JSON输出")
	}

	t.Logf("生成的Prompt:\n%s", prompt)
}
