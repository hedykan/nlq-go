package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/llm"
	"github.com/channelwill/nlq/internal/sql"
	"github.com/channelwill/nlq/pkg/utils"
	"gorm.io/gorm"
)

// QueryHandler 查询处理器
type QueryHandler struct {
	db         *gorm.DB
	parser     *database.SchemaParser
	executor   *sql.Executor
	llmClient  LLMClient
	useRealLLM bool
}

// LLMClient LLM客户端接口
type LLMClient interface {
	GenerateSQL(ctx context.Context, schema, question string) (string, error)
	GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	SetKnowledge(docs []knowledge.Document)
	GetKnowledge() []knowledge.Document
	SetModel(model string)
	GetModel() string
	IsAvailable() bool
	Type() string
}

// NewQueryHandler 创建查询处理器（用于不需要LLM的场景：直接SQL执行、Schema查看）
// 注意：此处理器不包含LLM功能，仅用于SQL执行和Schema查询
// 如需自然语言转SQL功能，请使用 NewQueryHandlerWithLLM
func NewQueryHandler(db *gorm.DB) *QueryHandler {
	return &QueryHandler{
		db:         db,
		parser:     database.NewSchemaParser(db),
		executor:   sql.NewExecutor(db),
		useRealLLM: false,
		llmClient:  nil,
	}
}

// NewQueryHandlerWithLLM 创建带LLM的查询处理器（强制要求API Key）
func NewQueryHandlerWithLLM(db *gorm.DB, apiKey, baseURL, model string, temperature float64, maxTokens int) *QueryHandler {
	handler := &QueryHandler{
		db:         db,
		parser:     database.NewSchemaParser(db),
		executor:   sql.NewExecutor(db),
		useRealLLM: false,
	}

	if apiKey == "" {
		handler.useRealLLM = false
		return handler
	}

	opts := &llm.LLMOptions{
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	client, err := llm.NewLLMClient("zhipuai", apiKey, baseURL, model, opts)
	if err != nil {
		handler.useRealLLM = false
		return handler
	}
	handler.llmClient = client
	handler.useRealLLM = true

	return handler
}

// NewTwoPhaseQueryHandlerWithLLM 创建两阶段查询处理器（推荐用于大型数据库）
func NewTwoPhaseQueryHandlerWithLLM(db *gorm.DB, apiKey, baseURL, model string, temperature float64, maxTokens int) *TwoPhaseQueryHandler {
	parser := database.NewSchemaParser(db)

	if apiKey == "" {
		return &TwoPhaseQueryHandler{
			db:        parser,
			llmClient: nil,
		}
	}

	opts := &llm.LLMOptions{
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	llmClient, err := llm.NewLLMClient("zhipuai", apiKey, baseURL, model, opts)
	if err != nil {
		return &TwoPhaseQueryHandler{
			db:        parser,
			llmClient: nil,
		}
	}

	return NewTwoPhaseQueryHandler(parser, db, llmClient)
}

// QueryResult 查询结果
type QueryResult struct {
	Question           string                 `json:"question"`
	SQL                string                 `json:"sql"`
	Result             *sql.ExecuteResult     `json:"result,omitempty"`
	Error              string                 `json:"error,omitempty"`
	Duration           time.Duration          `json:"duration"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	FieldClarification *FieldClarification    `json:"field_clarification,omitempty"` // 字段澄清信息
}

// Handle 处理自然语言查询（简化版本）
func (h *QueryHandler) Handle(ctx context.Context, question string) (*QueryResult, error) {
	start := time.Now()

	result := &QueryResult{
		Question: question,
		Metadata: make(map[string]interface{}),
	}

	// 1. 首先检查LLM客户端（强制要求，放在最前面以便测试）
	if h.llmClient == nil {
		result.Error = "NLQ服务需要配置GLM API Key才能使用"
		result.Metadata["error_type"] = "no_llm_client"
		return result, fmt.Errorf("NLQ服务需要配置GLM API Key才能使用")
	}

	if !h.llmClient.IsAvailable() {
		result.Error = "LLM客户端不可用，请检查API Key配置"
		result.Metadata["error_type"] = "llm_unavailable"
		return result, fmt.Errorf("LLM客户端不可用，请检查API Key配置")
	}

	// 2. 解析Schema
	utils.Info("🔍 [Handler] 开始解析Schema...")
	schemaStart := time.Now()
	schema, err := h.parser.FormatForPrompt()
	utils.Info("🔍 [Handler] Schema解析完成，耗时: %v，长度: %d", time.Since(schemaStart), len(schema))
	if err != nil {
		result.Error = fmt.Sprintf("解析Schema失败: %v", err)
		return result, err
	}

	// 3. 使用LLM生成SQL（强制要求LLM模式）
	var generatedSQL string

	if !h.llmClient.IsAvailable() {
		result.Error = "LLM客户端不可用，请检查API Key配置"
		result.Metadata["error_type"] = "llm_unavailable"
		return result, fmt.Errorf("LLM客户端不可用，请检查API Key配置")
	}

	// 使用真实的LLM客户端
	result.Metadata["llm_type"] = h.llmClient.Type()
	result.Metadata["llm_model"] = h.llmClient.GetModel()
	result.Metadata["use_real_llm"] = true

	generatedSQL, err = h.llmClient.GenerateSQL(ctx, schema, question)
	if err != nil {
		result.Error = fmt.Sprintf("LLM生成SQL失败: %v", err)
		result.Metadata["error_type"] = "llm_generation_failed"
		return result, err
	}

	result.SQL = generatedSQL
	result.Metadata["sql_generated"] = true

	// 4. 验证SQL
	if err := h.executor.ValidateOnly(generatedSQL); err != nil {
		result.Error = fmt.Sprintf("SQL验证失败: %v", err)
		return result, err
	}

	// 5. 执行SQL
	execResult, err := h.executor.Execute(ctx, generatedSQL)
	if err != nil {
		result.Error = fmt.Sprintf("执行SQL失败: %v", err)
		return result, err
	}

	result.Result = execResult
	result.Duration = time.Since(start)

	return result, nil
}

// HandleWithSQL 直接使用SQL查询
func (h *QueryHandler) HandleWithSQL(ctx context.Context, sqlQuery string) (*QueryResult, error) {
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

// GetSchema 获取数据库Schema
func (h *QueryHandler) GetSchema() (string, error) {
	return h.parser.FormatForPrompt()
}

// GetTableList 获取表列表
func (h *QueryHandler) GetTableList() ([]database.TableSchema, error) {
	return h.parser.ParseSchema()
}

// GetTableInfo 获取表信息
func (h *QueryHandler) GetTableInfo(tableName string) (database.TableSchema, error) {
	return h.parser.ParseTable(tableName)
}

// SetKnowledge 设置知识库文档
func (h *QueryHandler) SetKnowledge(docs []knowledge.Document) error {
	if h.llmClient == nil {
		return fmt.Errorf("LLM客户端未初始化")
	}

	h.llmClient.SetKnowledge(docs)
	return nil
}

// QueryHandlerInterface 查询处理器接口（用于依赖注入和测试）
type QueryHandlerInterface interface {
	Handle(ctx context.Context, question string) (*QueryResult, error)
	HandleWithSQL(ctx context.Context, sqlQuery string) (*QueryResult, error)
	SetKnowledge(docs []knowledge.Document) error
}
