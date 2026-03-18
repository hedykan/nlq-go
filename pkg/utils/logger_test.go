package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHTTPRequestLogger 测试HTTP请求日志记录器
func TestHTTPRequestLogger(t *testing.T) {
	logger := NewHTTPRequestLogger()

	// 创建测试请求
	req := httptest.NewRequest("POST", "/api/v1/query?verbose=true", strings.NewReader(`{"question": "测试查询"}`))
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "127.0.0.1:12345"

	// 测试LogRequest
	requestID := logger.LogRequest(req)
	if requestID == "" {
		t.Error("LogRequest应该返回非空的请求ID")
	}

	// 测试LogRequestStage
	details := map[string]any{
		"question": "测试查询",
		"stage":    "1",
	}
	logger.LogRequestStage(requestID, "1.请求解析", details)
	logger.LogRequestStage(requestID, "2.请求验证", details)
	logger.LogRequestStage(requestID, "3.开始查询", details)

	// 测试LogRequestSuccess
	logger.LogRequestSuccess(requestID, 100, 200)

	// 测试LogRequestError
	logger.LogRequestError(requestID, 50, http.ErrHandlerTimeout)
}

// TestLogLevels 测试日志级别
func TestLogLevels(t *testing.T) {
	// 设置为DEBUG级别，显示所有日志
	SetLogLevel(DEBUG)

	Debug("这是调试信息")
	Info("这是普通信息")
	Warn("这是警告信息")
	Error("这是错误信息")

	// 设置为ERROR级别，只显示错误
	SetLogLevel(ERROR)

	Debug("这条调试信息不应该显示")
	Info("这条普通信息不应该显示")
	Warn("这条警告信息不应该显示")
	Error("这条错误信息应该显示")
}

// BenchmarkHTTPRequestLogger 性能测试
func BenchmarkHTTPRequestLogger(b *testing.B) {
	logger := NewHTTPRequestLogger()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogRequest(req)
		logger.LogRequestSuccess("REQ-1", 100, 200)
	}
}
