package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/pkg/utils"
)

// SSEHandler SSE流式传输处理器
type SSEHandler struct {
	queryHandler    QueryHandlerInterface
	feedbackStorage feedback.Storage
}

// NewSSEHandler 创建SSE处理器
func NewSSEHandler(queryHandler QueryHandlerInterface, feedbackStorage feedback.Storage) *SSEHandler {
	return &SSEHandler{
		queryHandler:    queryHandler,
		feedbackStorage: feedbackStorage,
	}
}

// SSEEvent SSE事件
type SSEEvent struct {
	Event string `json:"event"` // thinking, progress, result, error
	Data  string `json:"data"`  // JSON数据
}

// HandleStreamQuery 处理流式查询请求
func (h *SSEHandler) HandleStreamQuery(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // 禁用Nginx缓冲

	// 尝试获取flusher（有些HTTP包装器可能不支持）
	flusher, _ := w.(http.Flusher)
	if flusher == nil {
		utils.Warn("⚠️ [SSE] Flusher不可用，流式传输可能不稳定")
	}

	// 解析问题
	question := r.URL.Query().Get("question")
	if question == "" {
		h.sendSSEError(w, flusher, "问题不能为空")
		return
	}

	utils.Info("🤖 [SSE] 开始流式查询: %s", question)

	// 发送开始事件
	h.sendSSEEvent(w, flusher, "start", map[string]interface{}{
		"message": "开始处理查询",
		"time":    time.Now().Format(time.RFC3339),
	})

	// 1. 表选择阶段
	h.sendSSEEvent(w, flusher, "thinking", map[string]interface{}{
		"phase":    "table_selection",
		"stage":    "表选择",
		"message":  "正在分析数据库表结构...",
		"progress": 10,
	})

	// 模拟表选择处理（实际应该调用LLM进行表选择）
	time.Sleep(500 * time.Millisecond)

	h.sendSSEEvent(w, flusher, "thinking", map[string]interface{}{
		"phase":    "table_selection",
		"stage":    "表选择完成",
		"message":  "已确定需要查询的表",
		"progress": 20,
	})

	// 2. SQL生成阶段
	h.sendSSEEvent(w, flusher, "thinking", map[string]interface{}{
		"phase":    "sql_generation",
		"stage":    "SQL生成",
		"message":  "正在生成SQL查询...",
		"progress": 30,
	})

	// 调用LLM流式生成SQL
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var generatedSQL string
	var streamErr error

	// 这里需要调用queryHandler的流式生成方法
	// 暂时使用普通生成，然后模拟流式输出
	result, err := h.queryHandler.Handle(ctx, question)
	if err != nil {
		h.sendSSEError(w, flusher, fmt.Sprintf("查询失败: %v", err))
		return
	}

	generatedSQL = result.SQL

	// 模拟流式输出SQL
	h.sendSSEEvent(w, flusher, "progress", map[string]interface{}{
		"phase":    "sql_generation",
		"delta":    generatedSQL[:min(len(generatedSQL), 10)],
		"progress": 50,
	})
	if flusher != nil {
		flusher.Flush()
	}

	if len(generatedSQL) > 10 {
		time.Sleep(100 * time.Millisecond)
		h.sendSSEEvent(w, flusher, "progress", map[string]interface{}{
			"phase":    "sql_generation",
			"delta":    generatedSQL[10:min(len(generatedSQL), 30)],
			"progress": 70,
		})
		if flusher != nil {
			flusher.Flush()
		}
	}

	if len(generatedSQL) > 30 {
		time.Sleep(100 * time.Millisecond)
		h.sendSSEEvent(w, flusher, "progress", map[string]interface{}{
			"phase":    "sql_generation",
			"delta":    generatedSQL[30:],
			"progress": 90,
		})
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 检查是否有错误
	if streamErr != nil {
		h.sendSSEError(w, flusher, streamErr.Error())
		return
	}

	// 3. 执行SQL
	h.sendSSEEvent(w, flusher, "thinking", map[string]interface{}{
		"stage":    "执行SQL",
		"message":  "正在执行查询...",
		"progress": 95,
	})

	// 查询已在queryHandler.Handle中执行，使用结果
	queryID := utils.GenerateQueryID()

	// 存储查询上下文
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
	_ = h.feedbackStorage.SetQueryContext(queryContext)

	// 生成反馈链接
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	baseURL := fmt.Sprintf("%s://%s", scheme, host)

	// 发送最终结果
	h.sendSSEEvent(w, flusher, "result", map[string]interface{}{
		"success":     true,
		"question":    result.Question,
		"sql":         generatedSQL,
		"result":      result.Result.Rows,
		"count":       result.Result.Count,
		"duration_ms": result.Duration.Milliseconds(),
		"query_id":    queryID,
		"feedback": map[string]string{
			"positive_url": fmt.Sprintf("%s/feedback/positive/%s", baseURL, queryID),
			"negative_url": fmt.Sprintf("%s/feedback/negative/%s", baseURL, queryID),
		},
		"progress":    100,
	})

	utils.Info("✅ [SSE] 流式查询完成: %s", queryID)
}

// sendSSEEvent 发送SSE事件
func (h *SSEHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		utils.Error("❌ [SSE] 序列化事件数据失败: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	if flusher != nil {
		flusher.Flush()
	}
}

// sendSSEError 发送SSE错误事件
func (h *SSEHandler) sendSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	h.sendSSEEvent(w, flusher, "error", map[string]interface{}{
		"error": message,
	})
}

// min 返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
