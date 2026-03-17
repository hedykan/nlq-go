package llm

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ColumnSchema 列结构定义
type ColumnSchema struct {
	Name     string
	Type     string
	Nullable bool
	Comment  string
}

// TableSchema 表结构定义
type TableSchema struct {
	Name    string
	Columns []ColumnSchema
}

// FewShotExample Few-Shot学习示例
type FewShotExample struct {
	Question string
	SQL      string
}

const (
	// SQLGenerationPromptTemplate SQL生成Prompt模板
	SQLGenerationPromptTemplate = `你是一个专业的SQL专家。根据数据库Schema和用户问题，生成准确的SQL查询语句。

{{.Schema}}

用户问题: {{.Question}}

请只返回SQL语句，不要包含任何解释或注释。确保SQL语法正确且符合MySQL规范。

SQL查询:`
)

// BuildSQLGenerationPrompt 构建SQL生成Prompt
func BuildSQLGenerationPrompt(schema, question string) (string, error) {
	if strings.TrimSpace(schema) == "" {
		return "", errors.New("数据库Schema不能为空")
	}
	if strings.TrimSpace(question) == "" {
		return "", errors.New("用户问题不能为空")
	}

	prompt := strings.ReplaceAll(SQLGenerationPromptTemplate, "{{.Schema}}", schema)
	prompt = strings.ReplaceAll(prompt, "{{.Question}}", question)

	return prompt, nil
}

// BuildSQLCorrectionPrompt 构建SQL修正Prompt
func BuildSQLCorrectionPrompt(sql, errorMsg, schema string) (string, error) {
	if strings.TrimSpace(sql) == "" {
		return "", errors.New("SQL语句不能为空")
	}
	if strings.TrimSpace(errorMsg) == "" {
		return "", errors.New("错误信息不能为空")
	}

	prompt := fmt.Sprintf(`你是一个SQL专家。以下SQL查询执行失败，请根据错误信息修正SQL。

数据库Schema:
%s

错误的SQL查询:
%s

错误信息:
%s

请分析错误原因并提供修正后的SQL查询。只返回修正后的SQL语句，不要包含解释。

修正后的SQL:`, schema, sql, errorMsg)

	return prompt, nil
}

// ParseSQLFromResponse 从LLM响应中解析SQL
func ParseSQLFromResponse(response string) (string, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return "", errors.New("LLM响应为空")
	}

	// 尝试提取代码块中的SQL
	if code, found := ExtractSQLCodeBlock(response); found {
		return strings.TrimSpace(code), nil
	}

	// 从文本中查找SQL语句
	lines := strings.Split(response, "\n")
	var sqlLines []string
	inSQL := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 跳过空行
		if trimmedLine == "" {
			if inSQL {
				// SQL中的空行也要保留
				sqlLines = append(sqlLines, "")
			}
			continue
		}

		// 检查是否是SQL语句的开始
		if !inSQL && (strings.HasPrefix(strings.ToUpper(trimmedLine), "SELECT") ||
			strings.HasPrefix(strings.ToUpper(trimmedLine), "WITH")) {
			inSQL = true
		}

		if inSQL {
			sqlLines = append(sqlLines, trimmedLine)

			// 检查是否是SQL语句的结束
			if strings.HasSuffix(trimmedLine, ";") {
				break
			}

			// 如果遇到新的非SQL关键字，可能是后续的解释文本
			// 检查这一行是否看起来像SQL（包含常见SQL关键字）
			upperLine := strings.ToUpper(trimmedLine)
			sqlKeywords := []string{"SELECT", "FROM", "WHERE", "JOIN", "GROUP", "ORDER", "HAVING", "LIMIT", "AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "WITH"}
			isSQLLine := false
			for _, keyword := range sqlKeywords {
				if strings.Contains(upperLine, keyword) {
					isSQLLine = true
					break
				}
			}

			// 如果这一行不包含SQL关键字，且不是SQL的开始，可能是解释文本
			if !isSQLLine && len(sqlLines) > 1 {
				// 移除这一行
				sqlLines = sqlLines[:len(sqlLines)-1]
				break
			}
		}
	}

	if len(sqlLines) == 0 {
		return "", errors.New("未找到有效的SQL语句")
	}

	sql := strings.Join(sqlLines, "\n")
	sql = strings.TrimSuffix(sql, ";")

	return sql, nil
}

// ExtractSQLCodeBlock 从响应中提取SQL代码块
func ExtractSQLCodeBlock(response string) (string, bool) {
	// 匹配 ```sql 或 ``` 代码块
	re := regexp.MustCompile("```(?:sql)?\n([^`]+)\n```")
	matches := re.FindStringSubmatch(response)

	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), true
	}

	return "", false
}

// ValidateSQLQuery 验证SQL查询是否有效
func ValidateSQLQuery(sql string) bool {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return false
	}

	// 去除注释
	if strings.Contains(sql, "--") || strings.Contains(sql, "/*") {
		return false
	}

	// 检查多语句
	if strings.Contains(sql, ";") {
		// 移除字符串字面量中的内容，避免误判
		cleanedSQL := removeStringLiterals(sql)

		// 如果有分号，检查分号后是否还有内容
		parts := strings.SplitN(cleanedSQL, ";", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			// 分号后还有内容，可能是多语句
			return false
		}
	}

	// 检查是否是SELECT或WITH开头
	upperSQL := strings.ToUpper(sql)
	return strings.HasPrefix(upperSQL, "SELECT") || strings.HasPrefix(upperSQL, "WITH")
}

// removeStringLiterals 简单的字符串字面量移除
func removeStringLiterals(sql string) string {
	// 移除单引号字符串
	re := regexp.MustCompile(`'[^']*'`)
	return re.ReplaceAllString(sql, "")
}

// FormatPromptSchema 格式化Schema用于Prompt
func FormatPromptSchema(tables []TableSchema) string {
	var builder strings.Builder

	builder.WriteString("数据库Schema:\n\n")

	for _, table := range tables {
		builder.WriteString(fmt.Sprintf("表: %s\n", table.Name))

		for _, col := range table.Columns {
		 nullable := "NOT NULL"
			if col.Nullable {
				nullable = "NULL"
			}

			comment := ""
			if col.Comment != "" {
				comment = fmt.Sprintf(" // %s", col.Comment)
			}

			builder.WriteString(fmt.Sprintf("  - %s: %s %s%s\n",
				col.Name, col.Type, nullable, comment))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// BuildFewShotExamples 构建Few-Shot示例
func BuildFewShotExamples() []FewShotExample {
	return []FewShotExample{
		{
			Question: "查询所有用户的数量",
			SQL:      "SELECT COUNT(*) as total FROM users",
		},
		{
			Question: "查询年龄大于25岁的用户名字",
			SQL:      "SELECT name FROM users WHERE age > 25",
		},
		{
			Question: "查询每个城市的用户数量",
			SQL:      "SELECT city, COUNT(*) as user_count FROM users GROUP BY city",
		},
		{
			Question: "查询订单金额最高的前10个用户",
			SQL:      "SELECT u.name, SUM(o.amount) as total_amount FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.id, u.name ORDER BY total_amount DESC LIMIT 10",
		},
		{
			Question: "查询最近7天内的订单",
			SQL:      "SELECT * FROM orders WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)",
		},
	}
}

// BuildPromptWithExamples 构建带示例的Prompt
func BuildPromptWithExamples(schema, question string, includeExamples bool) (string, error) {
	if includeExamples {
		examples := BuildFewShotExamples()

		var exampleBuilder strings.Builder
		exampleBuilder.WriteString("\n以下是参考示例：\n\n")

		for i, example := range examples {
			exampleBuilder.WriteString(fmt.Sprintf("示例%d:\n", i+1))
			exampleBuilder.WriteString(fmt.Sprintf("问题: %s\n", example.Question))
			exampleBuilder.WriteString(fmt.Sprintf("SQL: %s\n\n", example.SQL))
		}

		schemaWithExamples := schema + exampleBuilder.String()
		return BuildSQLGenerationPrompt(schemaWithExamples, question)
	}

	return BuildSQLGenerationPrompt(schema, question)
}

// GenerateSystemPrompt 生成系统Prompt
func GenerateSystemPrompt() string {
	return `你是一个专业的SQL助手，专门负责将自然语言问题转换为准确的SQL查询语句。

你的职责：
1. 理解用户的自然语言问题
2. 根据数据库Schema生成正确的SQL查询
3. 确保SQL语法正确且符合MySQL规范
4. 只返回SQL语句，不要包含任何解释或注释

注意事项：
- 只使用SELECT查询，不要使用DELETE、UPDATE、INSERT等修改数据的语句
- 确保列名和表名与Schema中定义的完全一致
- 使用适当的JOIN来关联多个表
- 使用WHERE、GROUP BY、ORDER BY等子句来精确查询
- 当需要聚合时，使用COUNT、SUM、AVG等聚合函数
- 使用LIMIT来限制返回的结果数量`
}

// BuildChatMessages 构建聊天消息
func BuildChatMessages(systemPrompt, userPrompt string) []map[string]string {
	return []map[string]string{
		{
			"role":    "system",
			"content": systemPrompt,
		},
		{
			"role":    "user",
			"content": userPrompt,
		},
	}
}
