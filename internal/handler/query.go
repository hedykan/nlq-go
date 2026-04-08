package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/llm"
	"github.com/channelwill/nlq/internal/sql"
	"github.com/tmc/langchaingo/llms"
	"gorm.io/gorm"
)

// LLMClient LLM客户端接口
type LLMClient interface {
	GenerateSQL(ctx context.Context, schema, question string) (string, error)
	GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	GenerateWithHistory(ctx context.Context, messages []llms.MessageContent) (string, error)
	SetKnowledge(docs []knowledge.Document)
	GetKnowledge() []knowledge.Document
	SetModel(model string)
	GetModel() string
	IsAvailable() bool
	Type() string
}

// QueryHandlerInterface 查询处理器接口（用于依赖注入和测试）
// 注意：QueryResult 定义在 agent.go 中
type QueryHandlerInterface interface {
	Handle(ctx context.Context, question string) (*QueryResult, error)
	HandleWithSQL(ctx context.Context, sqlQuery string) (*QueryResult, error)
	SetKnowledge(docs []knowledge.Document) error
}

// LegacyQueryHandler 旧版查询处理器（仅CLI工具使用）
type LegacyQueryHandler struct {
	db         *gorm.DB
	parser     *database.SchemaParser
	executor   *sql.Executor
	llmClient  LLMClient
	useRealLLM bool
}

// NewQueryHandler 创建查询处理器（不需要LLM的场景）
func NewQueryHandler(db *gorm.DB) *LegacyQueryHandler {
	return &LegacyQueryHandler{
		db:        db,
		parser:    database.NewSchemaParser(db),
		executor:  sql.NewExecutor(db),
		useRealLLM: false,
		llmClient: nil,
	}
}

// NewQueryHandlerWithLLM 创建带LLM的查询处理器（CLI工具使用）
func NewQueryHandlerWithLLM(db *gorm.DB, apiKey, baseURL, model string, temperature float64, maxTokens int) *LegacyQueryHandler {
	handler := &LegacyQueryHandler{
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

// Handle LegacyQueryHandler 的 Handle 方法（实现 QueryHandlerInterface）
func (h *LegacyQueryHandler) Handle(ctx context.Context, question string) (*QueryResult, error) {
	start := time.Now()

	result := &QueryResult{
		Question: question,
		Metadata: make(map[string]interface{}),
	}

	if h.llmClient == nil || !h.llmClient.IsAvailable() {
		result.Error = "NLQ服务需要配置GLM API Key才能使用"
		result.Metadata["error_type"] = "no_llm_client"
		return result, fmt.Errorf("NLQ服务需要配置GLM API Key才能使用")
	}

	schema, err := h.parser.FormatForPrompt()
	if err != nil {
		result.Error = fmt.Sprintf("解析Schema失败: %v", err)
		return result, err
	}

	generatedSQL, err := h.llmClient.GenerateSQL(ctx, schema, question)
	if err != nil {
		result.Error = fmt.Sprintf("LLM生成SQL失败: %v", err)
		return result, err
	}

	result.SQL = generatedSQL

	if err := h.executor.ValidateOnly(generatedSQL); err != nil {
		result.Error = fmt.Sprintf("SQL验证失败: %v", err)
		return result, err
	}

	execResult, err := h.executor.Execute(ctx, generatedSQL)
	if err != nil {
		result.Error = fmt.Sprintf("执行SQL失败: %v", err)
		return result, err
	}

	result.Result = execResult
	result.Duration = time.Since(start)
	return result, nil
}

// HandleWithSQL LegacyQueryHandler 的 HandleWithSQL 方法
func (h *LegacyQueryHandler) HandleWithSQL(ctx context.Context, sqlQuery string) (*QueryResult, error) {
	start := time.Now()

	result := &QueryResult{
		SQL:      sqlQuery,
		Metadata: make(map[string]interface{}),
	}

	if err := h.executor.ValidateOnly(sqlQuery); err != nil {
		result.Error = fmt.Sprintf("SQL验证失败: %v", err)
		return result, err
	}

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
func (h *LegacyQueryHandler) SetKnowledge(docs []knowledge.Document) error {
	if h.llmClient == nil {
		return fmt.Errorf("LLM客户端未初始化")
	}
	h.llmClient.SetKnowledge(docs)
	return nil
}

// GetSchema 获取数据库Schema（CLI工具使用）
func (h *LegacyQueryHandler) GetSchema() (string, error) {
	return h.parser.FormatForPrompt()
}

// GetTableList 获取表列表（CLI工具使用）
func (h *LegacyQueryHandler) GetTableList() ([]database.TableSchema, error) {
	return h.parser.ParseSchema()
}

// GetTableInfo 获取表信息（CLI工具使用）
func (h *LegacyQueryHandler) GetTableInfo(tableName string) (database.TableSchema, error) {
	return h.parser.ParseTable(tableName)
}
