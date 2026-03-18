package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/gorilla/websocket"
)

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	upgrader        websocket.Upgrader
	clients         map[*websocket.Conn]bool
	mutex           sync.RWMutex
	handler         QueryHandlerInterface
	feedbackStorage feedback.Storage
}

// WebSocketMessage WebSocket消息
type WebSocketMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// NewWebSocketServer 创建WebSocket服务器
func NewWebSocketServer(queryHandler QueryHandlerInterface) *WebSocketServer {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
		clients:         make(map[*websocket.Conn]bool),
		handler:         queryHandler,
		feedbackStorage: feedback.NewMockStorage(),
	}
}

// SetFeedbackStorage 设置反馈存储
func (ws *WebSocketServer) SetFeedbackStorage(storage feedback.Storage) {
	ws.feedbackStorage = storage
}

// HandleWebSocket 处理WebSocket连接
func (ws *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket升级失败: %v\n", err)
		return
	}
	defer conn.Close()

	// 添加到客户端列表
ws.mutex.Lock()
	ws.clients[conn] = true
	ws.mutex.Unlock()

	// 发送欢迎消息
	ws.sendWelcome(conn)

	// 消息处理循环
	for {
		var message WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			fmt.Printf("读取消息失败: %v\n", err)
			break
		}

		// 处理消息
		ws.handleMessage(conn, message)
	}

	// 从客户端列表移除
	ws.mutex.Lock()
	delete(ws.clients, conn)
	ws.mutex.Unlock()
}

// sendWelcome 发送欢迎消息
func (ws *WebSocketServer) sendWelcome(conn *websocket.Conn) {
	welcome := WebSocketMessage{
		Type: "welcome",
		Data: map[string]interface{}{
			"message": "欢迎使用NLQ WebSocket服务",
			"version": "1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	conn.WriteJSON(welcome)
}

// handleMessage 处理接收到的消息
func (ws *WebSocketServer) handleMessage(conn *websocket.Conn, message WebSocketMessage) {
	switch message.Type {
	case "query":
		ws.handleQuery(conn, message)
	case "feedback":
		ws.handleFeedback(conn, message)
	case "ping":
		ws.handlePing(conn)
	case "status":
		ws.handleStatus(conn)
	default:
		ws.sendError(conn, fmt.Sprintf("未知消息类型: %s", message.Type))
	}
}

// handleQuery 处理查询消息
func (ws *WebSocketServer) handleQuery(conn *websocket.Conn, message WebSocketMessage) {
	// 提取问题
	question, ok := message.Data["question"].(string)
	if !ok || question == "" {
		ws.sendError(conn, "问题不能为空")
		return
	}

	// 提取verbose选项
	verbose := false
	if v, ok := message.Data["verbose"].(bool); ok {
		verbose = v
	}

	// 发送处理中消息
	ws.sendMessage(conn, WebSocketMessage{
		Type: "processing",
		Data: map[string]interface{}{
			"message": "正在处理查询...",
		},
	})

	// 执行查询
	ctx := context.Background()
	result, err := ws.handler.Handle(ctx, question)

	if err != nil {
		ws.sendError(conn, fmt.Sprintf("查询失败: %v", err))
		return
	}

	// 生成QueryID
	queryID := GenerateQueryID()

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
	_ = ws.feedbackStorage.SetQueryContext(queryContext) // 忽略错误，不阻止查询

	// 发送结果（包含反馈链接）
	response := WebSocketMessage{
		Type: "result",
		Data: map[string]interface{}{
			"success":    true,
			"question":   result.Question,
			"sql":        result.SQL,
			"result":     result.Result.Rows,
			"count":      result.Result.Count,
			"durationMs": result.Duration.Milliseconds(),
			"metadata":   result.Metadata,
			"query_id":   queryID,
			"feedback": map[string]interface{}{
				"message":    "如果结果符合预期，请发送 feedback 消息，type 为 'positive' 或 'negative'",
				"query_id":   queryID,
				"expires_at": time.Now().Add(24 * time.Hour).Unix(),
			},
		},
	}

	if verbose {
		response.Data["verbose"] = true
	}

	ws.sendMessage(conn, response)
}

// handleFeedback 处理反馈消息
func (ws *WebSocketServer) handleFeedback(conn *websocket.Conn, message WebSocketMessage) {
	// 提取必需字段
	queryID, ok := message.Data["query_id"].(string)
	if !ok || queryID == "" {
		ws.sendError(conn, "query_id 不能为空")
		return
	}

	// 提取反馈类型
	feedbackType, ok := message.Data["type"].(string)
	if !ok || (feedbackType != "positive" && feedbackType != "negative") {
		ws.sendError(conn, "type 必须是 'positive' 或 'negative'")
		return
	}

	// 提取可选字段
	userComment := ""
	if comment, ok := message.Data["comment"].(string); ok {
		userComment = comment
	}

	correctSQL := ""
	if sql, ok := message.Data["correct_sql"].(string); ok {
		correctSQL = sql
	}

	// 创建反馈请求
	req := feedback.FeedbackRequest{
		QueryID:     queryID,
		IsPositive:  feedbackType == "positive",
		UserComment: userComment,
		CorrectSQL:  correctSQL,
	}

	// 创建收集器并收集反馈
	collector := feedback.NewCollector(ws.feedbackStorage)
	if err := collector.Collect(req); err != nil {
		ws.sendError(conn, fmt.Sprintf("提交反馈失败: %v", err))
		return
	}

	// 获取查询上下文（用于写入文件）
	context, _ := ws.feedbackStorage.GetQueryContext(queryID)

	// 异步写入 pending pool 文件
	go func() {
		if err := appendToPendingPoolFile(queryID, context, req.IsPositive, userComment, correctSQL); err != nil {
			fmt.Printf("写入 pending pool 失败: %v\n", err)
		}
	}()

	// 发送成功响应
	ws.sendMessage(conn, WebSocketMessage{
		Type: "feedback_received",
		Data: map[string]interface{}{
			"success": true,
			"message": "反馈已收到，感谢您的反馈！",
			"query_id": queryID,
		},
	})
}

// handlePing 处理ping消息
func (ws *WebSocketServer) handlePing(conn *websocket.Conn) {
	ws.sendMessage(conn, WebSocketMessage{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// handleStatus 处理状态消息
func (ws *WebSocketServer) handleStatus(conn *websocket.Conn) {
	ws.sendMessage(conn, WebSocketMessage{
		Type: "status",
		Data: map[string]interface{}{
			"service":  "nlq",
			"version":  "1.0.0",
			"status":   "running",
			"clients":  len(ws.clients),
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}

// sendMessage 发送消息
func (ws *WebSocketServer) sendMessage(conn *websocket.Conn, message WebSocketMessage) {
	if err := conn.WriteJSON(message); err != nil {
		fmt.Printf("发送消息失败: %v\n", err)
	}
}

// sendError 发送错误消息
func (ws *WebSocketServer) sendError(conn *websocket.Conn, errMsg string) {
	ws.sendMessage(conn, WebSocketMessage{
		Type: "error",
		Data: map[string]interface{}{
			"error": errMsg,
		},
	})
}

// Broadcast 广播消息到所有客户端
func (ws *WebSocketServer) Broadcast(message WebSocketMessage) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	for conn := range ws.clients {
		ws.sendMessage(conn, message)
	}
}

// GetClientCount 获取连接的客户端数量
func (ws *WebSocketServer) GetClientCount() int {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	return len(ws.clients)
}

// appendToPendingPoolFile 将反馈追加到 pending pool 文件（辅助函数）
func appendToPendingPoolFile(queryID string, context *feedback.QueryContext, isPositive bool, userComment, correctSQL string) error {
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

	// 打开文件（追加模式）
	f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	// 写入条目
	if _, err := f.WriteString(builder.String()); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}
