package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/pkg/utils"
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
func NewGLMClient(apiKey, baseURL, model string) *GLMClient {
	// 如果未指定模型，使用默认值
	if model == "" {
		model = "glm-4-plus"
	}

	return &GLMClient{
		apiKey:            apiKey,
		baseURL:           baseURL,
		model:             model,
		timeout:           90 * time.Second, // 增加到90秒（GLM-4.7响应较慢）
		maxRetries:        3,
		httpClient:        &http.Client{Timeout: 90 * time.Second},
		knowledgeDocs:     []knowledge.Document{},
		knowledgeInjector: knowledge.NewInjector(),
	}
}

// GLMRequest GLM API请求结构
type GLMRequest struct {
	Model       string       `json:"model"`
	Messages    []GLMMessage `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream,omitempty"` // 是否启用流式响应
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

// RateLimitError 限流错误
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("API限流错误: %s (建议等待: %v)", e.Message, e.RetryAfter)
}

// EmptyResponseError 空响应错误
type EmptyResponseError struct {
	Message string
}

func (e *EmptyResponseError) Error() string {
	return fmt.Sprintf("API返回空响应: %s", e.Message)
}

// GenerateSQL 生成SQL查询
func (c *GLMClient) GenerateSQL(ctx context.Context, schema, question string) (string, error) {
	utils.Info("🤖 [LLM] 开始生成SQL...")
	utils.Debug("🤖 [LLM] 问题: %s", question)

	// 构建基础Prompt
	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		utils.Error("❌ [LLM] 构建Prompt失败: %v", err)
		return "", fmt.Errorf("构建Prompt失败: %w", err)
	}

	utils.Debug("🤖 [LLM] User Prompt长度: %d字符", len(userPrompt))

	// 如果有知识库文档，注入知识库内容
	if len(c.knowledgeDocs) > 0 {
		userPrompt = c.knowledgeInjector.Inject(userPrompt, c.knowledgeDocs)
		utils.Debug("🤖 [LLM] 已注入知识库文档")
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

	utils.Debug("🤖 [LLM] 模型: %s | Temperature: %.1f | MaxTokens: %d", c.model, 0.1, 1000)

	// 调用API
	response, err := c.callAPI(ctx, request)
	if err != nil {
		utils.Error("❌ [LLM] 调用GLM API失败: %v", err)
		return "", fmt.Errorf("调用GLM API失败: %w", err)
	}

	// 检查错误
	if response.Error != nil {
		utils.Error("❌ [LLM] GLM API返回错误: %s", response.Error.Message)
		return "", fmt.Errorf("GLM API错误: %s", response.Error.Message)
	}

	// 提取SQL
	if len(response.Choices) == 0 {
		utils.Error("❌ [LLM] GLM API返回空响应（Choices为空）")
		return "", errors.New("GLM API返回空响应")
	}

	content := response.Choices[0].Message.Content
	utils.Info("🤖 [LLM] API返回内容: %s", content)
	utils.Debug("🤖 [LLM] 内容长度: %d字符", len(content))

	// 检查空响应（GLM API有时候会返回200但content为空）
	if strings.TrimSpace(content) == "" {
		utils.Error("❌ [LLM] GLM API返回空内容（200状态码但content为空）")
		utils.Error("❌ [LLM] 这可能是API限流或临时问题，建议重试")
		return "", &EmptyResponseError{Message: "GLM API返回空内容"}
	}

	// 解析SQL
	sql, err := ParseSQLFromResponse(content)
	if err != nil {
		utils.Error("❌ [LLM] 解析SQL失败: %v", err)
		utils.Error("❌ [LLM] 原始内容: %s", content)
		return "", fmt.Errorf("解析SQL失败: %w", err)
	}

	utils.Info("✅ [LLM] 成功生成SQL: %s", sql)

	// 验证SQL
	if !ValidateSQLQuery(sql) {
		utils.Error("❌ [LLM] 生成的SQL不安全或无效: %s", sql)
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
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 判断是否是限流错误
			var rateLimitErr *RateLimitError
			if errors.As(lastErr, &rateLimitErr) {
				// 使用API返回的Retry-After时间
				waitTime := rateLimitErr.RetryAfter
				fmt.Printf("⏳ API限流，等待 %v 后重试 (尝试 %d/%d)...\n", waitTime, attempt, c.maxRetries)

				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(waitTime):
				}
			} else {
				// 其他错误使用指数退避
				waitTime := time.Duration(attempt) * 2 * time.Second
				fmt.Printf("⚠️ 请求失败，%v 后重试 (尝试 %d/%d): %v\n", waitTime, attempt, c.maxRetries, lastErr)

				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(waitTime):
				}
			}
		}

		// 构建请求
		request := GLMRequest{
			Model: c.model,
			Messages: []GLMMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			Temperature: 0.1,
			MaxTokens:   4096, // 增加到4096以支持更复杂的SQL生成
		}

		// 调用API
		response, err := c.callAPI(ctx, request)
		if err != nil {
			lastErr = err
			continue
		}

		// 检查错误
		if response.Error != nil {
			lastErr = fmt.Errorf("GLM API错误: %s", response.Error.Message)
			continue
		}

		// 提取内容
		if len(response.Choices) == 0 {
			lastErr = errors.New("GLM API返回空响应")
			continue
		}

		content := response.Choices[0].Message.Content
		return content, nil
	}

	return "", fmt.Errorf("重试%d次后仍然失败: %w", c.maxRetries, lastErr)
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

	// 记录API调用详情
	utils.Debug("🔍 [LLM API] 调用API: %s", endpoint)
	utils.Debug("🔍 [LLM API] 请求体: %s", string(reqBody))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	utils.Debug("🔍 [LLM API] 请求头: Content-Type=%s, Authorization=Bearer %s...",
		req.Header.Get("Content-Type"),
		func() string {
			if len(c.apiKey) > 10 {
				return c.apiKey[:10]
			}
			return c.apiKey
		}())

	// 记录请求开始时间
	startTime := time.Now()

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		utils.Error("❌ [LLM API] HTTP请求失败: %v", err)
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 记录响应时间
	duration := time.Since(startTime)
	utils.Info("⏱️  [LLM API] 响应时间: %dms | 状态码: %d", duration.Milliseconds(), resp.StatusCode)

	// 检查响应状态
	if resp.StatusCode == http.StatusTooManyRequests {
		// 429 限流错误，尝试读取Retry-After头
		retryAfter := c.parseRetryAfter(resp.Header.Get("Retry-After"))

		// 读取错误消息
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := "请求过于频繁，请稍后再试"
		if len(bodyBytes) > 0 {
			var glmErr GLMResponse
			if json.Unmarshal(bodyBytes, &glmErr) == nil && glmErr.Error != nil {
				errorMsg = glmErr.Error.Message
			}
		}

		utils.Warn("⚠️  [LLM API] 限流错误: %s (建议等待: %v)", errorMsg, retryAfter)

		return nil, &RateLimitError{
			RetryAfter: retryAfter,
			Message:    errorMsg,
		}
	}

	if resp.StatusCode != http.StatusOK {
		// 读取错误响应体
		bodyBytes, _ := io.ReadAll(resp.Body)
		utils.Error("❌ [LLM API] API返回错误: 状态码=%d, 响应=%s", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	// 读取并记录响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.Error("❌ [LLM API] 读取响应体失败: %v", err)
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	utils.Debug("🔍 [LLM API] 响应体: %s", string(bodyBytes))

	// 解析响应
	var result GLMResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		utils.Error("❌ [LLM API] 解析响应JSON失败: %v | 原始响应: %s", err, string(bodyBytes))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查响应结构
	if len(result.Choices) == 0 {
		utils.Warn("⚠️  [LLM API] API返回空Choices数组")
		if result.Error != nil {
			utils.Error("❌ [LLM API] API返回错误: %s (代码: %s, 类型: %s)",
				result.Error.Message, result.Error.Code, result.Error.Type)
		}
	} else {
		utils.Info("✅ [LLM API] API调用成功 | Choices数量: %d", len(result.Choices))
	}

	return &result, nil
}

// parseRetryAfter 解析Retry-After头部
func (c *GLMClient) parseRetryAfter(retryAfter string) time.Duration {
	if retryAfter == "" {
		// 默认等待5秒
		return 5 * time.Second
	}

	// 尝试解析为秒数
	var seconds int
	if _, err := fmt.Sscanf(retryAfter, "%d", &seconds); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// 尝试解析为HTTP-date
	if t, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(t)
	}

	// 默认等待5秒
	return 5 * time.Second
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

// ==================== 流式生成支持 ====================

// StreamChunk 流式响应数据块
type StreamChunk struct {
	Delta      string `json:"delta"`       // 增量文本
	Finished   bool   `json:"finished"`    // 是否完成
	Error      string `json:"error"`       // 错误信息
}

// GenerateSQLStream 流式生成SQL查询
func (c *GLMClient) GenerateSQLStream(ctx context.Context, schema, question string, callback func(StreamChunk)) error {
	utils.Info("🤖 [LLM] 开始流式生成SQL...")

	// 构建基础Prompt
	systemPrompt := GenerateSystemPrompt()
	userPrompt, err := BuildSQLGenerationPrompt(schema, question)
	if err != nil {
		return fmt.Errorf("构建Prompt失败: %w", err)
	}

	// 如果有知识库文档，注入知识库内容
	if len(c.knowledgeDocs) > 0 {
		userPrompt = c.knowledgeInjector.Inject(userPrompt, c.knowledgeDocs)
	}

	// 构建请求（启用流式响应）
	request := GLMRequest{
		Model: c.model,
		Messages: []GLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		MaxTokens:   1000,
		Stream:      true, // 启用流式响应
	}

	// 发送流式请求
	chunkChan, errChan := c.callAPIStream(ctx, request)

	// 处理流式响应
	var fullContent string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-chunkChan:
			if !ok {
				// 流结束
				utils.Info("✅ [LLM] 流式生成完成")

				// 解析完整的SQL
				sql, err := ParseSQLFromResponse(fullContent)
				if err != nil {
					callback(StreamChunk{Error: fmt.Sprintf("解析SQL失败: %v", err), Finished: true})
					return fmt.Errorf("解析SQL失败: %w", err)
				}

				// 验证SQL
				if !ValidateSQLQuery(sql) {
					callback(StreamChunk{Error: "生成的SQL不安全或无效", Finished: true})
					return errors.New("生成的SQL不安全或无效")
				}

				// 发送完成事件（包含最终SQL）
				callback(StreamChunk{Delta: sql, Finished: true})
				return nil
			}

			// 累积内容
			fullContent += chunk.Delta
			utils.Debug("📝 [LLM] 收到chunk: %s", chunk.Delta)

			// 发送进度回调
			callback(chunk)

		case err, ok := <-errChan:
			if !ok {
				continue // channel已关闭，继续等待chunk
			}
			utils.Error("❌ [LLM] 流式API错误: %v", err)
			callback(StreamChunk{Error: err.Error(), Finished: true})
			return err
		}
	}
}

// callAPIStream 调用GLM流式API
func (c *GLMClient) callAPIStream(ctx context.Context, request GLMRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// 序列化请求
		reqBody, err := json.Marshal(request)
		if err != nil {
			errChan <- fmt.Errorf("序列化请求失败: %w", err)
			return
		}

		// 构建HTTP请求
		endpoint := fmt.Sprintf("%s/chat/completions", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(reqBody)))
		if err != nil {
			errChan <- fmt.Errorf("创建HTTP请求失败: %w", err)
			return
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		req.Header.Set("Accept", "text/event-stream")

		utils.Info("🔍 [LLM API] 开始流式调用...")

		// 发送请求
		resp, err := c.httpClient.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("发送HTTP请求失败: %w", err)
			return
		}
		defer resp.Body.Close()

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		// 读取流式响应
		scanner := newSSELineScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// SSE格式: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// "[DONE]" 表示流结束
			if data == "[DONE]" {
				break
			}

			// 解析chunk数据
			var streamResp struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				utils.Warn("⚠️ [LLM API] 解析chunk失败: %v, data: %s", err, data)
				continue
			}

			// 提取增量内容
			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta.Content
				if delta != "" {
					chunkChan <- StreamChunk{Delta: delta}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("读取流式响应失败: %w", err)
		}
	}()

	return chunkChan, errChan
}

// ==================== SSE行扫描器 ====================

// sseLineScanner SSE行扫描器
type sseLineScanner struct {
	reader *bufio.Reader
	line   string
	err    error
}

// newSSELineScanner 创建SSE行扫描器
func newSSELineScanner(r io.Reader) *sseLineScanner {
	return &sseLineScanner{
		reader: bufio.NewReader(r),
	}
}

// Scan 读取下一行
func (s *sseLineScanner) Scan() bool {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		s.err = err
		return false
	}
	s.line = strings.TrimSuffix(line, "\n")
	return true
}

// Text 返回当前行内容
func (s *sseLineScanner) Text() string {
	return s.line
}

// Err 返回错误
func (s *sseLineScanner) Err() error {
	return s.err
}
