package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/sql"
)

// MockQueryHandler 模拟查询处理器（适配器）
type MockQueryHandler struct {
	queryResult *handler.QueryResult
	queryError  error
}

func (m *MockQueryHandler) Handle(ctx context.Context, question string) (*handler.QueryResult, error) {
	if m.queryError != nil {
		return nil, m.queryError
	}
	if m.queryResult != nil {
		return m.queryResult, nil
	}
	// 默认返回成功结果，包含完整的Result字段
	return &handler.QueryResult{
		Question: question,
		SQL:      "SELECT * FROM users WHERE id = 1",
		Result: &sql.ExecuteResult{
			Columns: []string{"id", "name", "email"},
			Rows: []map[string]interface{}{
				{"id": 1, "name": "张三", "email": "zhangsan@example.com"},
				{"id": 2, "name": "李四", "email": "lisi@example.com"},
			},
			Count: 2,
		},
		Duration: 100 * time.Millisecond,
		Metadata: make(map[string]interface{}),
	}, nil
}

func (m *MockQueryHandler) HandleWithSQL(ctx context.Context, sqlQuery string) (*handler.QueryResult, error) {
	if m.queryError != nil {
		return nil, m.queryError
	}
	if m.queryResult != nil {
		return m.queryResult, nil
	}
	return &handler.QueryResult{
		SQL: sqlQuery,
		Result: &sql.ExecuteResult{
			Columns: []string{"id", "name"},
			Rows: []map[string]interface{}{
				{"id": 1, "name": "测试数据"},
			},
			Count: 1,
		},
		Duration: 50 * time.Millisecond,
		Metadata: make(map[string]interface{}),
	}, nil
}

func (m *MockQueryHandler) SetKnowledge(docs []knowledge.Document) error {
	return nil
}

// MockQueryHandlerAdapter 适配器，将MockQueryHandler适配为*handler.QueryHandler接口
type MockQueryHandlerAdapter struct {
	mock *MockQueryHandler
}

func (a *MockQueryHandlerAdapter) Handle(ctx context.Context, question string) (*handler.QueryResult, error) {
	return a.mock.Handle(ctx, question)
}

func (a *MockQueryHandlerAdapter) HandleWithSQL(ctx context.Context, sqlQuery string) (*handler.QueryResult, error) {
	return a.mock.HandleWithSQL(ctx, sqlQuery)
}

func (a *MockQueryHandlerAdapter) SetKnowledge(docs []knowledge.Document) error {
	return a.mock.SetKnowledge(docs)
}

// TestQueryHandler_HandleQuery_Success 测试成功的自然语言查询
func TestQueryHandler_HandleQuery_Success(t *testing.T) {
	// 创建mock handler
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandlerWithHandler(mockHandler)

	request := QueryRequest{
		Question: "查询VIP用户",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	queryHandler.HandleQuery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", w.Code)
	}

	var response QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Success != true {
		t.Error("期望Success为true")
	}

	if response.SQL == "" {
		t.Error("期望返回SQL语句")
	}

	if response.Question != "查询VIP用户" {
		t.Errorf("期望问题为'查询VIP用户'，实际: %s", response.Question)
	}
}

// TestQueryHandler_HandleQuery_InvalidJSON 测试无效的JSON请求
func TestQueryHandler_HandleQuery_InvalidJSON(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	req := httptest.NewRequest("POST", "/api/v1/query", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	queryHandler.HandleQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码400，实际: %d", w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Error == "" {
		t.Error("期望返回错误信息")
	}
}

// TestQueryHandler_HandleQuery_EmptyQuestion 测试空问题
func TestQueryHandler_HandleQuery_EmptyQuestion(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	request := QueryRequest{
		Question: "",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	queryHandler.HandleQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码400，实际: %d", w.Code)
	}
}

// TestQueryHandler_HandleSQL_Success 测试成功的SQL查询
func TestQueryHandler_HandleSQL_Success(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	request := SQLRequest{
		SQL: "SELECT * FROM users WHERE id = 1",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/sql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	queryHandler.HandleSQL(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", w.Code)
	}

	var response QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Success != true {
		t.Error("期望Success为true")
	}

	if response.SQL == "" {
		t.Error("期望返回SQL语句")
	}
}

// TestQueryHandler_HandleSQL_EmptySQL 测试空SQL
func TestQueryHandler_HandleSQL_EmptySQL(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	request := SQLRequest{
		SQL: "",
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/sql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	queryHandler.HandleSQL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码400，实际: %d", w.Code)
	}
}

// TestQueryHandler_HealthCheck 测试健康检查
func TestQueryHandler_HealthCheck(t *testing.T) {
	queryHandler := NewQueryHandler(nil)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	queryHandler.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", w.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("期望状态为healthy，实际: %s", response.Status)
	}
}

// TestQueryHandler_Status 测试服务状态
func TestQueryHandler_Status(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	queryHandler.Status(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", w.Code)
	}

	var response StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Service != "nlq" {
		t.Errorf("期望服务名为nlq，实际: %s", response.Service)
	}

	if response.Version == "" {
		t.Error("期望返回版本号")
	}
}

// TestQueryHandler_CORS 测试CORS支持
func TestQueryHandler_CORS(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	req := httptest.NewRequest("OPTIONS", "/api/v1/query", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	queryHandler.HandleQuery(w, req)

	// 检查CORS头
	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader == "" {
		t.Error("期望设置CORS头")
	}

	if w.Code != http.StatusOK {
		t.Errorf("OPTIONS请求期望状态码200，实际: %d", w.Code)
	}
}

// TestNewQueryHandler 测试创建查询处理器
func TestNewQueryHandler(t *testing.T) {
	mockHandler := &MockQueryHandler{}
	queryHandler := NewQueryHandler(mockHandler)

	if queryHandler == nil {
		t.Fatal("期望返回非nil的处理器")
	}

	if queryHandler.queryHandler == nil {
		t.Error("期望设置查询处理器")
	}
}

// TestQueryHandler_HandleQuery_WithKnowledgeBase 测试带知识库的查询
func TestQueryHandler_HandleQuery_WithKnowledgeBase(t *testing.T) {
	mockHandler := &MockQueryHandler{
		queryResult: &handler.QueryResult{
			Question: "查询VIP用户",
			SQL:      "SELECT * FROM users WHERE level = 'C'",
			Result: &sql.ExecuteResult{
				Columns: []string{"id", "name", "level"},
				Rows: []map[string]interface{}{
					{"id": 1, "name": "VIP用户1", "level": "C"},
					{"id": 2, "name": "VIP用户2", "level": "C"},
				},
				Count: 2,
			},
			Duration: 100 * time.Millisecond,
			Metadata: make(map[string]interface{}),
		},
	}
	queryHandler := NewQueryHandler(mockHandler)

	request := QueryRequest{
		Question:       "查询VIP用户",
		KnowledgeBase:  "./knowledge",
		Verbose:        true,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/query", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	queryHandler.HandleQuery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码200，实际: %d", w.Code)
	}

	var response QueryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !response.Success {
		t.Error("期望查询成功")
	}

	if response.SQL == "" {
		t.Error("期望返回SQL语句")
	}
}
