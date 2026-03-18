package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/pkg/utils"
	"github.com/gorilla/mux"
)

// FeedbackHandler 反馈处理器
type FeedbackHandler struct {
	storage   feedback.Storage
	collector *feedback.Collector
	mu        sync.RWMutex // 用于并发写入保护
}

// NewFeedbackHandler 创建反馈处理器
func NewFeedbackHandler(storage feedback.Storage) *FeedbackHandler {
	return &FeedbackHandler{
		storage:   storage,
		collector: feedback.NewCollector(storage),
	}
}

// FeedbackRequest 反馈提交请求
type FeedbackRequest struct {
	QueryID     string `json:"query_id"`     // 查询唯一标识
	IsPositive  bool   `json:"is_positive"`  // true=符合预期, false=不符合预期
	UserComment string `json:"user_comment"` // 用户备注（可选）
	CorrectSQL  string `json:"correct_sql"`  // 正确的SQL（负面反馈时可选）
}

// FeedbackResponse 反馈提交响应
type FeedbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// HandleFeedbackPage 处理反馈页面展示（GET请求）
func (h *FeedbackHandler) HandleFeedbackPage(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	h.setCORSHeaders(w)

	vars := mux.Vars(r)
	queryID := vars["query_id"]

	// 获取查询上下文
	context, err := h.storage.GetQueryContext(queryID)
	if err != nil {
		// 查询上下文不存在或已过期
		h.sendHTMLResponse(w, `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>反馈链接无效</title>
    <style>
        body { font-family: Arial, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; padding: 40px; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #e74c3c; margin-bottom: 20px; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <h1>❌ 反馈链接无效</h1>
        <p>该反馈链接已过期或不存在。</p>
        <p>Query ID: `+queryID+`</p>
    </div>
</body>
</html>`, http.StatusNotFound)
		return
	}

	// 确定反馈类型（从URL路径）
	feedbackType := "positive"
	if r.URL.Path[:len(r.URL.Path)-len(queryID)-1] == "/feedback/negative" {
		feedbackType = "negative"
	}

	// 返回反馈表单页面
	h.sendHTMLResponse(w, h.generateFeedbackHTML(queryID, context, feedbackType), http.StatusOK)
}

// generateFeedbackHTML 生成反馈表单HTML
func (h *FeedbackHandler) generateFeedbackHTML(queryID string, context *feedback.QueryContext, feedbackType string) string {
	emoji := "👍"
	title := "符合预期"
	description := "如果查询结果符合您的预期，请点击确认。"
	bgColor := "#27ae60"
	buttonColor := "#2ecc71"
	isPositive := "true"

	if feedbackType == "negative" {
		emoji = "👎"
		title = "不符合预期"
		description = "如果查询结果不符合您的预期，请告诉我们问题所在。"
		bgColor = "#e74c3c"
		buttonColor = "#c0392b"
		isPositive = "false"
	}

	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>`+title+` - 查询反馈</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, `+bgColor+` 0%, #2c3e50 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            max-width: 500px;
            width: 100%;
            padding: 40px;
        }
        .header { text-align: center; margin-bottom: 30px; }
        .emoji { font-size: 64px; margin-bottom: 15px; }
        h1 { font-size: 28px; color: #2c3e50; margin-bottom: 10px; }
        .description { color: #7f8c8d; font-size: 16px; line-height: 1.5; }
        .query-info {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }
        .query-info h3 { font-size: 14px; color: #95a5a6; margin-bottom: 8px; text-transform: uppercase; }
        .query-info p { color: #34495e; font-size: 15px; line-height: 1.6; word-wrap: break-word; }
        .form-group { margin-bottom: 20px; }
        label { display: block; color: #34495e; font-size: 14px; font-weight: 500; margin-bottom: 8px; }
        textarea {
            width: 100%;
            padding: 12px;
            border: 2px solid #ecf0f1;
            border-radius: 8px;
            font-size: 14px;
            font-family: inherit;
            resize: vertical;
            min-height: 100px;
            transition: border-color 0.3s;
        }
        textarea:focus { outline: none; border-color: `+bgColor+`; }
        .button-group { display: flex; gap: 10px; }
        button {
            flex: 1;
            padding: 14px 24px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
        }
        .btn-submit { background: `+buttonColor+`; color: white; }
        .btn-submit:hover { background: `+bgColor+`; transform: translateY(-2px); box-shadow: 0 4px 12px rgba(0,0,0,0.15); }
        .btn-cancel { background: #ecf0f1; color: #7f8c8d; }
        .btn-cancel:hover { background: #bdc3c7; }
        .loading { display: none; text-align: center; color: #7f8c8d; padding: 20px; }
        .success { display: none; text-align: center; color: #27ae60; padding: 20px; }
        .error { display: none; text-align: center; color: #e74c3c; padding: 20px; }
        .query-id { font-size: 12px; color: #bdc3c7; text-align: center; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="emoji">`+emoji+`</div>
            <h1>`+title+`</h1>
            <p class="description">`+description+`</p>
        </div>
        <div class="query-info">
            <h3>您的问题</h3>
            <p>`+h.escapeHTML(context.Question)+`</p>
        </div>
        <div class="query-info">
            <h3>生成的SQL</h3>
            <p><code>`+h.escapeHTML(context.SQL)+`</code></p>
        </div>
        <form id="feedbackForm">
            <div class="form-group">
                <label for="comment">备注（可选）</label>
                <textarea id="comment" name="comment" placeholder="请告诉我们您的意见..."></textarea>
            </div>
            <div class="button-group">
                <button type="submit" class="btn-submit">提交反馈</button>
                <button type="button" class="btn-cancel" onclick="window.close()">关闭</button>
            </div>
        </form>
        <div class="loading">提交中...</div>
        <div class="success">✅ 反馈已收到，感谢您的反馈！</div>
        <div class="error"></div>
        <div class="query-id">Query ID: `+queryID+`</div>
    </div>
    <script>
        document.getElementById('feedbackForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const form = e.target;
            const loading = document.querySelector('.loading');
            const success = document.querySelector('.success');
            const error = document.querySelector('.error');

            loading.style.display = 'block';
            success.style.display = 'none';
            error.style.display = 'none';

            try {
                const response = await fetch('/feedback/submit', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        query_id: '`+queryID+`',
                        is_positive: `+isPositive+`,
                        user_comment: document.getElementById('comment').value
                    })
                });

                const data = await response.json();

                if (data.success) {
                    form.style.display = 'none';
                    success.style.display = 'block';
                    setTimeout(() => window.close(), 2000);
                } else {
                    throw new Error(data.message || '提交失败');
                }
            } catch (err) {
                error.textContent = '❌ ' + err.message;
                error.style.display = 'block';
            } finally {
                loading.style.display = 'none';
            }
        });
    </script>
</body>
</html>`
}

// escapeHTML 转义HTML特殊字符
func (h *FeedbackHandler) escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(text)
}

// sendHTMLResponse 发送HTML响应
func (h *FeedbackHandler) sendHTMLResponse(w http.ResponseWriter, html string, statusCode int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write([]byte(html))
}

// HandleFeedbackSubmit 处理反馈提交
func (h *FeedbackHandler) HandleFeedbackSubmit(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	h.setCORSHeaders(w)

	// 处理OPTIONS请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST请求
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, "方法不允许", "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var request FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "无效的JSON请求", "invalid_json", http.StatusBadRequest)
		return
	}

	// 转换为内部反馈请求
	internalReq := feedback.FeedbackRequest{
		QueryID:     request.QueryID,
		IsPositive:  request.IsPositive,
		UserComment: request.UserComment,
		CorrectSQL:  request.CorrectSQL,
	}

	// 收集反馈
	if err := h.collector.Collect(internalReq); err != nil {
		h.sendErrorResponse(w, "提交反馈失败: "+err.Error(), "feedback_failed", http.StatusBadRequest)
		return
	}

	// 获取查询上下文（用于写入文件）
	context, _ := h.storage.GetQueryContext(request.QueryID)

	// 异步写入 pending pool 文件
	go func() {
		if err := h.appendToPendingPool(request.QueryID, context, request.IsPositive, request.UserComment, request.CorrectSQL); err != nil {
			// 记录错误但不影响响应
			fmt.Printf("写入 pending pool 失败: %v\n", err)
			return
		}

		// 方案4: 每次反馈后立即尝试自动合并（异步执行）
		// 稍微延迟以确保文件写入完成
		time.Sleep(100 * time.Millisecond)
		h.autoMergeFeedback()
	}()

	// 返回成功响应
	response := FeedbackResponse{
		Success: true,
		Message: "反馈已收到，感谢您的反馈！",
	}

	h.sendJSONResponse(w, response, http.StatusOK)
}

// autoMergeFeedback 自动合并反馈（不依赖HTTP响应）
func (h *FeedbackHandler) autoMergeFeedback() {
	feedbackTypes := []string{"positive", "negative"}

	for _, feedbackType := range feedbackTypes {
		pendingPath := fmt.Sprintf("knowledge/%s/%s_pool.md", feedbackType, feedbackType)

		// 检查 pending pool 是否有内容
		pendingContent, err := os.ReadFile(pendingPath)
		if err != nil || len(strings.TrimSpace(string(pendingContent))) == 0 {
			continue
		}

		// 执行合并
		if _, err := h.mergeFeedbackFile(feedbackType, 0); err != nil {
			fmt.Printf("自动合并 %s 反馈失败: %v\n", feedbackType, err)
		} else {
			fmt.Printf("✅ 自动合并 %s 反馈成功\n", feedbackType)
		}
	}
}

// RecordExecutionError 自动记录执行失败的SQL到负面反馈池
func (h *FeedbackHandler) RecordExecutionError(question, sql, errorMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	queryID := utils.GenerateQueryID() // 使用查询ID生成器

	// 调试：打印SQL长度
	fmt.Printf("🔍 记录错误: Question长度=%d, SQL长度=%d, Error长度=%d\n",
		len(question), len(sql), len(errorMsg))
	if len(sql) < 100 {
		fmt.Printf("🔍 SQL内容: %q\n", sql)
	} else {
		fmt.Printf("🔍 SQL前100字符: %q...\n", sql[:100])
	}

	// 格式化错误条目
	var builder strings.Builder
	builder.WriteString("\n---\n")
	builder.WriteString(fmt.Sprintf("**提交时间**: %s\n", timestamp))
	builder.WriteString(fmt.Sprintf("**QueryID**: %s\n", queryID))
	builder.WriteString(fmt.Sprintf("**问题**: %s\n", question))

	// 检查SQL是否过短（可能是不完整的SQL）
	sqlTooShort := len(sql) < 50 && (strings.HasPrefix(strings.ToUpper(sql), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(sql), "INSERT") ||
		strings.HasPrefix(strings.ToUpper(sql), "UPDATE") ||
		strings.HasPrefix(strings.ToUpper(sql), "DELETE"))

	if sqlTooShort {
		builder.WriteString("**错误的SQL**: " + sql + " (注意：SQL可能不完整，LLM生成时被截断)\n")
	} else if len(sql) > 200 {
		builder.WriteString("**错误的SQL**:\n")
		builder.WriteString("```sql\n")
		builder.WriteString(sql)
		builder.WriteString("\n```\n")
	} else {
		builder.WriteString(fmt.Sprintf("**错误的SQL**: %s\n", sql))
	}

	builder.WriteString(fmt.Sprintf("**错误信息**: %s\n", errorMsg))
	builder.WriteString("\n")

	// 写入 negative pool
	targetFile := "knowledge/negative/negative_pool.md"
	f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("❌ 写入错误反馈池失败: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(builder.String()); err != nil {
		fmt.Printf("❌ 写入错误反馈池失败: %v\n", err)
		return
	}

	fmt.Printf("📝 已记录SQL执行错误到负面反馈池 (QueryID: %s)\n", queryID)

	// 异步触发合并
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.autoMergeFeedback()
	}()
}

// setCORSHeaders 设置CORS头
func (h *FeedbackHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")
}

// sendJSONResponse 发送JSON响应
func (h *FeedbackHandler) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
	}
}

// sendErrorResponse 发送错误响应
func (h *FeedbackHandler) sendErrorResponse(w http.ResponseWriter, message, code string, statusCode int) {
	response := ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	}

	h.sendJSONResponse(w, response, statusCode)
}

// appendToPendingPool 将反馈追加到 pending pool 文件
func (h *FeedbackHandler) appendToPendingPool(queryID string, context *feedback.QueryContext, isPositive bool, userComment, correctSQL string) error {
	if context == nil {
		return fmt.Errorf("查询上下文为空")
	}

	// 确定目标文件
	var targetFile string
	if isPositive {
		targetFile = "knowledge/positive/positive_pool.md"
	} else {
		targetFile = "knowledge/negative/negative_pool.md"
	}

	// 格式化条目
	entry := h.formatFeedbackEntry(queryID, context, isPositive, userComment, correctSQL)

	// 打开文件（追加模式）
	f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	// 写入条目
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// formatFeedbackEntry 格式化反馈条目
func (h *FeedbackHandler) formatFeedbackEntry(queryID string, context *feedback.QueryContext, isPositive bool, userComment, correctSQL string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var builder strings.Builder

	builder.WriteString("\n---\n")
	builder.WriteString(fmt.Sprintf("**提交时间**: %s\n", timestamp))
	builder.WriteString(fmt.Sprintf("**QueryID**: %s\n", queryID))
	builder.WriteString(fmt.Sprintf("**问题**: %s\n", context.Question))
	builder.WriteString(fmt.Sprintf("**生成的SQL**: %s\n", context.SQL))

	if userComment != "" {
		builder.WriteString(fmt.Sprintf("**用户备注**: %s\n", userComment))
	}

	if correctSQL != "" {
		builder.WriteString(fmt.Sprintf("**正确的SQL**: %s\n", correctSQL))
	}

	builder.WriteString("\n")

	return builder.String()
}

// HandleFeedbackMerge 处理反馈合并请求
func (h *FeedbackHandler) HandleFeedbackMerge(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	h.setCORSHeaders(w)

	// 处理OPTIONS请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST请求
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, "方法不允许", "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var request struct {
		Force bool `json:"force"` // 是否强制合并（跳过LLM去重）
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// 允许空请求体
	}

	// 统计合并前的条目数
	statsBefore := h.calculateMergeStats()

	// 合并正面反馈
	posMerged, err := h.mergeFeedbackFile("positive", statsBefore.PositiveExisting)
	if err != nil {
		h.sendErrorResponse(w, "合并正面反馈失败: "+err.Error(), "merge_failed", http.StatusInternalServerError)
		return
	}

	// 合并负面反馈
	negMerged, err := h.mergeFeedbackFile("negative", statsBefore.NegativeExisting)
	if err != nil {
		h.sendErrorResponse(w, "合并负面反馈失败: "+err.Error(), "merge_failed", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	response := struct {
		Success      bool   `json:"success"`
		Message      string `json:"message"`
		PendingCount int    `json:"pending_count"`
		MergedCount  int    `json:"merged_count"`
		TotalCount   int    `json:"total_count"`
	}{
		Success:      true,
		Message:      "反馈已成功合并到知识库",
		PendingCount: statsBefore.PendingCount,
		MergedCount:  posMerged + negMerged,
		TotalCount:   statsBefore.PendingCount + posMerged + negMerged,
	}

	h.sendJSONResponse(w, response, http.StatusOK)

	// 重新加载知识库（可选）
	// TODO: 触发知识库重新加载
}

// mergeFeedbackFile 合并单个反馈文件
func (h *FeedbackHandler) mergeFeedbackFile(feedbackType string, existingCount int) (int, error) {
	pendingPath := fmt.Sprintf("knowledge/%s/%s_pool.md", feedbackType, feedbackType)
	targetPath := fmt.Sprintf("knowledge/%s/%s_examples.md", feedbackType, feedbackType)

	// 读取 pending pool
	pendingContent, err := os.ReadFile(pendingPath)
	if err != nil {
		return 0, fmt.Errorf("读取pending文件失败: %w", err)
	}

	// 解析 pending 条目
	pendingEntries := h.parsePendingEntries(string(pendingContent))
	if len(pendingEntries) == 0 {
		return 0, nil
	}

	// 读取现有 examples
	existingContent, _ := os.ReadFile(targetPath)
	existingEntriesMap := h.parseExistingEntriesToMap(string(existingContent))
	existingEntriesList := h.parseExistingEntriesToList(string(existingContent))

	// 去重合并 - 先添加原有条目，再添加新条目
	var mergedEntries []string

	// 首先添加所有原有条目（保留原有内容）
	mergedEntries = append(mergedEntries, existingEntriesList...)

	// 添加新的非重复条目
	addedCount := 0
	for _, pending := range pendingEntries {
		// 提取问题用于去重
		question := h.extractQuestion(pending)
		questionKey := strings.TrimSpace(question)

		// 检查是否已存在
		if _, exists := existingEntriesMap[questionKey]; !exists {
			formattedEntry := h.convertPendingToStandardFormat(pending, feedbackType)
			mergedEntries = append(mergedEntries, formattedEntry)
			addedCount++
		}
	}

	// 写入目标文件
	output := h.formatExamplesDocument(mergedEntries, feedbackType)
	if err := os.WriteFile(targetPath, []byte(output), 0644); err != nil {
		return 0, fmt.Errorf("写入目标文件失败: %w", err)
	}

	// 清空 pending pool
	if err := h.resetPendingPool(pendingPath, feedbackType); err != nil {
		return 0, fmt.Errorf("清空pending文件失败: %w", err)
	}

	return addedCount, nil
}

// parsePendingEntries 解析 pending pool 条目
func (h *FeedbackHandler) parsePendingEntries(content string) []string {
	if content == "" {
		return []string{}
	}

	parts := strings.Split(content, "---")
	var entries []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		// 跳过空内容和标题
		if part == "" || strings.HasPrefix(part, "# ") {
			continue
		}
		// 跳过纯注释行（以 * 开头且不包含其他内容）
		trimmed := strings.TrimPrefix(part, "*")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" || trimmed == "（待合并反馈）*" || strings.HasPrefix(part, "*") && !strings.Contains(part, "**") {
			continue
		}
		// 包含 QueryID 的才是有效条目
		if strings.Contains(part, "**QueryID**:") {
			entries = append(entries, part)
		}
	}

	return entries
}

// parseExistingEntries 解析现有 examples 条目
func (h *FeedbackHandler) parseExistingEntries(content string) []map[string]string {
	if content == "" {
		return []map[string]string{}
	}

	parts := strings.Split(content, "## ")
	var entries []map[string]string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "# 正面查询示例" || part == "# 需要避免的错误模式" {
			continue
		}

		entry := make(map[string]string)
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "**问题**:") {
				entry["question"] = strings.TrimPrefix(line, "**问题**:")
			} else if strings.HasPrefix(line, "**SQL**:") {
				entry["sql"] = strings.TrimPrefix(line, "**SQL**:")
			}
		}

		if entry["question"] != "" {
			entries = append(entries, entry)
		}
	}

	return entries
}

// isDuplicate 检查是否重复
func (h *FeedbackHandler) isDuplicate(pending string, existing []map[string]string) bool {
	lines := strings.Split(pending, "\n")
	question := ""
	sql := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**问题**:") {
			question = strings.TrimSpace(strings.TrimPrefix(line, "**问题**:"))
		} else if strings.HasPrefix(line, "**生成的SQL**:") {
			sql = strings.TrimSpace(strings.TrimPrefix(line, "**生成的SQL**:"))
		}
	}

	// 检查是否与现有条目重复
	for _, exist := range existing {
		if exist["question"] == question || exist["sql"] == sql {
			return true
		}
	}

	return false
}

// convertToStandardFormat 转换为标准格式
func (h *FeedbackHandler) convertToStandardFormat(pending string, feedbackType string) string {
	lines := strings.Split(pending, "\n")
	var builder strings.Builder

	if feedbackType == "positive" {
		builder.WriteString("## 示例\n")
	} else {
		builder.WriteString("## 错误模式\n")
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**QueryID**:") || strings.HasPrefix(line, "**提交时间**:") {
			continue // 跳过元数据
		}
		if strings.HasPrefix(line, "**生成的SQL**:") {
			if feedbackType == "negative" {
				builder.WriteString(fmt.Sprintf("**错误SQL**: %s\n", strings.TrimPrefix(line, "**生成的SQL**:")))
			} else {
				builder.WriteString(fmt.Sprintf("**SQL**: %s\n", strings.TrimPrefix(line, "**生成的SQL**:")))
			}
		} else if line != "" {
			builder.WriteString(line + "\n")
		}
	}

	return builder.String()
}

// formatExamplesDocument 格式化完整的 examples 文档
func (h *FeedbackHandler) formatExamplesDocument(entries []string, feedbackType string) string {
	var builder strings.Builder

	if feedbackType == "positive" {
		builder.WriteString("# 正面查询示例\n\n")
		builder.WriteString("本文档包含用户确认符合预期的查询示例，用于提高SQL生成准确性。\n\n")
	} else {
		builder.WriteString("# 需要避免的错误模式\n\n")
		builder.WriteString("本文档包含用户确认不符合预期的查询示例，用于避免常见错误。\n\n")
	}

	builder.WriteString("---\n\n")

	for i, entry := range entries {
		builder.WriteString(entry)
		if i < len(entries)-1 {
			builder.WriteString("\n---\n\n")
		}
	}

	return builder.String()
}

// HandleFeedbackStats 获取反馈统计信息
func (h *FeedbackHandler) HandleFeedbackStats(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "方法不允许", "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.calculateMergeStats()

	response := struct {
		PositivePending int `json:"positive_pending"`
		PositiveExisting int `json:"positive_existing"`
		NegativePending int `json:"negative_pending"`
		NegativeExisting int `json:"negative_existing"`
	}{
		PositivePending: stats.PositivePending,
		PositiveExisting: stats.PositiveExisting,
		NegativePending: stats.NegativePending,
		NegativeExisting: stats.NegativeExisting,
	}

	h.sendJSONResponse(w, response, http.StatusOK)
}

// mergeStats 合并统计信息
type mergeStats struct {
	PendingCount int
	MergedCount  int
	PositivePending int
	PositiveExisting int
	NegativePending int
	NegativeExisting int
}

// calculateMergeStats 计算合并统计
func (h *FeedbackHandler) calculateMergeStats() *mergeStats {
	stats := &mergeStats{}

	// 读取文件
	posPool, _ := os.ReadFile("knowledge/positive/positive_pool.md")
	posExist, _ := os.ReadFile("knowledge/positive/positive_examples.md")
	negPool, _ := os.ReadFile("knowledge/negative/negative_pool.md")
	negExist, _ := os.ReadFile("knowledge/negative/negative_examples.md")

	// 计算条目数（按 "---" 计数，因为我们的格式用 --- 分隔条目）
	poolContent := string(posPool)
	existContent := string(posExist)

	// 对于 pending pool，条目数 = --- 出现次数 - 1（开头的）
	stats.PositivePending = strings.Count(poolContent, "---") - 1
	if stats.PositivePending < 0 {
		stats.PositivePending = 0
	}

	// 对于 examples，条目数 = ## 出现次数
	stats.PositiveExisting = strings.Count(existContent, "## ")

	poolContent = string(negPool)
	existContent = string(negExist)
	stats.NegativePending = strings.Count(poolContent, "---") - 1
	if stats.NegativePending < 0 {
		stats.NegativePending = 0
	}
	stats.NegativeExisting = strings.Count(existContent, "## ")

	stats.PendingCount = stats.PositivePending + stats.NegativePending
	stats.MergedCount = stats.PositiveExisting + stats.NegativeExisting

	return stats
}

// parseExistingEntriesToMap 解析现有条目为map（用于去重）
func (h *FeedbackHandler) parseExistingEntriesToMap(content string) map[string]bool {
	result := make(map[string]bool)

	if content == "" {
		return result
	}

	// 按行扫描，提取所有问题
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**问题**:") {
			question := strings.TrimSpace(strings.TrimPrefix(line, "**问题**:"))
			result[question] = true
		}
	}

	return result
}

// parseExistingEntriesToList 解析现有条目为列表（保留完整格式，但不包含分隔符）
func (h *FeedbackHandler) parseExistingEntriesToList(content string) []string {
	if content == "" {
		return []string{}
	}

	parts := strings.Split(content, "---")
	var entries []string

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// 跳过空内容
		if part == "" {
			continue
		}

		// 跳过标题行（# 开头的）
		if strings.HasPrefix(part, "# ") {
			continue
		}

		// 跳过描述文本
		if strings.HasPrefix(part, "本文档包含") {
			continue
		}

		// 包含 "## 示例" 或 "**问题**:" 的才是有效条目
		if strings.Contains(part, "## 示例") || strings.Contains(part, "**问题**:") {
			entries = append(entries, part)
		}
	}

	return entries
}

// formatEntryFromMap 从map格式化条目（保留兼容性）
func (h *FeedbackHandler) formatEntryFromMap(entry map[string]string, feedbackType string) string {
	var builder strings.Builder
	builder.WriteString("## 示例\n")
	if question, ok := entry["question"]; ok {
		builder.WriteString(fmt.Sprintf("**问题**: %s\n", question))
	}
	if sql, ok := entry["sql"]; ok {
		if feedbackType == "negative" {
			builder.WriteString(fmt.Sprintf("**错误SQL**: %s\n", sql))
		} else {
			builder.WriteString(fmt.Sprintf("**SQL**: %s\n", sql))
		}
	}
	return builder.String()
}

// extractQuestion 从pending条目中提取问题
func (h *FeedbackHandler) extractQuestion(pending string) string {
	lines := strings.Split(pending, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "**问题**:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "**问题**:"))
		}
	}
	return ""
}

// convertPendingToStandardFormat 转换pending格式为标准格式
func (h *FeedbackHandler) convertPendingToStandardFormat(pending string, feedbackType string) string {
	var builder strings.Builder
	builder.WriteString("## 示例\n")

	lines := strings.Split(pending, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过元数据
		if strings.HasPrefix(line, "**QueryID**:") || strings.HasPrefix(line, "**提交时间**:") {
			continue
		}
		// 转换SQL字段名
		if strings.HasPrefix(line, "**生成的SQL**:") {
			sql := strings.TrimSpace(strings.TrimPrefix(line, "**生成的SQL**:"))
			if feedbackType == "negative" {
				builder.WriteString("**错误SQL**: " + sql + "\n")
			} else {
				builder.WriteString("**SQL**: " + sql + "\n")
			}
		} else if strings.HasPrefix(line, "**用户备注**:") {
			note := strings.TrimSpace(strings.TrimPrefix(line, "**用户备注**:"))
			builder.WriteString("**说明**: " + note + "\n")
		} else if line != "" && !strings.HasPrefix(line, "---") {
			builder.WriteString(line + "\n")
		}
	}

	return builder.String()
}

// resetPendingPool 重置pending pool文件
func (h *FeedbackHandler) resetPendingPool(path, feedbackType string) error {
	title := "正面"
	if feedbackType == "negative" {
		title = "负面"
	}

	content := fmt.Sprintf("# %s反馈待合并池\n\n本文档暂存用户的%s反馈，等待合并到知识库中。\n\n---\n\n*（待合并反馈）*\n", title, feedbackType)
	return os.WriteFile(path, []byte(content), 0644)
}
