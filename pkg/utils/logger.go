package utils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
)

var (
	// logLevel 当前日志级别
	logLevel = INFO
	// logger 标准日志输出
	logger = log.New(os.Stdout, "", 0)
	// errorLogger 错误日志输出
	errorLogger = log.New(os.Stderr, "", 0)
	// loggerMutex 日志锁
	loggerMutex sync.Mutex
)

// SetLogLevel 设置日志级别
func SetLogLevel(level LogLevel) {
	loggerMutex.Lock()
	defer loggerMutex.Unlock()
	logLevel = level
}

// formatLevel 格式化日志级别
func formatLevel(level LogLevel) string {
	switch level {
	case DEBUG:
		return "🔍 [DEBUG]"
	case INFO:
		return "ℹ️  [INFO]"
	case WARN:
		return "⚠️  [WARN]"
	case ERROR:
		return "❌ [ERROR]"
	default:
		return "[LOG]"
	}
}

// log 内部日志方法
func logMsg(level LogLevel, format string, args ...interface{}) {
	if level < logLevel {
		return
	}

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	prefix := fmt.Sprintf("%s %s ", timestamp, formatLevel(level))

	if level >= ERROR {
		errorLogger.Println(prefix + msg)
	} else {
		logger.Println(prefix + msg)
	}
}

// Debug 调试日志
func Debug(format string, args ...interface{}) {
	logMsg(DEBUG, format, args...)
}

// Info 信息日志
func Info(format string, args ...interface{}) {
	logMsg(INFO, format, args...)
}

// Warn 警告日志
func Warn(format string, args ...interface{}) {
	logMsg(WARN, format, args...)
}

// Error 错误日志
func Error(format string, args ...interface{}) {
	logMsg(ERROR, format, args...)
}

// HTTPRequest HTTP请求日志
type HTTPRequest struct {
	ID         string
	Method     string
	Path       string
	Query      string
	RemoteAddr string
	UserAgent  string
	Headers    http.Header
	StartTime  time.Time
}

// HTTPRequestLogger HTTP请求日志记录器
type HTTPRequestLogger struct {
	requestID int
	mu        sync.Mutex
}

// NewHTTPRequestLogger 创建HTTP请求日志记录器
func NewHTTPRequestLogger() *HTTPRequestLogger {
	return &HTTPRequestLogger{}
}

// generateRequestID 生成请求ID
func (l *HTTPRequestLogger) generateRequestID() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requestID++
	return fmt.Sprintf("REQ-%d", l.requestID)
}

// LogRequest 记录请求开始
func (l *HTTPRequestLogger) LogRequest(r *http.Request) string {
	requestID := l.generateRequestID()
	query := r.URL.RawQuery
	if query != "" {
		query = "?" + query
	}

	Info("════════════════════════════════════════════════════════════════")
	Info("📨 [HTTP请求] ID=%s | Method=%s | Path=%s%s", requestID, r.Method, r.URL.Path, query)
	Info("📍 [客户端地址] RemoteAddr=%s", r.RemoteAddr)
	Info("🔑 [User-Agent] %s", r.UserAgent())

	return requestID
}

// LogRequestStage 记录请求阶段
func (l *HTTPRequestLogger) LogRequestStage(requestID, stage string, details map[string]any) {
	Info("🔄 [阶段-%s] ID=%s", stage, requestID)
	for key, value := range details {
		Info("   ├─ %s: %v", key, value)
	}
}

// LogRequestSuccess 记录请求成功
func (l *HTTPRequestLogger) LogRequestSuccess(requestID string, duration time.Duration, statusCode int) {
	Info("✅ [请求成功] ID=%s | Status=%d | Duration=%dms", requestID, statusCode, duration.Milliseconds())
	Info("════════════════════════════════════════════════════════════════")
}

// LogRequestError 记录请求错误
func (l *HTTPRequestLogger) LogRequestError(requestID string, duration time.Duration, err error) {
	Error("❌ [请求失败] ID=%s | Duration=%dms | Error=%v", requestID, duration.Milliseconds(), err)
	Error("════════════════════════════════════════════════════════════════")
}
