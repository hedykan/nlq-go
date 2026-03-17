package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
)

// QueryHandler 查询请求处理器
type QueryHandler struct {
	queryHandler QueryHandlerInterface
	enableCORS   bool
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
		queryHandler: queryHandler,
		enableCORS:   true,
	}
}

// NewQueryHandlerWithHandler 直接使用QueryHandler创建（与NewQueryHandler功能相同）
func NewQueryHandlerWithHandler(h QueryHandlerInterface) *QueryHandler {
	return &QueryHandler{
		queryHandler: h,
		enableCORS:   true,
	}
}

// HandleQuery 处理自然语言查询
func (h *QueryHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
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

	// 加载知识库（如果指定）
	if request.KnowledgeBase != "" {
		if err := h.loadKnowledgeBase(request.KnowledgeBase); err != nil && request.Verbose {
			// 记录警告但不阻止查询
			h.sendWarning(w, "加载知识库失败: "+err.Error())
		}
	}

	// 执行查询
	ctx := r.Context()
	result, err := h.queryHandler.Handle(ctx, request.Question)

	if err != nil {
		h.sendErrorResponse(w, "查询失败: "+err.Error(), "query_failed", http.StatusInternalServerError)
		return
	}

	// 构建响应
	response := QueryResponse{
		Success:    true,
		Question:   result.Question,
		SQL:        result.SQL,
		DurationMs: result.Duration.Milliseconds(),
		Metadata:   result.Metadata,
	}

	// 转换结果数据（检查Result是否为nil）
	if result.Result != nil {
		response.Count = result.Result.Count
		if len(result.Result.Rows) > 0 {
			response.Result = result.Result.Rows
		}
	}

	// 发送JSON响应
	h.sendJSONResponse(w, response, http.StatusOK)
}

// HandleSQL 处理SQL查询
func (h *QueryHandler) HandleSQL(w http.ResponseWriter, r *http.Request) {
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

	// 执行SQL查询
	ctx := r.Context()
	result, err := h.queryHandler.HandleWithSQL(ctx, request.SQL)

	if err != nil {
		h.sendErrorResponse(w, "SQL执行失败: "+err.Error(), "sql_failed", http.StatusInternalServerError)
		return
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
