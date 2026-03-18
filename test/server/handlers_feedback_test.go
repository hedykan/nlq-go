package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/server"
	"github.com/channelwill/nlq/pkg/utils"
)

// TestHandleQuery_WithFeedbackLinks 测试查询响应包含反馈链接
func TestHandleQuery_WithFeedbackLinks(t *testing.T) {
	// 创建Mock查询处理器
	mockHandler := &MockQueryHandler{}
	queryHandler := server.NewQueryHandler(mockHandler)

	// 创建请求
	reqBody := `{"question": "查询所有用户"}`
	req := httptest.NewRequest("POST", "/query", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 执行请求
	queryHandler.HandleQuery(w, req)

	// 检查状态码
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}

	// 解析响应
	var response server.QueryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证反馈相关字段
	if response.QueryID == "" {
		t.Error("响应应该包含query_id")
	}

	if response.Feedback == nil {
		t.Error("响应应该包含feedback对象")
	} else {
		if response.Feedback.PositiveURL == "" {
			t.Error("feedback应该包含positive_url")
		}
		if response.Feedback.NegativeURL == "" {
			t.Error("feedback应该包含negative_url")
		}
		if response.Feedback.ExpiresAt == 0 {
			t.Error("feedback应该包含expires_at")
		}
	}
}

// TestHandleFeedbackSubmit_Positive 测试提交正面反馈
func TestHandleFeedbackSubmit_Positive(t *testing.T) {
	// 创建反馈存储
	storage := feedback.NewMockStorage()
	feedbackHandler := server.NewFeedbackHandler(storage)

	// 设置查询上下文
	queryContext := &feedback.QueryContext{
		QueryID:  "qry_20250317_abc123",
		Question: "查询所有用户",
		SQL:      "SELECT * FROM users",
	}
	storage.SetQueryContext(queryContext)

	// 创建请求
	reqBody := server.FeedbackRequest{
		QueryID:    "qry_20250317_abc123",
		IsPositive: true,
		UserComment: "结果准确",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/feedback/submit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 执行请求
	feedbackHandler.HandleFeedbackSubmit(w, req)

	// 检查状态码
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}

	// 解析响应
	var response server.FeedbackResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应
	if !response.Success {
		t.Error("响应success应该为true")
	}
	if response.Message == "" {
		t.Error("响应应该包含message")
	}

	// 验证存储
	records := storage.GetRecords()
	if len(records) != 1 {
		t.Fatalf("期望1条记录，实际%d条", len(records))
	}

	if !records[0].IsPositive {
		t.Error("记录应该是正面反馈")
	}
}

// TestHandleFeedbackSubmit_Negative 测试提交负面反馈
func TestHandleFeedbackSubmit_Negative(t *testing.T) {
	storage := feedback.NewMockStorage()
	feedbackHandler := server.NewFeedbackHandler(storage)

	queryContext := &feedback.QueryContext{
		QueryID:  "qry_20250317_def456",
		Question: "查询最新订单",
		SQL:      "SELECT * FROM orders ORDER BY date DESC",
	}
	storage.SetQueryContext(queryContext)

	reqBody := server.FeedbackRequest{
		QueryID:    "qry_20250317_def456",
		IsPositive: false,
		CorrectSQL: "SELECT * FROM orders ORDER BY created_at DESC LIMIT 10",
		UserComment: "使用了错误的日期字段",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/feedback/submit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	feedbackHandler.HandleFeedbackSubmit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际%d", w.Code)
	}

	records := storage.GetRecords()
	if len(records) != 1 {
		t.Fatalf("期望1条记录，实际%d条", len(records))
	}

	if records[0].IsPositive {
		t.Error("记录应该是负面反馈")
	}
}

// TestHandleFeedbackSubmit_InvalidQueryID 测试无效的QueryID
func TestHandleFeedbackSubmit_InvalidQueryID(t *testing.T) {
	storage := feedback.NewMockStorage()
	feedbackHandler := server.NewFeedbackHandler(storage)

	reqBody := server.FeedbackRequest{
		QueryID:    "",
		IsPositive: true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/feedback/submit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	feedbackHandler.HandleFeedbackSubmit(w, req)

	// 应该返回错误
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码400，实际%d", w.Code)
	}
}

// TestHandleFeedbackSubmit_QueryNotFound 测试查询上下文不存在
func TestHandleFeedbackSubmit_QueryNotFound(t *testing.T) {
	storage := feedback.NewMockStorage()
	feedbackHandler := server.NewFeedbackHandler(storage)

	reqBody := server.FeedbackRequest{
		QueryID:    "qry_20250317_notfound",
		IsPositive: true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/feedback/submit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	feedbackHandler.HandleFeedbackSubmit(w, req)

	// 应该返回错误
	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码400，实际%d", w.Code)
	}
}

// TestGenerateQueryID 测试QueryID生成
func TestGenerateQueryID(t *testing.T) {
	// 生成多个ID，检查格式
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := utils.GenerateQueryID()

		// 检查格式
		if len(id) < 15 {
			t.Errorf("QueryID太短: %s", id)
		}

		// 检查是否以qry_开头
		if !startsWith(id, "qry_") {
			t.Errorf("QueryID应该以qry_开头: %s", id)
		}

		// 检查唯一性
		if ids[id] {
			t.Errorf("QueryID重复: %s", id)
		}
		ids[id] = true
	}
}

// startsWith 辅助函数
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// ===== Mock Query Handler =====

// MockQueryHandler 模拟查询处理器
type MockQueryHandler struct {
	queryID string
}

// Handle 模拟查询处理
func (m *MockQueryHandler) Handle(ctx context.Context, question string) (*handler.QueryResult, error) {
	m.queryID = utils.GenerateQueryID()

	return &handler.QueryResult{
		Question: question,
		SQL:      "SELECT * FROM users",
		Duration: 100 * time.Millisecond,
	}, nil
}

// HandleWithSQL 模拟SQL查询
func (m *MockQueryHandler) HandleWithSQL(ctx context.Context, sqlQuery string) (*handler.QueryResult, error) {
	return &handler.QueryResult{
		SQL:      sqlQuery,
		Duration: 50 * time.Millisecond,
	}, nil
}

// SetKnowledge 设置知识库
func (m *MockQueryHandler) SetKnowledge(docs []knowledge.Document) error {
	return nil
}
