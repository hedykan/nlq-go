package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/llm"
	"github.com/channelwill/nlq/internal/sql"
	"gorm.io/gorm"
)

// TwoPhaseQueryHandler 两阶段查询处理器（适合动态数据库）
type TwoPhaseQueryHandler struct {
	db            *database.SchemaParser
	dbGORM        *gorm.DB                  // GORM数据库连接（用于执行SQL）
	llmClient     LLMClient
	tableSelector *TableSelector
	schemaBuilder *SchemaBuilder
	executor      *sql.Executor             // SQL执行器
}

// TableSelector 表选择器（阶段1）
type TableSelector struct {
	llmClient      LLMClient
	exampleRepo    *llm.ExampleRepository
	fieldAliasMap  map[string][]string // 字段别名映射
}

// SchemaBuilder Schema构建器（阶段2）
type SchemaBuilder struct {
	parser *database.SchemaParser
}

// TableSelection 表选择结果
type TableSelection struct {
	PrimaryTables      []string             `json:"primary_tables"`       // 主要相关的表
	SecondaryTables    []string             `json:"secondary_tables"`     // 可能相关的表（保险起见）
	Reasoning          string               `json:"reasoning"`            // 选择理由（用于调试）
	FieldClarification *FieldClarification  `json:"field_clarification,omitempty"` // 字段澄清信息
}

// FieldClarification 字段澄清信息
type FieldClarification struct {
	AmbiguousField     string   `json:"ambiguous_field"`     // 模糊字段名（如 "name"）
	PossibleFields     []string `json:"possible_fields"`     // 可能的实际字段
	SuggestedQuestion  string   `json:"suggested_question"`  // 建议的问题示例
}

// NewTwoPhaseQueryHandler 创建两阶段查询处理器
func NewTwoPhaseQueryHandler(parser *database.SchemaParser, dbGORM *gorm.DB, llmClient LLMClient) *TwoPhaseQueryHandler {
	return &TwoPhaseQueryHandler{
		db:            parser,
		dbGORM:        dbGORM,
		llmClient:     llmClient,
		tableSelector: NewTableSelector(llmClient),
		schemaBuilder: NewSchemaBuilder(parser),
		executor:      sql.NewExecutor(dbGORM),
	}
}

// Handle 两阶段处理流程
func (h *TwoPhaseQueryHandler) Handle(ctx context.Context, question string) (*QueryResult, error) {
	start := time.Now()

	// 阶段1: 选择相关表
	selection, err := h.tableSelector.SelectTables(ctx, question, h.db)
	if err != nil {
		return &QueryResult{
			Error: fmt.Sprintf("表选择失败: %v", err),
		}, err
	}

	// 检查是否需要字段澄清
	if selection.FieldClarification != nil {
		// 返回字段澄清信息，不执行SQL生成
		return &QueryResult{
			Question:           question,
			FieldClarification: selection.FieldClarification,
			Duration:           time.Since(start),
			Metadata: map[string]interface{}{
				"primary_tables":        selection.PrimaryTables,
				"secondary_tables":      selection.SecondaryTables,
				"reasoning":             selection.Reasoning,
				"mode":                  "two_phase",
				"needs_clarification":   true,
			},
		}, nil
	}

	// 阶段2: 构建相关表的Schema
	schema := h.schemaBuilder.BuildSchema(selection.PrimaryTables, selection.SecondaryTables)

	// 阶段3: 生成SQL（使用精准的Schema）
	generatedSQL, err := h.llmClient.GenerateSQL(ctx, schema, question)
	if err != nil {
		return &QueryResult{
			Error: fmt.Sprintf("SQL生成失败: %v", err),
		}, err
	}

	// 阶段4: 验证SQL
	if err := h.executor.ValidateOnly(generatedSQL); err != nil {
		return &QueryResult{
			Question: question,
			SQL:      generatedSQL,
			Error:    fmt.Sprintf("SQL验证失败: %v", err),
			Metadata: map[string]interface{}{
				"primary_tables":   selection.PrimaryTables,
				"secondary_tables": selection.SecondaryTables,
				"reasoning":        selection.Reasoning,
				"mode":             "two_phase",
			},
		}, err
	}

	// 阶段5: 执行SQL
	execResult, err := h.executor.Execute(ctx, generatedSQL)
	if err != nil {
		return &QueryResult{
			Question: question,
			SQL:      generatedSQL,
			Error:    fmt.Sprintf("执行SQL失败: %v", err),
			Metadata: map[string]interface{}{
				"primary_tables":   selection.PrimaryTables,
				"secondary_tables": selection.SecondaryTables,
				"reasoning":        selection.Reasoning,
				"mode":             "two_phase",
			},
		}, err
	}

	return &QueryResult{
		Question: question,
		SQL:      generatedSQL,
		Result:   execResult,
		Duration: time.Since(start),
		Metadata: map[string]interface{}{
			"primary_tables":   selection.PrimaryTables,
			"secondary_tables": selection.SecondaryTables,
			"reasoning":        selection.Reasoning,
			"mode":             "two_phase",
		},
	}, nil
}

// HandleWithSQL 直接使用SQL查询
func (h *TwoPhaseQueryHandler) HandleWithSQL(ctx context.Context, sqlQuery string) (*QueryResult, error) {
	start := time.Now()

	result := &QueryResult{
		SQL:      sqlQuery,
		Metadata: make(map[string]interface{}),
	}

	// 验证SQL
	if err := h.executor.ValidateOnly(sqlQuery); err != nil {
		result.Error = fmt.Sprintf("SQL验证失败: %v", err)
		return result, err
	}

	// 执行SQL
	execResult, err := h.executor.Execute(ctx, sqlQuery)
	if err != nil {
		result.Error = fmt.Sprintf("执行SQL失败: %v", err)
		return result, err
	}

	result.Result = execResult
	result.Duration = time.Since(start)

	return result, nil
}

// SetKnowledge 设置知识库文档
func (h *TwoPhaseQueryHandler) SetKnowledge(docs []knowledge.Document) error {
	// 检查LLM客户端是否支持知识库
	if h.llmClient == nil {
		return fmt.Errorf("LLM客户端未初始化")
	}

	// 尝试将知识库设置到LLM客户端
	if glmClient, ok := h.llmClient.(*llm.GLMClient); ok {
		glmClient.SetKnowledge(docs)
		return nil
	}

	return fmt.Errorf("LLM客户端不支持知识库功能")
}

// NewTableSelector 创建表选择器
func NewTableSelector(llmClient LLMClient) *TableSelector {
	// 创建示例仓库（使用data目录）
	dataPath := "./data"
	exampleRepo := llm.NewExampleRepository(dataPath)

	return &TableSelector{
		llmClient:     llmClient,
		exampleRepo:   exampleRepo,
		fieldAliasMap: initializeFieldAliasMap(),
	}
}

// initializeFieldAliasMap 初始化字段别名映射
func initializeFieldAliasMap() map[string][]string {
	return map[string][]string{
		"name":     {"username", "shop_name", "customer_name", "first_name", "last_name", "full_name"},
		"user":     {"username", "customer_name", "user_name"},
		"customer": {"customer_name", "client_name"},
		"email":    {"email_address", "mail"},
		"phone":    {"mobile", "telephone", "contact_number"},
		"time":     {"created_at", "updated_at", "timestamp", "date"},
		"status":   {"state", "level", "condition"},
		"price":    {"amount", "cost", "total"},
	}
}

// SelectTables 阶段1: 选择相关表（增强版，包含字段级别上下文）
func (s *TableSelector) SelectTables(ctx context.Context, question string, parser *database.SchemaParser) (*TableSelection, error) {
	// 1. 获取所有表的增强摘要信息（包含关键字段）
	tables, err := parser.GetTableSummariesEnhanced()
	if err != nil {
		return nil, fmt.Errorf("获取表摘要失败: %w", err)
	}

	// 2. 分析问题中的模糊字段
	ambiguousFields := s.analyzeAmbiguousFields(question, tables)

	// 3. 构建增强的Prompt（包含Few-shot示例和字段信息）
	prompt := s.buildTableSelectionPromptEnhanced(question, tables, ambiguousFields)

	// 4. 调用LLM选择表
	response, err := s.callLLMForTableSelection(ctx, prompt, question)
	if err != nil {
		return nil, fmt.Errorf("LLM表选择失败: %w", err)
	}

	// 5. 解析LLM响应
	selection := s.parseTableSelection(response, tables)

	// 6. 计算字段匹配置信度
	// 暂时禁用字段澄清功能，因为当前实现过于保守
	// TODO: 优化字段匹配算法后再启用
	_ = s.calculateFieldMatchConfidence(question, selection.PrimaryTables, tables)
	// 字段澄清已禁用
	selection.FieldClarification = nil


	return selection, nil
}

// callLLMForTableSelection 调用LLM进行表选择
func (s *TableSelector) callLLMForTableSelection(ctx context.Context, prompt, question string) (string, error) {
	// 使用GenerateContent接口获取表选择结果
	systemPrompt := "你是数据库表选择专家。根据用户问题，从可用表中选择最相关的表，并返回JSON格式的结果。"
	userPrompt := prompt + "\n\n" + question

	response, err := s.llmClient.GenerateContent(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

// buildTableSelectionPrompt 构建表选择Prompt
func (s *TableSelector) buildTableSelectionPrompt(question string, tables []database.TableSummary) string {
	var builder strings.Builder

	builder.WriteString("# 任务说明\n")
	builder.WriteString("你是数据库表选择专家。根据用户问题，从可用表中选择最相关的表。\n\n")

	builder.WriteString("# 用户问题\n")
	builder.WriteString(fmt.Sprintf("问题: %s\n\n", question))

	builder.WriteString("# 可用表列表\n")
	for i, table := range tables {
		builder.WriteString(fmt.Sprintf("%d. %s", i+1, table.Name))
		if table.Comment != "" {
			builder.WriteString(fmt.Sprintf(" - %s", table.Comment))
		}
		if table.RowCount > 0 {
			builder.WriteString(fmt.Sprintf(" (数据量: %d条)", table.RowCount))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("\n# 选择标准\n")
	builder.WriteString("1. PRIMARY: 直接相关、必需包含的表\n")
	builder.WriteString("2. SECONDARY: 可能相关、作为备选的表\n")
	builder.WriteString("3. 如果不确定，宁可多选，不要遗漏\n")
	builder.WriteString("4. 考虑表名的语义（如: user表用于用户相关查询）\n\n")

	builder.WriteString("# 输出格式（只返回JSON，不要其他内容）\n")
	builder.WriteString("```json\n")
	builder.WriteString("{\n")
	builder.WriteString("  \"primary_tables\": [\"表名1\", \"表名2\"],\n")
	builder.WriteString("  \"secondary_tables\": [\"表名3\"],\n")
	builder.WriteString("  \"reasoning\": \"选择理由简述\"\n")
	builder.WriteString("}\n")
	builder.WriteString("```\n")

	return builder.String()
}

// parseTableSelection 解析表选择结果
func (s *TableSelector) parseTableSelection(response string, allTables []database.TableSummary) *TableSelection {
	// 尝试从响应中提取JSON
	jsonStr := s.extractJSON(response)
	if jsonStr == "" {
		// 如果无法解析JSON，返回一个保守的选择（包含所有表）
		var allTableNames []string
		for _, t := range allTables {
			allTableNames = append(allTableNames, t.Name)
		}
		return &TableSelection{
			PrimaryTables:   allTableNames,
			SecondaryTables: []string{},
			Reasoning:       "无法解析LLM响应，使用所有表作为保守策略",
		}
	}

	// 解析JSON
	var selection struct {
		PrimaryTables   []string `json:"primary_tables"`
		SecondaryTables []string `json:"secondary_tables"`
		Reasoning       string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &selection); err != nil {
		// JSON解析失败，返回保守策略
		var allTableNames []string
		for _, t := range allTables {
			allTableNames = append(allTableNames, t.Name)
		}
		return &TableSelection{
			PrimaryTables:   allTableNames,
			SecondaryTables: []string{},
			Reasoning:       "JSON解析失败，使用所有表作为保守策略",
		}
	}

	return &TableSelection{
		PrimaryTables:   selection.PrimaryTables,
		SecondaryTables: selection.SecondaryTables,
		Reasoning:       selection.Reasoning,
	}
}

// extractJSON 从文本中提取JSON代码块
func (s *TableSelector) extractJSON(text string) string {
	// 查找```json代码块
	start := strings.Index(text, "```json")
	if start == -1 {
		start = strings.Index(text, "```")
		if start == -1 {
			// 尝试直接解析为JSON（没有代码块标记）
			trimmed := strings.TrimSpace(text)
			if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
				return trimmed
			}
			return "" // 不是JSON格式
		}
		start += 3 // 跳过```
	} else {
		start += 7 // 跳过```json
	}

	end := strings.Index(text[start:], "```")
	if end == -1 {
		return "" // 没有结束标记
	}

	jsonStr := strings.TrimSpace(text[start : start+end])
	return jsonStr
}

// NewSchemaBuilder 创建Schema构建器
func NewSchemaBuilder(parser *database.SchemaParser) *SchemaBuilder {
	return &SchemaBuilder{
		parser: parser,
	}
}

// BuildSchema 阶段2: 构建Schema
func (b *SchemaBuilder) BuildSchema(primaryTables, secondaryTables []string) string {
	var builder strings.Builder

	builder.WriteString("# 数据库Schema\n\n")

	// 主要表（详细信息）
	if len(primaryTables) > 0 {
		builder.WriteString("## 主要表\n")
		for _, tableName := range primaryTables {
			table, _ := b.parser.GetTableDetail(tableName)
			builder.WriteString(b.formatTableDetail(table, true))
		}
	}

	// 次要表（简化信息）
	if len(secondaryTables) > 0 {
		builder.WriteString("\n## 备选表\n")
		builder.WriteString("(如果主要表不足以回答问题，可以考虑以下表)\n\n")
		for _, tableName := range secondaryTables {
			table, _ := b.parser.GetTableDetail(tableName)
			builder.WriteString(b.formatTableDetail(table, false))
		}
	}

	return builder.String()
}

// formatTableDetail 格式化表详情
func (b *SchemaBuilder) formatTableDetail(table database.TableDetail, verbose bool) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("### 表: %s\n", table.Name))
	if table.Comment != "" {
		builder.WriteString(fmt.Sprintf("说明: %s\n", table.Comment))
	}

	if verbose {
		// 详细模式：显示所有字段
		builder.WriteString("\n列:\n")
		for _, col := range table.Columns {
			builder.WriteString(fmt.Sprintf("  - %s (%s)", col.Name, col.Type))
			if !col.Nullable {
				builder.WriteString(" NOT NULL")
			}
			if col.Comment != "" {
				builder.WriteString(fmt.Sprintf(" -- %s", col.Comment))
			}
			builder.WriteString("\n")
		}

		// 显示外键关系
		if len(table.ForeignKeys) > 0 {
			builder.WriteString("\n关联:\n")
			for _, fk := range table.ForeignKeys {
				builder.WriteString(fmt.Sprintf("  - %s -> %s.%s\n", fk.Column, fk.ReferTable, fk.ReferColumn))
			}
		}
	} else {
		// 简化模式：只显示关键字段
		builder.WriteString(fmt.Sprintf("字段数: %d\n", len(table.Columns)))
	}

	builder.WriteString("\n")

	return builder.String()
}

// ========== TableSelector 增强方法 ==========

// buildTableSelectionPromptEnhanced 构建增强的表选择Prompt
func (s *TableSelector) buildTableSelectionPromptEnhanced(question string, tables []database.TableSummary, ambiguousFields []string) string {
	var builder strings.Builder

	builder.WriteString("# 任务说明\n")
	builder.WriteString("你是数据库表选择专家。根据用户问题，从可用表中选择最相关的表。\n\n")

	builder.WriteString("# 用户问题\n")
	builder.WriteString(fmt.Sprintf("问题: %s\n\n", question))

	// 添加Few-shot示例
	examples := s.exampleRepo.RetrieveExamples(question, 2)
	builder.WriteString(s.exampleRepo.FormatExamplesForPrompt(examples))

	builder.WriteString("# 可用表列表\n")
	for i, table := range tables {
		builder.WriteString(fmt.Sprintf("%d. %s", i+1, table.Name))
		if table.Comment != "" {
			builder.WriteString(fmt.Sprintf(" - %s", table.Comment))
		}
		if table.RowCount > 0 {
			builder.WriteString(fmt.Sprintf(" (数据量: %d条)", table.RowCount))
		}
		// 显示关键字段
		if len(table.KeyColumns) > 0 {
			builder.WriteString(fmt.Sprintf("\n   关键字段: %s", strings.Join(table.KeyColumns, ", ")))
		}
		builder.WriteString("\n")
	}

	// 添加字段语义分析
	if len(ambiguousFields) > 0 {
		builder.WriteString("\n# 字段语义分析\n")
		builder.WriteString("问题中包含以下模糊字段名，可能对应不同的实际字段：\n")
		for _, field := range ambiguousFields {
			if aliases, ok := s.fieldAliasMap[field]; ok {
				builder.WriteString(fmt.Sprintf("- '%s' 可能是: %s\n", field, strings.Join(aliases, ", ")))
			}
		}
		builder.WriteString("\n")
	}

	builder.WriteString("# 选择标准\n")
	builder.WriteString("1. PRIMARY: 直接相关、必需包含的表\n")
	builder.WriteString("2. SECONDARY: 可能相关、作为备选的表\n")
	builder.WriteString("3. 如果不确定，宁可多选，不要遗漏\n")
	builder.WriteString("4. 考虑表名的语义（如: user表用于用户相关查询）\n")
	builder.WriteString("5. 优先选择包含问题中提及字段的表\n\n")

	builder.WriteString("# 输出格式（只返回JSON，不要其他内容）\n")
	builder.WriteString("```json\n")
	builder.WriteString("{\n")
	builder.WriteString("  \"primary_tables\": [\"表名1\", \"表名2\"],\n")
	builder.WriteString("  \"secondary_tables\": [\"表名3\"],\n")
	builder.WriteString("  \"reasoning\": \"选择理由简述\"\n")
	builder.WriteString("}\n")
	builder.WriteString("```\n")

	return builder.String()
}

// analyzeAmbiguousFields 分析问题中的模糊字段名
func (s *TableSelector) analyzeAmbiguousFields(question string, tables []database.TableSummary) []string {
	questionLower := strings.ToLower(question)
	var ambiguousFields []string

	// 检查常见的模糊字段名
	commonAmbiguousFields := []string{"name", "user", "customer", "info", "detail", "data"}

	for _, field := range commonAmbiguousFields {
		if strings.Contains(questionLower, field) {
			// 检查是否在多个表中都有相关字段
			foundInMultipleTables := 0
			for _, table := range tables {
				for _, keyCol := range table.KeyColumns {
					if strings.Contains(strings.ToLower(keyCol), field) {
						foundInMultipleTables++
						break
					}
				}
			}
			if foundInMultipleTables >= 2 {
				ambiguousFields = append(ambiguousFields, field)
			}
		}
	}

	return ambiguousFields
}

// calculateFieldMatchConfidence 计算字段匹配置信度
func (s *TableSelector) calculateFieldMatchConfidence(question string, selectedTables []string, allTables []database.TableSummary) float64 {
	questionLower := strings.ToLower(question)
	totalMatches := 0
	totalFields := 0

	// 统计问题中提到的字段在选定表中的匹配度
	for _, tableName := range selectedTables {
		for _, table := range allTables {
			if table.Name == tableName {
				for _, keyCol := range table.KeyColumns {
					totalFields++
					if strings.Contains(questionLower, strings.ToLower(keyCol)) {
						totalMatches++
					}
				}
				break
			}
		}
	}

	if totalFields == 0 {
		return 0.5 // 默认中等置信度
	}

	return float64(totalMatches) / float64(totalFields)
}

// getPossibleFields 获取可能的字段列表
func (s *TableSelector) getPossibleFields(_ string, selectedTables []string, allTables []database.TableSummary) []string {
	var possibleFields []string

	for _, tableName := range selectedTables {
		for _, table := range allTables {
			if table.Name == tableName {
				for _, keyCol := range table.KeyColumns {
					possibleFields = append(possibleFields, fmt.Sprintf("%s.%s", table.Name, keyCol))
				}
				break
			}
		}
	}

	return possibleFields
}

// buildSuggestedQuestion 构建建议的问题
func (s *TableSelector) buildSuggestedQuestion(_ string, possibleFields []string) string {
	// 提取字段名
	var fieldNames []string
	for _, field := range possibleFields {
		parts := strings.Split(field, ".")
		if len(parts) > 1 {
			fieldNames = append(fieldNames, parts[1])
		}
	}

	if len(fieldNames) > 0 {
		return fmt.Sprintf("当前描述不准确，是否是查找以下字段内容：%s", strings.Join(fieldNames, "、"))
	}

	return "请提供更具体的字段名称"
}
