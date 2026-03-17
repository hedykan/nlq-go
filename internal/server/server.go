package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/channelwill/nlq/internal/config"
	"github.com/gorilla/mux"
)

// Server HTTP服务器
type Server struct {
	router          *mux.Router
	server          *http.Server
	queryHandler    *QueryHandler
	wsServer        *WebSocketServer
	config          *config.Config
}

// NewServer 创建新的HTTP服务器
func NewServer(cfg *config.Config, dbHandler QueryHandlerInterface) (*Server, error) {
	// 创建路由器
	router := mux.NewRouter()

	// 创建查询处理器
	queryHandler := NewQueryHandlerWithHandler(dbHandler)

	// 创建WebSocket服务器
	wsServer := NewWebSocketServer(dbHandler)

	// 配置路由
	server := &Server{
		router:       router,
		queryHandler: queryHandler,
		wsServer:     wsServer,
		config:       cfg,
	}

	server.setupRoutes(router)

	// 创建HTTP服务器
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	server.server = httpServer

	return server, nil
}

// setupRoutes 配置路由
func (s *Server) setupRoutes(router *mux.Router) {
	// API路由
	api := router.PathPrefix("/api/v1").Subrouter()

	// 查询相关
	api.HandleFunc("/query", s.queryHandler.HandleQuery).Methods("POST", "OPTIONS")
	api.HandleFunc("/sql", s.queryHandler.HandleSQL).Methods("POST", "OPTIONS")

	// Schema相关
	api.HandleFunc("/schema", s.handleGetSchema).Methods("GET")
	api.HandleFunc("/schema/{table}", s.handleGetTableSchema).Methods("GET")

	// 健康检查和状态
	api.HandleFunc("/health", s.queryHandler.HealthCheck).Methods("GET")
	api.HandleFunc("/status", s.queryHandler.Status).Methods("GET")

	// WebSocket路由
	router.HandleFunc("/ws/v1/query", s.wsServer.HandleWebSocket)

	// 静态文件服务（可选）
	// router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))
}

// handleGetSchema 处理获取数据库Schema
func (s *Server) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现获取Schema的逻辑
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"schema": "TODO"}`))
}

// handleGetTableSchema 处理获取表Schema
func (s *Server) handleGetTableSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tableName := vars["table"]

	// TODO: 实现获取表Schema的逻辑
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"table": "%s", "schema": "TODO"}`, tableName)))
}

// Start 启动服务器
func (s *Server) Start() error {
	fmt.Printf("🌐 HTTP服务器启动在 http://%s:%d\n", s.config.Server.Host, s.config.Server.Port)
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("🛑 正在关闭HTTP服务器...")
	return s.server.Shutdown(ctx)
}

// GetRouter 获取路由器（用于测试）
func (s *Server) GetRouter() *mux.Router {
	return s.router
}
