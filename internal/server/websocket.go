package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketServer WebSocket服务器
type WebSocketServer struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mutex    sync.RWMutex
	handler  QueryHandlerInterface
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
		clients: make(map[*websocket.Conn]bool),
		handler: queryHandler,
	}
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

	// 发送结果
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
		},
	}

	if verbose {
		response.Data["verbose"] = true
	}

	ws.sendMessage(conn, response)
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
