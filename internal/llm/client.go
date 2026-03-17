package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/knowledge"
)

// GLMClient GLM4.7客户端
type GLMClient struct {
	apiKey        string
	baseURL       string
	model         string
	timeout       time.Duration
	maxRetries    int
	httpClient    *http.Client
	knowledgeDocs []knowledge.Document // 知识库文档
	knowledgeInjector *knowledge.Injector // 知识库注入器
}

// NewGLMClient 创建GLM客户端
func NewGLMClient(apiKey, baseURL string) *GLMClient {
	return &GLMClient{
		apiKey:            apiKey,
		baseURL:           baseURL,
		model:             "glm-4-plus",
		timeout:           30 * time.Second,
		maxRetries:        3,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
		knowledgeDocs:     []knowledge.Document{},
		knowledgeInjector: knowledge.NewInjector(),
	}
}

// GLMRequest GLM API请求结构
type GLMRequest struct {
	Model    string          `json:"model"`
	Messages []GLMMessage    `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
}

// GLMMessage GLM消息结构
type GLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GLMResponse GLM API响应结构
type GLMResponse struct {
	Choices []GLMChoice `json:"choices"`
	Error   *GLMError   `json:"error,omitempty"`
}

// GLMChoice GLM选择
type GLMChoice struct {
	Message GLMMessage `json:"message"`
}

// GLMError GLM错误
type GLMError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// GenerateSQL 生成SQL查询
func (c *GLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	// 构建基础Prompt
	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		return "", fmt.Errorf("构建Prompt失败: %w", err)
	}

	// 如果有知识库文档，注入知识库内容
	if len(c.knowledgeDocs) > 0 {
		userPrompt = c.knowledgeInjector.Inject(userPrompt, c.knowledgeDocs)
	}

	// 构建请求
	request := GLMRequest{
		Model: c.model,
		Messages: []GLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   1000,
	}

	// 调用API
	response, err := c.callAPI(ctx, request)
	if err != nil {
		return "", fmt.Errorf("调用GLM API失败: %w", err)
	}

	// 检查错误
	if response.Error != nil {
		return "", fmt.Errorf("GLM API错误: %s", response.Error.Message)
	}

	// 提取SQL
	if len(response.Choices) == 0 {
		return "", errors.New("GLM API返回空响应")
	}

	content := response.Choices[0].Message.Content

	// 解析SQL
	sql, err := ParseSQLFromResponse(content)
	if err != nil {
		return "", fmt.Errorf("解析SQL失败: %w", err)
	}

	// 验证SQL
	if !ValidateSQLQuery(sql) {
		return "", errors.New("生成的SQL不安全或无效")
	}

	return sql, nil
}

// GenerateSQLWithRetry 使用重试机制生成SQL
func (c *GLMClient) GenerateSQLWithRetry(ctx context.Context, schema, question string) (string, error) {
	var lastErr error

	for i := 0; i < c.maxRetries; i++ {
		if i > 0 {
			// 指数退避
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

// GenerateContent 生成内容（不进行SQL解析，用于两步法等场景）
func (c *GLMClient) GenerateContent(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// 构建请求
	request := GLMRequest{
		Model: c.model,
		Messages: []GLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   2000,
	}

	// 调用API
	response, err := c.callAPI(ctx, request)
	if err != nil {
		return "", fmt.Errorf("调用GLM API失败: %w", err)
	}

	// 检查错误
	if response.Error != nil {
		return "", fmt.Errorf("GLM API错误: %s", response.Error.Message)
	}

	// 提取内容
	if len(response.Choices) == 0 {
		return "", errors.New("GLM API返回空响应")
	}

	content := response.Choices[0].Message.Content
	return content, nil
}


// CorrectSQL 修正错误的SQL
func (c *GLMClient) CorrectSQL(ctx context.Context, sql, errorMsg, schema string) (string, error) {
	// 构建修正Prompt
	prompt, err := BuildSQLCorrectionPrompt(sql, errorMsg, schema)
	if err != nil {
		return "", fmt.Errorf("构建修正Prompt失败: %w", err)
	}

	systemPrompt := "你是一个SQL专家，擅长修正SQL查询错误。"

	// 构建请求
	request := GLMRequest{
		Model: c.model,
		Messages: []GLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   1000,
	}

	// 调用API
	response, err := c.callAPI(ctx, request)
	if err != nil {
		return "", fmt.Errorf("调用GLM API失败: %w", err)
	}

	// 检查错误
	if response.Error != nil {
		return "", fmt.Errorf("GLM API错误: %s", response.Error.Message)
	}

	// 提取修正后的SQL
	if len(response.Choices) == 0 {
		return "", errors.New("GLM API返回空响应")
	}

	content := response.Choices[0].Message.Content

	// 解析修正后的SQL
	correctedSQL, err := ParseSQLFromResponse(content)
	if err != nil {
		return "", fmt.Errorf("解析修正后的SQL失败: %w", err)
	}

	// 验证SQL
	if !ValidateSQLQuery(correctedSQL) {
		return "", errors.New("修正后的SQL仍然不安全")
	}

	return correctedSQL, nil
}

// callAPI 调用GLM API
func (c *GLMClient) callAPI(ctx context.Context, request GLMRequest) (*GLMResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 构建HTTP请求
	endpoint := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var result GLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// SetTimeout 设置超时时间
func (c *GLMClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}

// SetModel 设置模型名称
func (c *GLMClient) SetModel(model string) {
	c.model = model
}

// GetModel 获取当前模型
func (c *GLMClient) GetModel() string {
	return c.model
}

// SetKnowledge 设置知识库文档
func (c *GLMClient) SetKnowledge(docs []knowledge.Document) {
	c.knowledgeDocs = docs
}

// GetKnowledge 获取知识库文档
func (c *GLMClient) GetKnowledge() []knowledge.Document {
	return c.knowledgeDocs
}

// IsAvailable 检查客户端是否可用
func (c *GLMClient) IsAvailable() bool {
	return c.apiKey != "" && c.apiKey != "your-api-key-here" && c.baseURL != ""
}

// MockGLMClient Mock GLM客户端（用于测试和演示）
type MockGLMClient struct {
	responses map[string]string
}

// NewMockGLMClient 创建Mock客户端
func NewMockGLMClient() *MockGLMClient {
	return &MockGLMClient{
		responses: make(map[string]string),
	}
}

// SetResponse 设置Mock响应
func (m *MockGLMClient) SetResponse(question, sql string) {
	m.responses[question] = sql
}

// GenerateSQL 实现接口
func (m *MockGLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	// 检查是否有预设的响应
	if sql, ok := m.responses[question]; ok {
		return sql, nil
	}

	// 否则返回默认的简单SQL（基于问题模式匹配）
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

	// 默认返回错误
	return "", fmt.Errorf("无法理解问题: %s", question)
}

// IsAvailable 检查Mock客户端是否可用
func (m *MockGLMClient) IsAvailable() bool {
	return true
}
