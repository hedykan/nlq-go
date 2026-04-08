package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/pkg/utils"
)

const (
	// 上下文键
	requestIDKey    = "requestID"
	httpLoggerKey   = "httpLogger"
	requestTimeKey  = "startTime"
)

// QueryHandler 查询请求处理器
type QueryHandler struct {
	queryHandler     QueryHandlerInterface
	enableCORS       bool
	feedbackStorage  feedback.Storage
	feedbackHandler  *FeedbackHandler // 用于自动记录错误
}

// QueryHandlerInterface 查询处理器接口（用于测试）
type QueryHandlerInterface interface {
	Handle(ctx context.Context, question string) (*handler.QueryResult, error)
	HandleWithSQL(ctx context.Context, sqlQuery string) (*handler.QueryResult, error)
	SetKnowledge(docs []knowledge.Document) error
}

// QueryRequest 自然语言查询请求
type QueryRequest struct {
	Question      string `json:"question"`
	KnowledgeBase string `json:"knowledge_base,omitempty"`
	Verbose       bool   `json:"verbose,omitempty"`
}

// SQLRequest SQL查询请求
type SQLRequest struct {
	SQL string `json:"sql"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Success    bool                   `json:"success"`
	Question   string                 `json:"question,omitempty"`
	SQL        string                 `json:"sql"`
	Result     []map[string]interface{} `json:"result,omitempty"`
	Count      int                    `json:"count"`
	DurationMs int64                  `json:"duration_ms"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Steps      []handler.AgentStep    `json:"steps,omitempty"` // Agent推理步骤

	// 新增：反馈相关字段
	QueryID  string        `json:"query_id,omitempty"`  // 查询唯一标识
	Feedback *FeedbackLinks `json:"feedback,omitempty"` // 反馈链接
}

// FeedbackLinks 反馈链接结构
type FeedbackLinks struct {
	PositiveURL string `json:"positive_url"` // 符合预期链接
	NegativeURL string `json:"negative_url"` // 不符合预期链接
	ExpiresAt   int64  `json:"expires_at"`   // 链接过期时间(Unix时间戳)
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// StatusResponse 状态响应
type StatusResponse struct {
	Service  string `json:"service"`
	Version  string `json:"version"`
	Status   string `json:"status"`
	Uptime   string `json:"uptime"`
	Database string `json:"database"`
}

// NewQueryHandler 创建查询处理器
func NewQueryHandler(queryHandler QueryHandlerInterface) *QueryHandler {
	return &QueryHandler{
		queryHandler:    queryHandler,
		enableCORS:      true,
		feedbackStorage: feedback.NewMockStorage(),
	}
}

// NewQueryHandlerWithHandler 直接使用QueryHandler创建（与NewQueryHandler功能相同）
func NewQueryHandlerWithHandler(h QueryHandlerInterface) *QueryHandler {
	return &QueryHandler{
		queryHandler:    h,
		enableCORS:      true,
		feedbackStorage: feedback.NewMockStorage(),
	}
}

// NewQueryHandlerWithFeedback 创建带反馈存储的查询处理器
func NewQueryHandlerWithFeedback(h QueryHandlerInterface, storage feedback.Storage, feedbackHandler *FeedbackHandler) *QueryHandler {
	return &QueryHandler{
		queryHandler:     h,
		enableCORS:       true,
		feedbackStorage:  storage,
		feedbackHandler:  feedbackHandler,
	}
}

// HandleQuery 处理自然语言查询
func (h *QueryHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	// 获取日志记录器和请求ID
	var httpLogger *utils.HTTPRequestLogger
	var requestID string

	if logger, ok := r.Context().Value(httpLoggerKey).(*utils.HTTPRequestLogger); ok {
		httpLogger = logger
	}
	if rid, ok := r.Context().Value(requestIDKey).(string); ok {
		requestID = rid
	}

	// 阶段1: 请求解析
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "1.请求解析", map[string]interface{}{
			"handler": "QueryHandler.HandleQuery",
		})
	}

	// 设置CORS头
	h.setCORSHeaders(w)

	// 处理OPTIONS请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析请求
	var request QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "无效的JSON请求", "invalid_json", http.StatusBadRequest)
		return
	}

	// 验证问题
	if request.Question == "" {
		h.sendErrorResponse(w, "问题不能为空", "empty_question", http.StatusBadRequest)
		return
	}

	// 记录解析结果
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "2.请求验证", map[string]interface{}{
			"question":       request.Question,
			"knowledge_base": request.KnowledgeBase,
			"verbose":        request.Verbose,
		})
	}

	// 加载知识库（如果指定）
	if request.KnowledgeBase != "" {
		if err := h.loadKnowledgeBase(request.KnowledgeBase); err != nil && request.Verbose {
			// 记录警告但不阻止查询
			h.sendWarning(w, "加载知识库失败: "+err.Error())
		}
	}

	// 记录开始执行查询
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "3.开始查询", map[string]interface{}{
			"question": request.Question,
		})
	}

	// 执行查询
	ctx := r.Context()
	result, err := h.queryHandler.Handle(ctx, request.Question)

	if err != nil {
		// 记录查询失败
		if httpLogger != nil && requestID != "" {
			httpLogger.LogRequestStage(requestID, "4.查询失败", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// 如果执行错误，自动记录到负面反馈池
		if h.feedbackHandler != nil {
			errorMsg := err.Error()
			failedSQL := ""

			// 尝试从错误信息中提取SQL
			if strings.Contains(errorMsg, "SQL:") {
				parts := strings.SplitN(errorMsg, "SQL:", 2)
				if len(parts) > 1 {
					// SQL: 后面的内容可能包含多行，需要完整保留
					failedSQL = strings.TrimLeft(parts[1], " \n")
					// 只取第一行作为错误描述的其余部分
					if idx := strings.Index(failedSQL, "\n"); idx != -1 {
						// 如果SQL中有换行，只取到错误信息为止
						errorMsg = strings.TrimSpace(parts[0])
					} else {
						// SQL单行，分开错误和SQL
						errorMsg = strings.TrimSpace(parts[0])
					}
				}
			}

			// 如果提取到了SQL，记录错误
			if failedSQL != "" {
				go h.feedbackHandler.RecordExecutionError(request.Question, failedSQL, errorMsg)
			}
		}
		h.sendErrorResponse(w, "查询失败: "+err.Error(), "query_failed", http.StatusInternalServerError)
		return
	}

	// 记录查询成功
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "4.查询成功", map[string]interface{}{
			"sql":         result.SQL,
			"duration_ms": result.Duration.Milliseconds(),
			"row_count":   func() int { if result.Result != nil { return result.Result.Count } else { return 0 } }(),
		})
	}

	// 生成QueryID
	queryID := utils.GenerateQueryID()

	// 存储查询上下文（供反馈使用）
	queryContext := &feedback.QueryContext{
		QueryID:   queryID,
		Question:  result.Question,
		SQL:       result.SQL,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if result.Result != nil {
		queryContext.Result = result.Result.Rows
	}
	_ = h.feedbackStorage.SetQueryContext(queryContext) // 忽略错误，不阻止查询

	// 构建响应
	response := QueryResponse{
		Success:    true,
		Question:   result.Question,
		SQL:        result.SQL,
		DurationMs: result.Duration.Milliseconds(),
		Metadata:   result.Metadata,
		QueryID:    queryID,
		Feedback:   h.generateFeedbackLinks(queryID, r),
		Steps:      result.Steps,
	}

	// 转换结果数据（检查Result是否为nil）
	if result.Result != nil {
		response.Count = result.Result.Count
		if len(result.Result.Rows) > 0 {
			response.Result = result.Result.Rows
		}
	}

	// 记录响应构建
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "5.构建响应", map[string]interface{}{
			"query_id": queryID,
			"success":  true,
		})
	}

	// 发送JSON响应
	h.sendJSONResponse(w, response, http.StatusOK)
}

// HandleSQL 处理SQL查询
func (h *QueryHandler) HandleSQL(w http.ResponseWriter, r *http.Request) {
	// 获取日志记录器和请求ID
	var httpLogger *utils.HTTPRequestLogger
	var requestID string

	if logger, ok := r.Context().Value(httpLoggerKey).(*utils.HTTPRequestLogger); ok {
		httpLogger = logger
	}
	if rid, ok := r.Context().Value(requestIDKey).(string); ok {
		requestID = rid
	}

	// 阶段1: 请求解析
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "1.请求解析", map[string]interface{}{
			"handler": "QueryHandler.HandleSQL",
		})
	}

	// 设置CORS头
	h.setCORSHeaders(w)

	// 处理OPTIONS请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析请求
	var request SQLRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "无效的JSON请求", "invalid_json", http.StatusBadRequest)
		return
	}

	// 验证SQL
	if request.SQL == "" {
		h.sendErrorResponse(w, "SQL不能为空", "empty_sql", http.StatusBadRequest)
		return
	}

	// 记录解析结果
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "2.请求验证", map[string]interface{}{
			"sql": request.SQL,
		})
	}

	// 执行SQL查询
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "3.开始执行SQL", nil)
	}

	ctx := r.Context()
	result, err := h.queryHandler.HandleWithSQL(ctx, request.SQL)

	if err != nil {
		// 记录执行失败
		if httpLogger != nil && requestID != "" {
			httpLogger.LogRequestStage(requestID, "4.执行失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
		h.sendErrorResponse(w, "SQL执行失败: "+err.Error(), "sql_failed", http.StatusInternalServerError)
		return
	}

	// 记录执行成功
	if httpLogger != nil && requestID != "" {
		httpLogger.LogRequestStage(requestID, "4.执行成功", map[string]interface{}{
			"duration_ms": result.Duration.Milliseconds(),
			"row_count":   result.Result.Count,
		})
	}

	// 构建响应
	response := QueryResponse{
		Success:    true,
		SQL:        result.SQL,
		Count:      result.Result.Count,
		DurationMs: result.Duration.Milliseconds(),
		Metadata:   result.Metadata,
	}

	// 转换结果数据
	if result.Result != nil && len(result.Result.Rows) > 0 {
		response.Result = result.Result.Rows
	}

	h.sendJSONResponse(w, response, http.StatusOK)
}

// HealthCheck 健康检查
func (h *QueryHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	h.sendJSONResponse(w, response, http.StatusOK)
}

// Status 服务状态
func (h *QueryHandler) Status(w http.ResponseWriter, r *http.Request) {
	response := StatusResponse{
		Service:  "nlq",
		Version:  "1.0.0",
		Status:   "running",
		Uptime:   "unknown", // TODO: 实现uptime计算
		Database: "connected", // TODO: 实际检查数据库连接
	}

	h.sendJSONResponse(w, response, http.StatusOK)
}

// HandleRecordError 处理来自CLI的错误记录请求
func (h *QueryHandler) HandleRecordError(w http.ResponseWriter, r *http.Request) {
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
		Question  string `json:"question"`
		SQL       string `json:"sql"`
		ErrorMsg  string `json:"error_msg"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendErrorResponse(w, "无效的JSON请求", "invalid_json", http.StatusBadRequest)
		return
	}

	// 记录错误
	if h.feedbackHandler != nil && request.Question != "" && request.SQL != "" {
		go h.feedbackHandler.RecordExecutionError(request.Question, request.SQL, request.ErrorMsg)
	}

	// 返回成功（异步处理，不等待完成）
	h.sendJSONResponse(w, map[string]bool{"success": true}, http.StatusOK)
}

// setCORSHeaders 设置CORS头
func (h *QueryHandler) setCORSHeaders(w http.ResponseWriter) {
	if h.enableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
	}

	w.Header().Set("Content-Type", "application/json")
}

// sendJSONResponse 发送JSON响应
func (h *QueryHandler) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
	}
}

// sendErrorResponse 发送错误响应
func (h *QueryHandler) sendErrorResponse(w http.ResponseWriter, message, code string, statusCode int) {
	response := ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	}

	h.sendJSONResponse(w, response, statusCode)
}

// sendWarning 发送警告响应（用于非致命错误）
func (h *QueryHandler) sendWarning(w http.ResponseWriter, message string) {
	w.Header().Set("X-NLQ-Warning", message)
}

// loadKnowledgeBase 加载知识库
func (h *QueryHandler) loadKnowledgeBase(knowledgePath string) error {
	loader := knowledge.NewLoader()
	docs, err := loader.LoadFromDirectory(knowledgePath)
	if err != nil {
		return err
	}

	if err := h.queryHandler.SetKnowledge(docs); err != nil {
		return err
	}

	return nil
}

// getLocalIP 获取本机IP地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	for _, addr := range addrs {
		// 检查是否是IP地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				// 返回第一个找到的非回环IPv4地址
				return ipnet.IP.String()
			}
		}
	}

	return "localhost"
}

// getServerPort 获取服务器端口
func getServerPort() string {
	return "8080"
}

// generateFeedbackLinks 生成反馈链接（使用本机IP）
func (h *QueryHandler) generateFeedbackLinks(queryID string, r *http.Request) *FeedbackLinks {
	// 使用本机IP和固定端口
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	localIP := getLocalIP()
	port := getServerPort()
	host := fmt.Sprintf("%s:%s", localIP, port)

	baseURL := fmt.Sprintf("%s://%s", scheme, host)

	return &FeedbackLinks{
		PositiveURL: fmt.Sprintf("%s/feedback/positive/%s", baseURL, queryID),
		NegativeURL: fmt.Sprintf("%s/feedback/negative/%s", baseURL, queryID),
		ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
	}
}
