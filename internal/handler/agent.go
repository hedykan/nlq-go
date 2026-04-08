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
	"github.com/channelwill/nlq/pkg/utils"
	"gorm.io/gorm"
)

// AgentConfig Agent 处理器配置
type AgentConfig struct {
	MaxSelfCorrect int  // SQL自检修正最大次数
	MaxTurns       int  // 最大总轮次
	Verbose        bool // 是否返回中间推理过程
}

// DefaultAgentConfig 默认 Agent 配置
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		MaxSelfCorrect: 3,
		MaxTurns:       5,
		Verbose:        false,
	}
}

// AgentStep Agent 推理步骤记录
type AgentStep struct {
	Turn     int                    `json:"turn"`
	Action   string                 `json:"action"` // resource_selection / sql_generation / self_check / execution / error_correction
	Detail   string                 `json:"detail"`
	Duration time.Duration          `json:"duration"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(step AgentStep)

// AgentQueryHandler Agent 查询处理器
// 用户输入一次问题 → LLM 内部多轮自主推理 → 自检修正 → 返回结果
type AgentQueryHandler struct {
	db               *database.SchemaParser
	dbGORM           *gorm.DB
	llmClient        llm.LLMClient
	knowledgeRouter  *knowledge.Router
	executor         *sql.Executor
	config           AgentConfig
}

// NewAgentQueryHandler 创建 Agent 查询处理器
func NewAgentQueryHandler(
	dbParser *database.SchemaParser,
	dbGORM *gorm.DB,
	llmClient llm.LLMClient,
	knowledgeRouter *knowledge.Router,
	config AgentConfig,
) *AgentQueryHandler {
	if config.MaxSelfCorrect <= 0 {
		config.MaxSelfCorrect = 3
	}
	if config.MaxTurns <= 0 {
		config.MaxTurns = 5
	}

	return &AgentQueryHandler{
		db:              dbParser,
		dbGORM:          dbGORM,
		llmClient:       llmClient,
		knowledgeRouter: knowledgeRouter,
		executor:        sql.NewExecutor(dbGORM),
		config:          config,
	}
}

// QueryResult 查询结果
type QueryResult struct {
	Question string                 `json:"question"`
	SQL      string                 `json:"sql"`
	Result   *sql.ExecuteResult     `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration time.Duration          `json:"duration"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Steps    []AgentStep            `json:"steps,omitempty"`
}

// Handle 处理自然语言查询（Agent 模式主入口）
func (h *AgentQueryHandler) Handle(ctx context.Context, question string) (*QueryResult, error) {
	return h.HandleWithProgress(ctx, question, nil)
}

// HandleWithProgress 处理自然语言查询（带进度回调）
func (h *AgentQueryHandler) HandleWithProgress(ctx context.Context, question string, callback ProgressCallback) (*QueryResult, error) {
	start := time.Now()
	steps := make([]AgentStep, 0)

	emitStep := func(step AgentStep) {
		steps = append(steps, step)
		if callback != nil {
			callback(step)
		}
	}

	utils.Info("🤖 [Agent] 开始处理: %s", question)
	utils.Info("🤖 [Agent] 配置: maxSelfCorrect=%d, maxTurns=%d", h.config.MaxSelfCorrect, h.config.MaxTurns)

	// ========== 初始化对话 ==========
	systemPrompt := h.buildSystemPrompt()
	conversation := llm.NewConversation(systemPrompt)

	// ========== 轮次1: 资源选择 ==========
	step1Start := time.Now()
	emitStep(AgentStep{Turn: 1, Action: "resource_selection", Detail: "正在分析问题，选择相关知识文档和数据库表..."})

	selectedDocs, selectedTables, selectionReasoning, err := h.selectResources(ctx, question, conversation)
	if err != nil {
		utils.Error("❌ [Agent] 轮次1资源选择失败: %v", err)
		return h.buildErrorResult(question, err, steps, start), err
	}

	emitStep(AgentStep{
		Turn:     1,
		Action:   "resource_selection",
		Detail:   fmt.Sprintf("选择了 %d 个知识文档、%d 张表", len(selectedDocs), len(selectedTables)),
		Duration: time.Since(step1Start),
		Data: map[string]interface{}{
			"selected_docs":   docTitles(selectedDocs),
			"selected_tables": selectedTables,
			"reasoning":       selectionReasoning,
		},
	})

	utils.Info("🤖 [Agent] 轮次1完成 | 文档: %d | 表: %v | 耗时: %v",
		len(selectedDocs), selectedTables, time.Since(step1Start))

	// ========== 预取Schema（方案5：与LLM调用并行） ==========
	prefetchStart := time.Now()
	prefetchedSchema := ""
	if len(selectedTables) > 0 {
		schema, err := h.db.FormatTablesForPrompt(selectedTables)
		if err == nil {
			prefetchedSchema = schema
		} else {
			utils.Warn("⚠️  [Agent] 预取Schema失败: %v", err)
		}
	}
	utils.Info("🤖 [Agent] Schema预取完成 | 耗时: %v, 大小: %d字符", time.Since(prefetchStart), len(prefetchedSchema))

	// ========== 轮次2: SQL 生成 ==========
	step2Start := time.Now()
	emitStep(AgentStep{Turn: 2, Action: "sql_generation", Detail: "正在生成SQL..."})

	generatedSQL, err := h.generateSQL(ctx, question, selectedDocs, selectedTables, conversation, prefetchedSchema)
	if err != nil {
		utils.Error("❌ [Agent] 轮次2 SQL生成失败: %v", err)
		return h.buildErrorResult(question, err, steps, start), err
	}

	conversation.AddAssistant(generatedSQL)

	emitStep(AgentStep{
		Turn:     2,
		Action:   "sql_generation",
		Detail:   "SQL生成完成",
		Duration: time.Since(step2Start),
		Data: map[string]interface{}{
			"sql": generatedSQL,
		},
	})

	utils.Info("🤖 [Agent] 轮次2完成 | SQL: %s | 耗时: %v", generatedSQL, time.Since(step2Start))

	// ========== 轮次3: 语法+逻辑自检 ==========
	currentSQL := generatedSQL

	// 方案2优化：简单查询跳过自检（1张表 + 无知识文档 = 低风险）
	if len(selectedDocs) == 0 && len(selectedTables) <= 1 {
		utils.Info("⏭️  [Agent] 简单查询（%d张表+0个文档），跳过自检", len(selectedTables))
		emitStep(AgentStep{Turn: 3, Action: "self_check", Detail: "简单查询，跳过自检", Duration: 0})
	} else {
		for checkRound := 1; checkRound <= h.config.MaxSelfCorrect; checkRound++ {
		step3Start := time.Now()
		actionLabel := "self_check"
		detail := "正在自检SQL（语法+逻辑审查）..."
		if checkRound > 1 {
			actionLabel = "self_correction"
			detail = fmt.Sprintf("第%d轮自检修正...", checkRound)
		}

		emitStep(AgentStep{Turn: 2 + checkRound, Action: actionLabel, Detail: detail})

		isCorrect, issues, fixedSQL, err := h.selfCheckSQL(ctx, question, currentSQL, selectedTables, conversation)
		if err != nil {
			utils.Warn("⚠️  [Agent] 自检调用失败（非致命）: %v", err)
			break // 自检失败不阻塞，继续执行
		}

		if isCorrect {
			emitStep(AgentStep{
				Turn:     2 + checkRound,
				Action:   "self_check",
				Detail:   "SQL自检通过",
				Duration: time.Since(step3Start),
			})
			utils.Info("✅ [Agent] 自检通过 (第%d轮)", checkRound)
			break
		}

		// 有问题，使用修正后的 SQL
		if fixedSQL != "" {
			utils.Info("🔄 [Agent] 自检发现问题: %v", issues)
			utils.Info("🔄 [Agent] 修正SQL: %s", fixedSQL)
			currentSQL = fixedSQL

			// 替换对话中最后的 assistant 回复
			conversation.AddAssistant(fmt.Sprintf("自检发现问题: %v\n修正后的SQL: %s", issues, fixedSQL))

			emitStep(AgentStep{
				Turn:     2 + checkRound,
				Action:   "self_correction",
				Detail:   fmt.Sprintf("发现 %d 个问题，已修正", len(issues)),
				Duration: time.Since(step3Start),
				Data: map[string]interface{}{
					"issues":   issues,
					"fixed_sql": fixedSQL,
				},
			})
		} else {
			utils.Info("⚠️  [Agent] 自检发现问题但未提供修正SQL，使用原始SQL")
			emitStep(AgentStep{
				Turn:     2 + checkRound,
				Action:   "self_check",
				Detail:   "发现问题但无修正建议，使用原始SQL",
				Duration: time.Since(step3Start),
				Data: map[string]interface{}{
					"issues": issues,
				},
			})
			break
		}
	}

	} // end of self-check (方案2: else branch)

	// ========== 轮次4: 执行验证 ==========
	execStep := len(steps) + 1
	step4Start := time.Now()
	emitStep(AgentStep{Turn: execStep, Action: "execution", Detail: "正在执行SQL..."})

	// 安全检查
	if err := h.executor.ValidateOnly(currentSQL); err != nil {
		utils.Error("❌ [Agent] SQL安全检查失败: %v", err)
		return h.buildFinalResult(question, currentSQL, nil, fmt.Sprintf("SQL安全检查失败: %v", err), steps, start), nil
	}

	// 执行SQL
	execResult, execErr := h.executor.Execute(ctx, currentSQL)
	if execErr != nil {
		// 执行失败，进入错误修正循环
		utils.Warn("⚠️  [Agent] SQL执行失败: %v，尝试修正...", execErr)
		emitStep(AgentStep{
			Turn:     execStep,
			Action:   "execution",
			Detail:   fmt.Sprintf("执行失败: %v", execErr),
			Duration: time.Since(step4Start),
			Data: map[string]interface{}{
				"error": execErr.Error(),
			},
		})

		// 轮次5: 执行错误修正
		correctedSQL, correctErr := h.correctSQLError(ctx, question, currentSQL, execErr.Error(), selectedTables, conversation, callback, &execStep)
		if correctErr != nil {
			utils.Error("❌ [Agent] 修正后仍失败: %v", correctErr)
			return h.buildFinalResult(question, currentSQL, nil, execErr.Error(), steps, start), nil
		}

		currentSQL = correctedSQL
		// 重新执行修正后的SQL
		execResult, execErr = h.executor.Execute(ctx, currentSQL)
		if execErr != nil {
			utils.Error("❌ [Agent] 修正后SQL执行仍失败: %v", execErr)
			return h.buildFinalResult(question, currentSQL, nil, execErr.Error(), steps, start), nil
		}

		utils.Info("✅ [Agent] 修正后SQL执行成功")
	} else {
		emitStep(AgentStep{
			Turn:     execStep,
			Action:   "execution",
			Detail:   fmt.Sprintf("执行成功，返回 %d 行", execResult.Count),
			Duration: time.Since(step4Start),
			Data: map[string]interface{}{
				"row_count": execResult.Count,
			},
		})
	}

	utils.Info("✅ [Agent] 全部完成 | 总耗时: %v", time.Since(start))

	return h.buildFinalResult(question, currentSQL, execResult, "", steps, start), nil
}

// ========== 轮次1: 资源选择 ==========

func (h *AgentQueryHandler) selectResources(ctx context.Context, question string, conversation *llm.Conversation) ([]knowledge.Document, []string, string, error) {
	// 1. 获取表摘要（使用轻量级查询，避免逐表查详情导致131*3=393次SQL）
	tables, err := h.db.GetTableSummaries()
	if err != nil {
		return nil, nil, "", fmt.Errorf("获取表摘要失败: %w", err)
	}

	// 2. 构建资源选择 prompt
	prompt := h.buildResourceSelectionPrompt(question, tables)
	conversation.AddUser(prompt)

	// 3. 调用 LLM
	response, err := h.llmClient.GenerateWithHistory(ctx, conversation.Messages())
	if err != nil {
		return nil, nil, "", fmt.Errorf("LLM资源选择失败: %w", err)
	}

	conversation.AddAssistant(response)

	// 4. 解析响应
	var result struct {
		SelectedDocs   []string `json:"selected_docs"`
		SelectedTables []string `json:"selected_tables"`
		Reasoning      string   `json:"reasoning"`
	}

	jsonStr := extractJSON(response)
	if jsonStr == "" {
		// 解析失败，使用保守策略：不选知识文档，不选表（后续会用空列表处理）
		return nil, nil, "LLM响应解析失败，使用保守策略", nil
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, nil, "JSON解析失败", nil
	}

	// 5. 获取选中的知识文档全文
	var selectedDocs []knowledge.Document
	if h.knowledgeRouter != nil && len(result.SelectedDocs) > 0 {
		docs, err := h.knowledgeRouter.SelectDocumentsFromIDs(result.SelectedDocs)
		if err != nil {
			utils.Warn("⚠️  [Agent] 获取知识文档失败: %v", err)
		} else {
			selectedDocs = docs
		}
	}

	return selectedDocs, result.SelectedTables, result.Reasoning, nil
}

// buildResourceSelectionPrompt 构建资源选择 prompt
func (h *AgentQueryHandler) buildResourceSelectionPrompt(question string, tables []database.TableSummary) string {
	var builder strings.Builder

	builder.WriteString("# 任务\n")
	builder.WriteString("你是数据库查询规划专家。根据用户问题，完成以下两件事：\n\n")

	builder.WriteString("## 1. 选择需要的知识文档\n")
	if h.knowledgeRouter != nil && h.knowledgeRouter.GetIndexCount() > 0 {
		builder.WriteString("以下是可用的知识库文档索引：\n\n")
		for _, idx := range h.knowledgeRouter.GetAllIndices() {
			builder.WriteString(fmt.Sprintf("- [%s] %s", idx.ID, idx.Title))
			if idx.FileSize > 0 {
				builder.WriteString(fmt.Sprintf(" (%dKB)", idx.FileSize/1024))
			}
			builder.WriteString("\n")
			if idx.Summary != "" {
				builder.WriteString(fmt.Sprintf("  %s\n", truncateStr(idx.Summary, 120)))
			}
		}
	} else {
		builder.WriteString("当前无知识库文档可用。\n")
	}

	builder.WriteString("\n## 2. 选择需要的数据库表\n")
	builder.WriteString("以下是可用的数据库表（表名 - 注释 - 数据量）：\n\n")
	for _, table := range tables {
		builder.WriteString(table.Name)
		if table.Comment != "" {
			builder.WriteString(fmt.Sprintf(" - %s", table.Comment))
		}
		if table.RowCount > 0 {
			builder.WriteString(fmt.Sprintf(" (%d)", table.RowCount))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("\n## 用户问题\n")
	builder.WriteString(fmt.Sprintf("%s\n\n", question))

	builder.WriteString("# 输出格式\n")
	builder.WriteString("只返回JSON，不要其他内容：\n")
	builder.WriteString("```json\n")
	builder.WriteString("{\n")
	builder.WriteString("  \"selected_docs\": [\"文档ID1\", \"文档ID2\"],\n")
	builder.WriteString("  \"selected_tables\": [\"表名1\", \"表名2\"],\n")
	builder.WriteString("  \"reasoning\": \"选择理由\"\n")
	builder.WriteString("}\n")
	builder.WriteString("```\n\n")
	builder.WriteString("注意：\n")
	builder.WriteString("- 如果问题不需要知识文档，selected_docs 返回空数组\n")
	builder.WriteString("- 选择表时要考虑可能需要 JOIN 的关联表\n")
	builder.WriteString("- 不确定时宁可多选，不要遗漏\n")

	return builder.String()
}

// ========== 轮次2: SQL 生成 ==========

func (h *AgentQueryHandler) generateSQL(ctx context.Context, question string, docs []knowledge.Document, tables []string, conversation *llm.Conversation, prefetchedSchema string) (string, error) {
	var builder strings.Builder

	// 注入选中的知识文档
	if len(docs) > 0 {
		builder.WriteString("━━━━━ 业务知识库 ━━━━━\n\n")
		for _, doc := range docs {
			builder.WriteString(fmt.Sprintf("### %s\n", doc.Title))
			// 限制每个文档的长度，避免 prompt 过大（方案4: 8000→3000字符）
			content := doc.Content
			if len(content) > 3000 {
				content = content[:3000] + "\n\n... (文档过长，已截断)"
			}
			builder.WriteString(content)
			builder.WriteString("\n\n")
		}
	}

	// 注入选中表的详细 Schema（使用预取结果）
	if len(tables) > 0 && prefetchedSchema != "" {
		builder.WriteString("\n━━━━━ 数据库Schema ━━━━━\n\n")
		builder.WriteString(prefetchedSchema)
	}

	// 构建生成指令
	builder.WriteString("\n━━━━━ SQL生成指令 ━━━━━\n\n")
	builder.WriteString(fmt.Sprintf("用户原始问题: %s\n\n", question))
	builder.WriteString("请根据以上知识库和数据库Schema，生成准确的SQL查询语句。\n")
	builder.WriteString("只返回SQL语句，不要包含任何解释或注释。\n")
	builder.WriteString("确保SQL语法正确且符合MySQL规范。\n")

	conversation.AddUser(builder.String())

	response, err := h.llmClient.GenerateWithHistory(ctx, conversation.Messages())
	if err != nil {
		return "", fmt.Errorf("LLM SQL生成失败: %w", err)
	}

	// 解析 SQL
	sql, err := llm.ParseSQLFromResponse(response)
	if err != nil {
		// 尝试直接使用响应内容
		trimmed := strings.TrimSpace(response)
		if strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") || strings.HasPrefix(strings.ToUpper(trimmed), "WITH") {
			return strings.TrimSuffix(trimmed, ";"), nil
		}
		return "", fmt.Errorf("解析SQL失败: %w", err)
	}

	return sql, nil
}

// ========== 轮次3: 语法+逻辑自检 ==========

func (h *AgentQueryHandler) selfCheckSQL(ctx context.Context, question, sql string, tables []string, _ *llm.Conversation) (bool, []string, string, error) {
	var builder strings.Builder

	builder.WriteString("你是SQL审查专家。请检查以下SQL的正确性。\n\n")
	builder.WriteString(fmt.Sprintf("## 原始问题\n%s\n\n", question))
	builder.WriteString(fmt.Sprintf("## 待审查的SQL\n```sql\n%s\n```\n\n", sql))

	// 提供表Schema供审查
	if len(tables) > 0 {
		schema, err := h.db.FormatTablesForPrompt(tables)
		if err == nil {
			builder.WriteString("## 可用表Schema（用于核对字段名和表名）\n")
			builder.WriteString(schema)
			builder.WriteString("\n")
		}
	}

	builder.WriteString("## 审查要点\n")
	builder.WriteString("1. 表名是否与Schema一致\n")
	builder.WriteString("2. 字段名是否存在于对应表中\n")
	builder.WriteString("3. JOIN条件是否正确（关联字段是否存在）\n")
	builder.WriteString("4. WHERE条件逻辑是否匹配原始问题的意图\n")
	builder.WriteString("5. 是否遗漏了必要的过滤条件\n")
	builder.WriteString("6. 聚合函数(GROUP BY/HAVING)是否正确\n")
	builder.WriteString("7. ORDER BY / LIMIT 是否合理\n\n")

	builder.WriteString("## 输出格式\n")
	builder.WriteString("只返回JSON，不要其他内容：\n")
	builder.WriteString("```json\n")
	builder.WriteString("{\n")
	builder.WriteString("  \"correct\": true,\n")
	builder.WriteString("  \"issues\": [],\n")
	builder.WriteString("  \"fixed_sql\": null\n")
	builder.WriteString("}\n")
	builder.WriteString("```\n")
	builder.WriteString("如果SQL完全正确，correct返回true。如果有问题，correct返回false，issues列出问题，fixed_sql给出修正后的完整SQL。\n")

	// 使用精简对话：system + 仅自检内容，不携带轮次1/2的完整历史
	// 这将prompt从~42KB降到~7KB
	checkConv := llm.NewConversation(h.buildSelfCheckSystemPrompt())
	checkConv.AddUser(builder.String())

	response, err := h.llmClient.GenerateWithHistory(ctx, checkConv.Messages())
	if err != nil {
		return false, nil, "", fmt.Errorf("LLM自检失败: %w", err)
	}

	// 解析结果（不追加到主对话，保持自检独立）
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		// 无法解析，假设正确
		return true, nil, "", nil
	}

	var result struct {
		Correct  bool     `json:"correct"`
		Issues   []string `json:"issues"`
		FixedSQL string   `json:"fixed_sql"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return true, nil, "", nil
	}

	return result.Correct, result.Issues, result.FixedSQL, nil
}

// ========== 轮次5: 执行错误修正 ==========

func (h *AgentQueryHandler) correctSQLError(ctx context.Context, question, sql, errorMsg string, tables []string, conversation *llm.Conversation, callback ProgressCallback, stepCounter *int) (string, error) {
	for round := 1; round <= h.config.MaxSelfCorrect; round++ {
		*stepCounter++
		currentStep := *stepCounter

		stepStart := time.Now()
		emitStep := func(step AgentStep) {
			if callback != nil {
				callback(step)
			}
		}

		emitStep(AgentStep{
			Turn:     currentStep,
			Action:   "error_correction",
			Detail:   fmt.Sprintf("第%d轮修正中...", round),
			Duration: time.Since(stepStart),
		})

		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("━━━━━ SQL执行错误修正 (第%d轮) ━━━━━\n\n", round))
		builder.WriteString(fmt.Sprintf("## 错误信息\n%s\n\n", errorMsg))
		builder.WriteString(fmt.Sprintf("## 失败的SQL\n```sql\n%s\n```\n\n", sql))

		if len(tables) > 0 {
			schema, err := h.db.FormatTablesForPrompt(tables)
			if err == nil {
				builder.WriteString("## 可用表Schema\n")
				builder.WriteString(schema)
				builder.WriteString("\n")
			}
		}

		builder.WriteString("请根据错误信息修正SQL，只返回修正后的SQL语句，不要解释。\n")

		conversation.AddUser(builder.String())

		response, err := h.llmClient.GenerateWithHistory(ctx, conversation.Messages())
		if err != nil {
			return "", fmt.Errorf("LLM修正失败: %w", err)
		}

		conversation.AddAssistant(response)

		correctedSQL, err := llm.ParseSQLFromResponse(response)
		if err != nil {
			trimmed := strings.TrimSpace(response)
			if strings.HasPrefix(strings.ToUpper(trimmed), "SELECT") || strings.HasPrefix(strings.ToUpper(trimmed), "WITH") {
				correctedSQL = strings.TrimSuffix(trimmed, ";")
			} else {
				return "", fmt.Errorf("解析修正SQL失败: %w", err)
			}
		}

		// 验证并执行修正后的SQL
		if err := h.executor.ValidateOnly(correctedSQL); err != nil {
			errorMsg = fmt.Sprintf("安全检查失败: %v", err)
			sql = correctedSQL
			continue
		}

		execResult, execErr := h.executor.Execute(ctx, correctedSQL)
		if execErr != nil {
			errorMsg = execErr.Error()
			sql = correctedSQL
			continue
		}

		// 执行成功
		_ = execResult // 成功了就不需要result，外层会重新执行
		emitStep(AgentStep{
			Turn:     currentStep,
			Action:   "error_correction",
			Detail:   fmt.Sprintf("第%d轮修正成功", round),
			Duration: time.Since(stepStart),
			Data: map[string]interface{}{
				"corrected_sql": correctedSQL,
			},
		})

		return correctedSQL, nil
	}

	return "", fmt.Errorf("经过 %d 轮修正仍失败，最后一次错误: %s", h.config.MaxSelfCorrect, errorMsg)
}

// ========== Prompt 构建 ==========

func (h *AgentQueryHandler) buildSystemPrompt() string {
	return `你是一个专业的数据库查询Agent。你的任务是将用户的自然语言问题转换为准确的SQL查询语句。

工作流程：
1. 分析用户问题，选择需要的知识文档和数据库表
2. 根据选中的知识和Schema生成SQL
3. 自检SQL的语法和逻辑正确性
4. 如果执行失败，根据错误信息修正SQL

注意事项：
- 只使用SELECT查询，绝不使用DELETE、UPDATE、INSERT等修改数据的语句
- 确保列名和表名与Schema中定义的完全一致
- 仔细检查JOIN条件，确保关联字段存在且正确
- 使用适当的WHERE条件精确查询
- 不确定的宁可多选表，不要遗漏`
}

func (h *AgentQueryHandler) buildSelfCheckSystemPrompt() string {
	return `你是SQL审查专家。你的唯一任务是检查SQL的语法和逻辑正确性。
只检查以下内容：表名/字段名是否存在、JOIN条件是否正确、WHERE逻辑是否合理。
只返回JSON格式的检查结果，不要返回其他内容。`
}

// ========== HandleWithSQL 直接使用SQL查询 ==========

func (h *AgentQueryHandler) HandleWithSQL(ctx context.Context, sqlQuery string) (*QueryResult, error) {
	start := time.Now()

	if err := h.executor.ValidateOnly(sqlQuery); err != nil {
		return &QueryResult{
			SQL:    sqlQuery,
			Error:  fmt.Sprintf("SQL安全检查失败: %v", err),
			Metadata: map[string]interface{}{
				"mode": "agent",
			},
		}, err
	}

	execResult, err := h.executor.Execute(ctx, sqlQuery)
	if err != nil {
		return &QueryResult{
			SQL:    sqlQuery,
			Error:  fmt.Sprintf("执行SQL失败: %v", err),
			Metadata: map[string]interface{}{
				"mode": "agent",
			},
		}, err
	}

	return &QueryResult{
		SQL:      sqlQuery,
		Result:   execResult,
		Duration: time.Since(start),
		Metadata: map[string]interface{}{
			"mode": "agent",
		},
	}, nil
}

// SetKnowledge Agent模式不需要手动设置知识库（由Router自动管理）
func (h *AgentQueryHandler) SetKnowledge(docs []knowledge.Document) error {
	// Agent 模式使用 Router，此方法为接口兼容保留
	utils.Warn("⚠️  [Agent] SetKnowledge 在Agent模式下无效，请使用 KnowledgeRouter")
	return nil
}

// ========== 辅助方法 ==========

func (h *AgentQueryHandler) buildErrorResult(question string, err error, steps []AgentStep, start time.Time) *QueryResult {
	return &QueryResult{
		Question: question,
		Error:    err.Error(),
		Duration: time.Since(start),
		Steps:    steps,
		Metadata: map[string]interface{}{
			"mode":  "agent",
			"error": err.Error(),
		},
	}
}

func (h *AgentQueryHandler) buildFinalResult(question, sql string, result *sql.ExecuteResult, errMsg string, steps []AgentStep, start time.Time) *QueryResult {
	qr := &QueryResult{
		Question: question,
		SQL:      sql,
		Result:   result,
		Error:    errMsg,
		Duration: time.Since(start),
		Steps:    steps,
		Metadata: map[string]interface{}{
			"mode":          "agent",
			"total_turns":   len(steps),
			"doc_count":     0,
			"table_count":   0,
		},
	}

	// 从步骤中提取文档数和表数
	for _, step := range steps {
		if step.Action == "resource_selection" && step.Data != nil {
			if docs, ok := step.Data["selected_docs"].([]string); ok {
				qr.Metadata["doc_count"] = len(docs)
			}
			if tables, ok := step.Data["selected_tables"].([]string); ok {
				qr.Metadata["table_count"] = len(tables)
			}
		}
	}

	// 非 verbose 模式下清空 steps
	if !h.config.Verbose {
		qr.Steps = nil
	}

	return qr
}

// extractJSON 从文本中提取 JSON
func extractJSON(text string) string {
	start := strings.Index(text, "```json")
	if start == -1 {
		start = strings.Index(text, "```")
		if start == -1 {
			trimmed := strings.TrimSpace(text)
			if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
				return trimmed
			}
			return ""
		}
		start += 3
	} else {
		start += 7
	}

	end := strings.Index(text[start:], "```")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(text[start : start+end])
}

// docTitles 从文档列表提取标题
func docTitles(docs []knowledge.Document) []string {
	titles := make([]string, len(docs))
	for i, d := range docs {
		titles[i] = d.Title
	}
	return titles
}

// truncateStr 截断字符串
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
