package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/internal/handler"
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

// HandleStreamQuery 处理流式查询请求
func (h *SSEHandler) HandleStreamQuery(w http.ResponseWriter, r *http.Request) {
	// 设置SSE响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(http.Flusher)
	if flusher == nil {
		utils.Warn("⚠️ [SSE] Flusher不可用，流式传输可能不稳定")
	}

	question := r.URL.Query().Get("question")
	if question == "" {
		h.sendSSEError(w, flusher, "问题不能为空")
		return
	}

	utils.Info("🤖 [SSE] 开始Agent流式查询: %s", question)

	// 发送开始事件
	h.sendSSEEvent(w, flusher, "start", map[string]interface{}{
		"message": "开始处理查询",
		"time":    time.Now().Format(time.RFC3339),
	})

	// 检查是否为 Agent 处理器，如果是则使用带进度的回调
	if agentHandler, ok := h.queryHandler.(*handler.AgentQueryHandler); ok {
		h.handleAgentStreamQuery(w, r, flusher, question, agentHandler)
	} else {
		h.handleLegacyStreamQuery(w, r, flusher, question)
	}
}

// handleAgentStreamQuery 使用 Agent 处理器的流式查询
func (h *SSEHandler) handleAgentStreamQuery(w http.ResponseWriter, r *http.Request, flusher http.Flusher, question string, agentHandler *handler.AgentQueryHandler) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// 定义进度回调
	callback := func(step handler.AgentStep) {
		eventType := "thinking"
		switch step.Action {
		case "execution":
			if step.Detail != "" && containsStr(step.Detail, "成功") {
				eventType = "progress"
			}
		case "error_correction":
			eventType = "progress"
		case "self_check", "self_correction":
			eventType = "thinking"
		}

		eventData := map[string]interface{}{
			"turn":     step.Turn,
			"action":   step.Action,
			"message":  step.Detail,
			"progress": turnToProgress(step.Turn, step.Action),
			"data":     step.Data,
		}
		if step.Duration > 0 {
			eventData["duration_ms"] = step.Duration.Milliseconds()
		}
		h.sendSSEEvent(w, flusher, eventType, eventData)
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 调用 Agent 处理器（带进度回调）
	result, err := agentHandler.HandleWithProgress(ctx, question, callback)
	if err != nil {
		h.sendSSEError(w, flusher, fmt.Sprintf("查询失败: %v", err))
		return
	}

	// 存储查询上下文
	queryID := utils.GenerateQueryID()
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
	resultData := map[string]interface{}{
		"success":     true,
		"question":    result.Question,
		"sql":         result.SQL,
		"count":       0,
		"duration_ms": result.Duration.Milliseconds(),
		"query_id":    queryID,
		"feedback": map[string]string{
			"positive_url": fmt.Sprintf("%s/feedback/positive/%s", baseURL, queryID),
			"negative_url": fmt.Sprintf("%s/feedback/negative/%s", baseURL, queryID),
		},
		"progress": 100,
	}

	if result.Result != nil {
		resultData["result"] = result.Result.Rows
		resultData["count"] = result.Result.Count
	}

	if result.Error != "" {
		resultData["success"] = false
		resultData["error"] = result.Error
	}

	h.sendSSEEvent(w, flusher, "result", resultData)
	utils.Info("✅ [SSE] Agent流式查询完成: %s", queryID)
}

// handleLegacyStreamQuery 旧版处理器的流式查询（兼容）
func (h *SSEHandler) handleLegacyStreamQuery(w http.ResponseWriter, r *http.Request, flusher http.Flusher, question string) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := h.queryHandler.Handle(ctx, question)
	if err != nil {
		h.sendSSEError(w, flusher, fmt.Sprintf("查询失败: %v", err))
		return
	}

	queryID := utils.GenerateQueryID()
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

	h.sendSSEEvent(w, flusher, "result", map[string]interface{}{
		"success":     true,
		"question":    result.Question,
		"sql":         result.SQL,
		"result":      result.Result.Rows,
		"count":       result.Result.Count,
		"duration_ms": result.Duration.Milliseconds(),
		"query_id":    queryID,
		"progress":    100,
	})
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

// turnToProgress 将轮次和动作转换为进度百分比
func turnToProgress(turn int, action string) int {
	switch action {
	case "resource_selection":
		return 15
	case "sql_generation":
		return 40
	case "self_check":
		return 60
	case "self_correction":
		return 65
	case "execution":
		return 85
	case "error_correction":
		return 70
	default:
		return 50
	}
}

// containsStr 检查字符串是否包含子串
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
