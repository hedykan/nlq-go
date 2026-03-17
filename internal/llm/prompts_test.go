package llm

import (
	"strings"
	"testing"
)

// TestBuildSQLGenerationPrompt 测试构建SQL生成Prompt
func TestBuildSQLGenerationPrompt(t *testing.T) {
	schema := "数据库Schema:\n表: users\n  - id: int\n  - name: varchar\n  - email: varchar"
	question := "查询所有用户的名字"

	prompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		t.Fatalf("构建Prompt失败: %v", err)
	}

	// 验证Prompt包含必要信息
	if !strings.Contains(prompt, "数据库Schema") {
		t.Error("Prompt应包含数据库Schema信息")
	}
	if !strings.Contains(prompt, "用户问题") {
		t.Error("Prompt应包含用户问题")
	}
	if !strings.Contains(prompt, question) {
		t.Errorf("Prompt应包含用户问题: %s", question)
	}
	if !strings.Contains(prompt, schema) {
		t.Errorf("Prompt应包含Schema信息")
	}
}

// TestBuildSQLGenerationPrompt_EmptySchema 测试空Schema
func TestBuildSQLGenerationPrompt_EmptySchema(t *testing.T) {
	schema := ""
	question := "查询用户"

	_, err := BuildSQLGenerationPrompt(schema, question)
	if err == nil {
		t.Error("空Schema应该返回错误")
	}
}

// TestBuildSQLGenerationPrompt_EmptyQuestion 测试空问题
func TestBuildSQLGenerationPrompt_EmptyQuestion(t *testing.T) {
	schema := "表: users"
	question := ""

	_, err := BuildSQLGenerationPrompt(schema, question)
	if err == nil {
		t.Error("空问题应该返回错误")
	}
}

// TestBuildSQLCorrectionPrompt 测试构建SQL修正Prompt
func TestBuildSQLCorrectionPrompt(t *testing.T) {
	sql := "SELECT name FORM users" // 错误的SQL（FORM应为FROM）
	errorMsg := "You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near 'FORM users' at line 1"
	schema := "表: users\n  - id: int\n  - name: varchar"

	prompt, err := BuildSQLCorrectionPrompt(sql, errorMsg, schema)
	if err != nil {
		t.Fatalf("构建修正Prompt失败: %v", err)
	}

	// 验证Prompt包含必要信息
	if !strings.Contains(prompt, "错误的SQL") {
		t.Error("Prompt应包含错误的SQL")
	}
	if !strings.Contains(prompt, errorMsg) {
		t.Error("Prompt应包含错误信息")
	}
	if !strings.Contains(prompt, sql) {
		t.Errorf("Prompt应包含原始SQL: %s", sql)
	}
}

// TestBuildSQLCorrectionPrompt_EmptySQL 测试空SQL
func TestBuildSQLCorrectionPrompt_EmptySQL(t *testing.T) {
	sql := ""
	errorMsg := "语法错误"
	schema := "表: users"

	_, err := BuildSQLCorrectionPrompt(sql, errorMsg, schema)
	if err == nil {
		t.Error("空SQL应该返回错误")
	}
}

// TestParseSQLFromResponse 测试解析LLM响应
func TestParseSQLFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
		hasError bool
	}{
		{
			name:     "纯SQL",
			response: "SELECT * FROM users",
			expected: "SELECT * FROM users",
			hasError: false,
		},
		{
			name:     "带代码块",
			response: "```sql\nSELECT * FROM users\n```",
			expected: "SELECT * FROM users",
			hasError: false,
		},
		{
			name:     "带解释",
			response: "这是一个查询:\nSELECT * FROM users\n可以获取所有用户",
			expected: "SELECT * FROM users",
			hasError: false,
		},
		{
			name:     "多行SQL",
			response: "SELECT id, name\nFROM users\nWHERE age > 18",
			expected: "SELECT id, name\nFROM users\nWHERE age > 18",
			hasError: false,
		},
		{
			name:     "空响应",
			response: "",
			expected: "",
			hasError: true,
		},
		{
			name:     "只有解释",
			response: "这是一个很好的查询建议",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSQLFromResponse(tt.response)
			if tt.hasError {
				if err == nil {
					t.Error("期望返回错误，但返回了nil")
				}
			} else {
				if err != nil {
					t.Errorf("不应返回错误: %v", err)
				}
				if result != tt.expected {
					t.Errorf("结果不匹配\n期望: %s\n实际: %s", tt.expected, result)
				}
			}
		})
	}
}

// TestExtractSQLCodeBlock 测试提取SQL代码块
func TestExtractSQLCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
		found    bool
	}{
		{
			name:     "标准代码块",
			response: "```sql\nSELECT * FROM users\n```",
			expected: "SELECT * FROM users",
			found:    true,
		},
		{
			name:     "无语言标记",
			response: "```\nSELECT * FROM users\n```",
			expected: "SELECT * FROM users",
			found:    true,
		},
		{
			name:     "无代码块",
			response: "SELECT * FROM users",
			expected: "",
			found:    false,
		},
		{
			name:     "嵌套代码块",
			response: "```\n```sql\nSELECT * FROM users\n```\n```",
			expected: "SELECT * FROM users",
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := ExtractSQLCodeBlock(tt.response)
			if found != tt.found {
				t.Errorf("期望found=%v, 实际found=%v", tt.found, found)
			}
			if tt.found && result != tt.expected {
				t.Errorf("结果不匹配\n期望: %s\n实际: %s", tt.expected, result)
			}
		})
	}
}

// TestValidateSQLQuery 测试验证SQL查询
func TestValidateSQLQuery(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected bool
	}{
		{
			name:     "有效SELECT",
			sql:      "SELECT * FROM users",
			expected: true,
		},
		{
			name:     "带条件的SELECT",
			sql:      "SELECT name FROM users WHERE age > 18",
			expected: true,
		},
		{
			name:     "空SQL",
			sql:      "",
			expected: false,
		},
		{
			name:     "只有空白字符",
			sql:      "   \n\t  ",
			expected: false,
		},
		{
			name:     "带注释的SQL",
			sql:      "-- SELECT * FROM users",
			expected: false,
		},
		{
			name:     "多语句",
			sql:      "SELECT * FROM users; SELECT * FROM orders",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSQLQuery(tt.sql)
			if result != tt.expected {
				t.Errorf("ValidateSQLQuery(%s) = %v, 期望 %v", tt.sql, result, tt.expected)
			}
		})
	}
}

// TestFormatPromptSchema 测试格式化Schema用于Prompt
func TestFormatPromptSchema(t *testing.T) {
	table := TableSchema{
		Name: "users",
		Columns: []ColumnSchema{
			{Name: "id", Type: "int", Nullable: false, Comment: "主键ID"},
			{Name: "name", Type: "varchar(255)", Nullable: false, Comment: "用户名"},
			{Name: "email", Type: "varchar(255)", Nullable: true, Comment: "邮箱"},
		},
	}

	result := FormatPromptSchema([]TableSchema{table})

	// 验证格式化结果
	if !strings.Contains(result, "users") {
		t.Error("应包含表名")
	}
	if !strings.Contains(result, "id") {
		t.Error("应包含列名")
	}
	if !strings.Contains(result, "主键ID") {
		t.Error("应包含列注释")
	}
}

// TestBuildFewShotExamples 测试构建Few-Shot示例
func TestBuildFewShotExamples(t *testing.T) {
	examples := BuildFewShotExamples()

	if len(examples) == 0 {
		t.Error("应至少有一个示例")
	}

	// 验证每个示例都有问题和SQL
	for _, example := range examples {
		if example.Question == "" {
			t.Error("示例问题不能为空")
		}
		if example.SQL == "" {
			t.Error("示例SQL不能为空")
		}
	}
}

// TestBuildPromptWithExamples 测试带示例的Prompt构建
func TestBuildPromptWithExamples(t *testing.T) {
	schema := "表: users"
	question := "查询所有用户"

	prompt, err := BuildPromptWithExamples(schema, question, true)
	if err != nil {
		t.Fatalf("构建Prompt失败: %v", err)
	}

	// 验证包含示例
	if !strings.Contains(prompt, "示例") {
		t.Error("应包含示例")
	}

	// 不使用示例
	promptNoExamples, err := BuildPromptWithExamples(schema, question, false)
	if err != nil {
		t.Fatalf("构建Prompt失败: %v", err)
	}

	if strings.Contains(promptNoExamples, "示例") {
		t.Error("不应包含示例")
	}
}
