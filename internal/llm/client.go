package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/pkg/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type GLMClient struct {
	llm               *openai.LLM
	model             string
	timeout           time.Duration
	maxRetries        int
	knowledgeDocs     []knowledge.Document
	knowledgeInjector *knowledge.Injector
}

func NewGLMClient(apiKey, baseURL, model string) (*GLMClient, error) {
	if model == "" {
		model = "glm-4-plus"
	}

	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		baseURL = strings.TrimSuffix(baseURL, "/chat/completions")
	}

	llmInstance, err := openai.New(
		openai.WithToken(apiKey),
		openai.WithBaseURL(baseURL),
		openai.WithModel(model),
	)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	return &GLMClient{
		llm:               llmInstance,
		model:             model,
		timeout:           90 * time.Second,
		maxRetries:        3,
		knowledgeDocs:     []knowledge.Document{},
		knowledgeInjector: knowledge.NewInjector(),
	}, nil
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

func (c *GLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	utils.Info("🤖 [LLM] 开始生成SQL...")
	utils.Debug("🤖 [LLM] 问题: %s", question)

	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		utils.Error("❌ [LLM] 构建Prompt失败: %v", err)
		return "", fmt.Errorf("构建Prompt失败: %w", err)
	}

	utils.Debug("🤖 [LLM] User Prompt长度: %d字符", len(userPrompt))

	if len(c.knowledgeDocs) > 0 {
		userPrompt = c.knowledgeInjector.Inject(userPrompt, c.knowledgeDocs)
		utils.Debug("🤖 [LLM] 已注入知识库文档")
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	utils.Debug("🤖 [LLM] 模型: %s | Temperature: %.1f | MaxTokens: %d", c.model, 0.1, 1000)

	startTime := time.Now()
	resp, err := c.llm.GenerateContent(ctx, messages,
		llms.WithMaxTokens(1000),
		llms.WithTemperature(0.1),
	)
	duration := time.Since(startTime)
	utils.Info("⏱️  [LLM API] 响应时间: %dms", duration.Milliseconds())

	if err != nil {
		utils.Error("❌ [LLM] 调用GLM API失败: %v", err)
		return "", fmt.Errorf("调用GLM API失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		utils.Error("❌ [LLM] GLM API返回空响应")
		return "", errors.New("GLM API返回空响应")
	}

	content := resp.Choices[0].Content
	utils.Info("🤖 [LLM] API返回内容: %s", content)

	if strings.TrimSpace(content) == "" {
		utils.Error("❌ [LLM] GLM API返回空内容")
		return "", &EmptyResponseError{Message: "GLM API返回空内容"}
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

func (c *GLMClient) GenerateSQLWithRetry(ctx context.Context, schema, question string) (string, error) {
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

func (c *GLMClient) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
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
			lastErr = errors.New("GLM API返回空响应")
			continue
		}

		return resp.Choices[0].Content, nil
	}

	return "", fmt.Errorf("重试%d次后仍然失败: %w", c.maxRetries, lastErr)
}

func (c *GLMClient) CorrectSQL(ctx context.Context, sql, errorMsg, schema string) (string, error) {
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
		return "", fmt.Errorf("调用GLM API失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("GLM API返回空响应")
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

func (c *GLMClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (c *GLMClient) SetModel(model string) {
	c.model = model
}

func (c *GLMClient) GetModel() string {
	return c.model
}

func (c *GLMClient) SetKnowledge(docs []knowledge.Document) {
	c.knowledgeDocs = docs
}

func (c *GLMClient) GetKnowledge() []knowledge.Document {
	return c.knowledgeDocs
}

func (c *GLMClient) IsAvailable() bool {
	return c.llm != nil
}

type MockGLMClient struct {
	responses map[string]string
}

func NewMockGLMClient() *MockGLMClient {
	return &MockGLMClient{
		responses: make(map[string]string),
	}
}

func (m *MockGLMClient) SetResponse(question, sql string) {
	m.responses[question] = sql
}

func (m *MockGLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
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

func (m *MockGLMClient) IsAvailable() bool {
	return true
}

type StreamChunk struct {
	Delta    string `json:"delta"`
	Finished bool   `json:"finished"`
	Error    string `json:"error"`
}

func (c *GLMClient) GenerateSQLStream(ctx context.Context, schema, question string, callback func(StreamChunk)) error {
	utils.Info("🤖 [LLM] 开始流式生成SQL...")

	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		return fmt.Errorf("构建Prompt失败: %w", err)
	}

	if len(c.knowledgeDocs) > 0 {
		userPrompt = c.knowledgeInjector.Inject(userPrompt, c.knowledgeDocs)
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
