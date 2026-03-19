package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/pkg/utils"
	"github.com/gorilla/mux"
)

// Server HTTP服务器
type Server struct {
	router              *mux.Router
	server              *http.Server
	queryHandler        *QueryHandler
	wsServer            *WebSocketServer
	feedbackHandler     *FeedbackHandler
	queryPageHandler    *QueryPageHandler
	suggestionsHandler  *SuggestionsHandler
	sseHandler          *SSEHandler
	config              *config.Config
	httpLogger          *utils.HTTPRequestLogger
}

// NewServer 创建新的HTTP服务器
func NewServer(cfg *config.Config, dbHandler QueryHandlerInterface) (*Server, error) {
	// 创建路由器
	router := mux.NewRouter()

	// 创建共享的反馈存储
	feedbackStorage := feedback.NewMockStorage()

	// 创建反馈处理器（使用共享的反馈存储）
	feedbackHandler := NewFeedbackHandler(feedbackStorage)

	// 创建查询处理器（使用共享的反馈存储和反馈处理器）
	queryHandler := NewQueryHandlerWithFeedback(dbHandler, feedbackStorage, feedbackHandler)

	// 创建WebSocket服务器（使用共享的反馈存储）
	wsServer := NewWebSocketServer(dbHandler)
	wsServer.SetFeedbackStorage(feedbackStorage)

	// 创建查询页面处理器
	queryPageHandler := NewQueryPageHandler(queryHandler)

	// 创建示例问题处理器
	suggestionsHandler := NewSuggestionsHandler(dbHandler)

	// 创建SSE处理器（使用底层的QueryHandlerInterface）
	sseHandler := NewSSEHandler(dbHandler, feedbackStorage)

	// 配置路由
	server := &Server{
		router:             router,
		queryHandler:       queryHandler,
		wsServer:           wsServer,
		feedbackHandler:    feedbackHandler,
		queryPageHandler:   queryPageHandler,
		suggestionsHandler: suggestionsHandler,
		sseHandler:         sseHandler,
		config:             cfg,
		httpLogger:         utils.NewHTTPRequestLogger(),
	}

	server.setupRoutes(router)

	// 添加日志中间件
	router.Use(server.loggingMiddleware)

	// 创建HTTP服务器
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  180 * time.Second,  // 增加空闲超时以支持慢速LLM
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

	// 错误记录
	api.HandleFunc("/record-error", s.queryHandler.HandleRecordError).Methods("POST", "OPTIONS")

	// Schema相关
	api.HandleFunc("/schema", s.handleGetSchema).Methods("GET")
	api.HandleFunc("/schema/{table}", s.handleGetTableSchema).Methods("GET")

	// 健康检查和状态
	api.HandleFunc("/health", s.queryHandler.HealthCheck).Methods("GET")
	api.HandleFunc("/status", s.queryHandler.Status).Methods("GET")

	// 示例问题
	api.HandleFunc("/suggestions", s.suggestionsHandler.HandleSuggestions).Methods("GET", "OPTIONS")

	// 流式查询
	api.HandleFunc("/stream-query", s.sseHandler.HandleStreamQuery).Methods("GET")

	// 查询页面路由
	router.HandleFunc("/query", s.queryPageHandler.HandleQueryPage).Methods("GET")

	// 反馈相关
	router.HandleFunc("/feedback/positive/{query_id}", s.feedbackHandler.HandleFeedbackPage).Methods("GET")
	router.HandleFunc("/feedback/negative/{query_id}", s.feedbackHandler.HandleFeedbackPage).Methods("GET")
	router.HandleFunc("/feedback/submit", s.feedbackHandler.HandleFeedbackSubmit).Methods("POST", "OPTIONS")
	router.HandleFunc("/feedback/merge", s.feedbackHandler.HandleFeedbackMerge).Methods("POST", "OPTIONS")
	router.HandleFunc("/feedback/stats", s.feedbackHandler.HandleFeedbackStats).Methods("GET", "OPTIONS")

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

// loggingMiddleware HTTP请求日志中间件
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 记录请求开始
		requestID := s.httpLogger.LogRequest(r)
		startTime := time.Now()

		// 创建响应记录器以捕获状态码
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 将请求ID添加到上下文
		ctx := r.Context()
		ctx = context.WithValue(ctx, "requestID", requestID)
		ctx = context.WithValue(ctx, "httpLogger", s.httpLogger)
		ctx = context.WithValue(ctx, "startTime", startTime)
		r = r.WithContext(ctx)

		// 调用下一个处理器
		defer func() {
			duration := time.Since(startTime)
			if recorder.statusCode >= 400 {
				s.httpLogger.LogRequestError(requestID, duration, fmt.Errorf("HTTP %d", recorder.statusCode))
			} else {
				s.httpLogger.LogRequestSuccess(requestID, duration, recorder.statusCode)
			}
		}()

		next.ServeHTTP(recorder, r)
	})
}

// responseRecorder 响应记录器
// 实现http.ResponseWriter和http.Flusher接口
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 拦截WriteHeader以记录状态码
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Flush 实现http.Flusher接口（如果底层ResponseWriter支持的话）
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack 实现http.Hijacker接口（用于WebSocket等场景）
func (r *responseRecorder) Hijack() (c interface{}, rw interface{}, err error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
}
