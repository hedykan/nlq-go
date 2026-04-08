package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/pkg/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderZhipuAI   LLMProvider = "zhipuai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderOllama    LLMProvider = "ollama"
	ProviderAzure     LLMProvider = "azure"
	ProviderMiniMax   LLMProvider = "minimax"
)

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

type LLMOptions struct {
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	MaxRetries  int
}

func DefaultLLMOptions() *LLMOptions {
	return &LLMOptions{
		Temperature: 0.0,
		MaxTokens:   2048,
		Timeout:     90 * time.Second,
		MaxRetries:  3,
	}
}

type OpenAIClient struct {
	provider          LLMProvider
	llm               *openai.LLM
	model             string
	timeout           time.Duration
	maxRetries        int
	temperature       float64
	maxTokens         int
	knowledgeDocs     []knowledge.Document
}

func (p *OpenAIClient) Type() string { return string(p.provider) }

func NewOpenAIClient(provider, apiKey, baseURL, model string, temperature float64, maxTokens int) (*OpenAIClient, error) {
	if model == "" {
		model = "glm-4-plus"
	}
	if temperature == 0 {
		temperature = 0.0
	}
	if maxTokens == 0 {
		maxTokens = 2048
	}

	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
	}

	httpClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	normalizedBaseURL := normalizeBaseURL(provider, baseURL)

	llmInstance, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL(normalizedBaseURL),
		openai.WithModel(model),
		openai.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	return &OpenAIClient{
		provider:          LLMProvider(provider),
		llm:               llmInstance,
		model:             model,
		timeout:           90 * time.Second,
		maxRetries:        3,
		temperature:       temperature,
		maxTokens:         maxTokens,
		knowledgeDocs:     []knowledge.Document{},
	}, nil
}

func normalizeBaseURL(provider, baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")

	switch strings.ToLower(provider) {
	case "zhipuai":
		if strings.Contains(baseURL, "bigmodel.cn") {
			return baseURL
		}
		return baseURL
	case "minimax":
		if !strings.Contains(baseURL, "minimax") && !strings.Contains(baseURL, "api.minimaxi.com") {
			return "https://api.minimaxi.com"
		}
		return baseURL
	case "openai", "azure":
		return baseURL
	}
	return baseURL
}

func NewLLMClient(provider, apiKey, baseURL, model string, opts ...*LLMOptions) (LLMClient, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	options := DefaultLLMOptions()
	if len(opts) > 0 && opts[0] != nil {
		options = opts[0]
		if options.Temperature == 0 {
			options.Temperature = 0.0
		}
		if options.MaxTokens == 0 {
			options.MaxTokens = 2048
		}
	}

	normalizedBaseURL := normalizeBaseURL(provider, baseURL)

	switch strings.ToLower(provider) {
	case "openai", "zhipuai", "azure", "minimax":
		return NewOpenAIClient(provider, apiKey, normalizedBaseURL, model, options.Temperature, options.MaxTokens)
	default:
		return NewOpenAIClient("openai", apiKey, normalizedBaseURL, model, options.Temperature, options.MaxTokens)
	}
}

type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("API限流错误: %s (建议等待: %v)", e.Message, e.RetryAfter)
}

type EmptyResponseError struct {
	Message string
}

func (e *EmptyResponseError) Error() string {
	return fmt.Sprintf("API返回空响应: %s", e.Message)
}

func (c *OpenAIClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	utils.Info("🤖 [LLM] 开始生成SQL...")
	utils.Debug("🤖 [LLM] 问题: %s", question)

	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		utils.Error("❌ [LLM] 构建Prompt失败: %v", err)
		return "", fmt.Errorf("构建Prompt失败: %w", err)
	}

	utils.Debug("🤖 [LLM] User Prompt长度: %d字符", len(userPrompt))

	// 注意：知识库注入已由 Agent 模式的 KnowledgeRouter 接管
	// 此处保留 knowledgeDocs 仅用于接口兼容

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	callOptions := []llms.CallOption{
		llms.WithMaxTokens(c.maxTokens),
		llms.WithTemperature(c.temperature),
	}

	utils.Debug("🤖 [LLM] 模型: %s | Temperature: %.1f | MaxTokens: %d", c.model, c.temperature, c.maxTokens)

	startTime := time.Now()
	resp, err := c.llm.GenerateContent(ctx, messages, callOptions...)
	duration := time.Since(startTime)
	utils.Info("⏱️  [LLM API] 响应时间: %dms", duration.Milliseconds())

	if err != nil {
		utils.Error("❌ [LLM] 调用LLM API失败: %v", err)
		return "", fmt.Errorf("调用LLM API失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		utils.Error("❌ [LLM] LLM API返回空响应")
		return "", errors.New("LLM API返回空响应")
	}

	content := resp.Choices[0].Content
	utils.Info("🤖 [LLM] API返回内容: %s", content)

	if strings.TrimSpace(content) == "" {
		utils.Error("❌ [LLM] LLM API返回空内容")
		return "", &EmptyResponseError{Message: "LLM API返回空内容"}
	}

	sql, err := ParseSQLFromResponse(content)
	if err != nil {
		utils.Error("❌ [LLM] 解析SQL失败: %v", err)
		utils.Error("❌ [LLM] 原始内容: %s", content)
		return "", fmt.Errorf("解析SQL失败: %w", err)
	}

	utils.Info("✅ [LLM] 成功生成SQL: %s", sql)

	if !ValidateSQLQuery(sql) {
		utils.Error("❌ [LLM] 生成的SQL不安全或无效: %s", sql)
		return "", errors.New("生成的SQL不安全或无效")
	}

	return sql, nil
}

func (c *OpenAIClient) GenerateSQLWithRetry(ctx context.Context, schema, question string) (string, error) {
	var lastErr error

	for i := 0; i < c.maxRetries; i++ {
		if i > 0 {
			waitTime := time.Duration(i+1) * 2 * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(waitTime):
			}
		}

		sql, err := c.GenerateSQL(ctx, schema, question)
		if err == nil {
			return sql, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("重试%d次后仍然失败: %w", c.maxRetries, lastErr)
}

func (c *OpenAIClient) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt) * 2 * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(waitTime):
			}
		}

		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
		}

		resp, err := c.llm.GenerateContent(ctx, messages,
			llms.WithMaxTokens(4096),
			llms.WithTemperature(0.1),
		)
		if err != nil {
			lastErr = err
			continue
		}

		if len(resp.Choices) == 0 {
			lastErr = errors.New("LLM API返回空响应")
			continue
		}

		return resp.Choices[0].Content, nil
	}

	return "", fmt.Errorf("重试%d次后仍然失败: %w", c.maxRetries, lastErr)
}

func (c *OpenAIClient) GenerateWithHistory(ctx context.Context, messages []llms.MessageContent) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("消息列表不能为空")
	}

	startTime := time.Now()
	resp, err := c.llm.GenerateContent(ctx, messages,
		llms.WithMaxTokens(c.maxTokens),
		llms.WithTemperature(c.temperature),
	)
	duration := time.Since(startTime)
	utils.Info("⏱️  [LLM API] GenerateWithHistory 响应时间: %dms, 消息数: %d", duration.Milliseconds(), len(messages))

	if err != nil {
		return "", fmt.Errorf("GenerateWithHistory 调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("GenerateWithHistory 返回空响应")
	}

	return resp.Choices[0].Content, nil
}

func (c *OpenAIClient) CorrectSQL(ctx context.Context, sql, errorMsg, schema string) (string, error) {
	prompt, err := BuildSQLCorrectionPrompt(sql, errorMsg, schema)
	if err != nil {
		return "", fmt.Errorf("构建修正Prompt失败: %w", err)
	}

	systemPrompt := "你是一个SQL专家，擅长修正SQL查询错误。"

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := c.llm.GenerateContent(ctx, messages,
		llms.WithMaxTokens(1000),
		llms.WithTemperature(0.1),
	)
	if err != nil {
		return "", fmt.Errorf("调用LLM API失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("LLM API返回空响应")
	}

	content := resp.Choices[0].Content

	correctedSQL, err := ParseSQLFromResponse(content)
	if err != nil {
		return "", fmt.Errorf("解析修正后的SQL失败: %w", err)
	}

	if !ValidateSQLQuery(correctedSQL) {
		return "", errors.New("修正后的SQL仍然不安全")
	}

	return correctedSQL, nil
}

func (c *OpenAIClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (c *OpenAIClient) SetModel(model string) {
	c.model = model
}

func (c *OpenAIClient) GetModel() string {
	return c.model
}

func (c *OpenAIClient) SetKnowledge(docs []knowledge.Document) {
	c.knowledgeDocs = docs
}

func (c *OpenAIClient) GetKnowledge() []knowledge.Document {
	return c.knowledgeDocs
}

func (c *OpenAIClient) IsAvailable() bool {
	return c.llm != nil
}

type MockLLMClient struct {
	provider  string
	model     string
	responses map[string]string
	knowledge []knowledge.Document
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		provider:  "mock",
		model:     "mock-model",
		responses: make(map[string]string),
	}
}

func (m *MockLLMClient) Type() string { return m.provider }

func (m *MockLLMClient) SetResponse(question, sql string) {
	m.responses[question] = sql
}

func (m *MockLLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	if sql, ok := m.responses[question]; ok {
		return sql, nil
	}

	question = strings.ToLower(question)

	if strings.Contains(question, "boom_user") {
		if strings.Contains(question, "多少") || strings.Contains(question, "数量") || strings.Contains(question, "总数") {
			return "SELECT COUNT(*) as total FROM boom_user", nil
		}
		return "SELECT * FROM boom_user LIMIT 100", nil
	}

	if strings.Contains(question, "boom_customer") {
		if strings.Contains(question, "多少") || strings.Contains(question, "数量") {
			return "SELECT COUNT(*) as total FROM boom_customer", nil
		}
		return "SELECT * FROM boom_customer LIMIT 100", nil
	}

	if strings.Contains(question, "boom_order") {
		if strings.Contains(question, "多少") || strings.Contains(question, "数量") {
			return "SELECT COUNT(*) as total FROM boom_order_paid_water", nil
		}
		return "SELECT * FROM boom_order_paid_water LIMIT 100", nil
	}

	return "", fmt.Errorf("无法理解问题: %s", question)
}

func (m *MockLLMClient) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "", errors.New("mock client does not support GenerateContent")
}

func (m *MockLLMClient) GenerateWithHistory(ctx context.Context, messages []llms.MessageContent) (string, error) {
	return "", errors.New("mock client does not support GenerateWithHistory")
}

func (m *MockLLMClient) SetKnowledge(docs []knowledge.Document) {
	m.knowledge = docs
}

func (m *MockLLMClient) GetKnowledge() []knowledge.Document {
	return m.knowledge
}

func (m *MockLLMClient) SetModel(model string) {
	m.model = model
}

func (m *MockLLMClient) GetModel() string {
	return m.model
}

func (m *MockLLMClient) IsAvailable() bool {
	return true
}

type StreamChunk struct {
	Delta    string `json:"delta"`
	Finished bool   `json:"finished"`
	Error    string `json:"error"`
}

func (c *OpenAIClient) GenerateSQLStream(ctx context.Context, schema, question string, callback func(StreamChunk)) error {
	utils.Info("🤖 [LLM] 开始流式生成SQL...")

	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		return fmt.Errorf("构建Prompt失败: %w", err)
	}

	if len(c.knowledgeDocs) > 0 {
		// 知识库注入已由 Agent 模式的 KnowledgeRouter 接管
		_ = c.knowledgeDocs // 保留引用以避免 unused 警告
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	var fullContent string
	callbackCalled := false

	_, err = c.llm.GenerateContent(ctx, messages,
		llms.WithMaxTokens(1000),
		llms.WithTemperature(0.1),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			fullContent += string(chunk)
			if !callbackCalled {
				callbackCalled = true
			}
			callback(StreamChunk{Delta: string(chunk)})
			return nil
		}),
	)

	if err != nil {
		callback(StreamChunk{Error: err.Error(), Finished: true})
		return err
	}

	sql, err := ParseSQLFromResponse(fullContent)
	if err != nil {
		callback(StreamChunk{Error: fmt.Sprintf("解析SQL失败: %v", err), Finished: true})
		return fmt.Errorf("解析SQL失败: %w", err)
	}

	if !ValidateSQLQuery(sql) {
		callback(StreamChunk{Error: "生成的SQL不安全或无效", Finished: true})
		return errors.New("生成的SQL不安全或无效")
	}

	callback(StreamChunk{Delta: sql, Finished: true})
	return nil
}
