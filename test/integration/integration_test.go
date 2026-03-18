package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/channelwill/nlq/internal/feedback"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/server"
	"github.com/channelwill/nlq/internal/sanitizer"
)

// TestIntegration_CompleteFeedbackFlow 测试完整的反馈流程
func TestIntegration_CompleteFeedbackFlow(t *testing.T) {
	// 创建临时知识库目录
	tempDir := t.TempDir()
	positiveDir := filepath.Join(tempDir, "positive")
	negativeDir := filepath.Join(tempDir, "negative")

	os.MkdirAll(positiveDir, 0755)
	os.MkdirAll(negativeDir, 0755)

	// 创建初始知识库文件
	positiveExamplesPath := filepath.Join(positiveDir, "positive_examples.md")
	negativeExamplesPath := filepath.Join(negativeDir, "negative_examples.md")

	os.WriteFile(positiveExamplesPath, []byte("# 正面示例\n\n"), 0644)
	os.WriteFile(negativeExamplesPath, []byte("# 负面示例\n\n"), 0644)

	// 创建存储和收集器
	storage := feedback.NewMockStorage()
	collector := feedback.NewCollector(storage)
	merger := feedback.NewMerger(&MockLLMClient{})

	t.Run("完整流程：查询→反馈→合并", func(t *testing.T) {
		// 步骤1: 模拟查询并生成query_id
		queryID := server.GenerateQueryID()
		t.Logf("生成的QueryID: %s", queryID)

		// 步骤2: 创建查询上下文
		queryContext := &feedback.QueryContext{
			QueryID:   queryID,
			Question: "查询销售额大于10000的产品",
			SQL:      "SELECT * FROM products WHERE sales > 10000",
			Timestamp: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		storage.SetQueryContext(queryContext)

		// 步骤3: 提交正面反馈
		feedbackReq := feedback.FeedbackRequest{
			QueryID:    queryID,
			IsPositive: true,
			UserComment: "结果准确",
		}

		err := collector.Collect(feedbackReq)
		if err != nil {
			t.Fatalf("收集反馈失败: %v", err)
		}

		// 步骤4: 验证反馈已存储
		records := storage.GetRecords()
		if len(records) != 1 {
			t.Fatalf("期望1条记录，实际%d条", len(records))
		}

		// 步骤5: 验证数据脱敏
		if records[0].Question != "查询销售额大于10000的产品" {
			t.Error("问题应该保持不变（无敏感信息）")
		}

		// 步骤6: 测试格式化
		entry := merger.FormatEntry(records[0])
		if !strings.Contains(entry, "## 示例") {
			t.Error("正面反馈应该包含示例标题")
		}
		if !strings.Contains(entry, "查询销售额大于10000的产品") {
			t.Error("条目应该包含原始问题")
		}

		t.Logf("✅ 完整流程测试通过")
		t.Logf("格式化的条目:\n%s", entry)
	})
}

// TestIntegration_DataSanitizationFlow 测试数据脱敏流程
func TestIntegration_DataSanitizationFlow(t *testing.T) {
	storage := feedback.NewMockStorage()
	collector := feedback.NewCollector(storage)

	// 创建包含敏感信息的查询上下文
	queryID := server.GenerateQueryID()
	queryContext := &feedback.QueryContext{
		QueryID:   queryID,
		Question: "联系test@example.com或13812345678",
		SQL:      "SELECT * FROM users WHERE email = 'test@example.com'",
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	storage.SetQueryContext(queryContext)

	// 提交反馈
	feedbackReq := feedback.FeedbackRequest{
		QueryID:    queryID,
		IsPositive: true,
		UserComment: "用户邮箱是admin@company.com，手机15912345678",
	}

	err := collector.Collect(feedbackReq)
	if err != nil {
		t.Fatalf("收集反馈失败: %v", err)
	}

	// 验证脱敏
	records := storage.GetRecords()
	if len(records) != 1 {
		t.Fatalf("期望1条记录，实际%d条", len(records))
	}

	record := records[0]

	// 验证问题中的敏感信息已脱敏
	if strings.Contains(record.Question, "test@example.com") {
		t.Error("问题中的邮箱应该被脱敏")
	}
	if strings.Contains(record.Question, "13812345678") {
		t.Error("问题中的手机号应该被脱敏")
	}
	if strings.Contains(record.Question, "admin@company.com") {
		t.Error("备注中的邮箱应该被脱敏")
	}
	if strings.Contains(record.Question, "15912345678") {
		t.Error("备注中的手机号应该被脱敏")
	}

	// 验证SQL中的敏感信息已脱敏
	if strings.Contains(record.GeneratedSQL, "test@example.com") {
		t.Error("SQL中的邮箱应该被脱敏")
	}

	// 验证脱敏后的格式
	if !strings.Contains(record.Question, "***@***.***") {
		t.Log("问题:", record.Question)
		t.Error("应该包含脱敏后的邮箱占位符")
	}

	t.Logf("✅ 数据脱敏流程测试通过")
	t.Logf("脱敏后的问题: %s", record.Question)
	t.Logf("脱敏后的SQL: %s", record.GeneratedSQL)
}

// TestIntegration_Sanitizer 测试脱敏器的集成
func TestIntegration_Sanitizer(t *testing.T) {
	s := sanitizer.NewSanitizer()

	testCases := []struct {
		name     string
		input    string
		checkFn  func(string) bool
	}{
		{
			name:  "邮箱脱敏",
			input: "联系: user@example.com 或 admin@test.org",
			checkFn: func(s string) bool {
				return !strings.Contains(s, "user@example.com") &&
					   !strings.Contains(s, "admin@test.org") &&
					   strings.Contains(s, "***@***.***")
			},
		},
		{
			name:  "手机号脱敏",
			input: "电话: 13812345678, 15987654321",
			checkFn: func(s string) bool {
				return !strings.Contains(s, "13812345678") &&
					   !strings.Contains(s, "15987654321") &&
					   strings.Contains(s, "138****5678")
			},
		},
		{
			name:  "身份证脱敏",
			input: "身份证: 110101199001011234, 31010119900101123X",
			checkFn: func(s string) bool {
				return !strings.Contains(s, "110101199001011234") &&
					   !strings.Contains(s, "31010119900101123X") &&
					   strings.Contains(s, "************1234")
			},
		},
		{
			name:  "IP脱敏",
			input: "服务器: 192.168.1.1, 8.8.8.8",
			checkFn: func(s string) bool {
				return !strings.Contains(s, "192.168.1.1") &&
					   strings.Contains(s, "***.***")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := s.SanitizeAll(tc.input)
			if !tc.checkFn(result) {
				t.Errorf("脱敏失败\n输入: %s\n输出: %s", tc.input, result)
			}
		})
	}

	t.Logf("✅ 脱敏器集成测试通过")
}

// TestIntegration_ConcurrentFeedback 测试并发反馈提交
func TestIntegration_ConcurrentFeedback(t *testing.T) {
	storage := feedback.NewMockStorage()
	collector := feedback.NewCollector(storage)

	numGoroutines := 10
	feedbacksPerGoroutine := 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// 并发提交反馈
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < feedbacksPerGoroutine; j++ {
				// 直接使用GenerateQueryID生成符合格式的query_id
				queryID := server.GenerateQueryID()

				queryContext := &feedback.QueryContext{
					QueryID:   queryID,
					Question: fmt.Sprintf("测试问题 %d-%d", workerID, j),
					SQL:      "SELECT * FROM test",
					Timestamp: time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				storage.SetQueryContext(queryContext)

				feedbackReq := feedback.FeedbackRequest{
					QueryID:    queryID,
					IsPositive: j%2 == 0,
					UserComment: "并发测试",
				}

				if err := collector.Collect(feedbackReq); err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errors)

	// 收集错误
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	// 验证记录数量
	records := storage.GetRecords()
	expectedCount := numGoroutines * feedbacksPerGoroutine

	// 报告结果
	if len(errorList) > 0 {
		t.Errorf("并发反馈出错 %d 次: %v", len(errorList), errorList[0])
	}

	if len(records) != expectedCount {
		t.Errorf("期望%d条记录，实际%d条", expectedCount, len(records))
	}

	if len(errorList) == 0 && len(records) == expectedCount {
		t.Logf("✅ 并发反馈测试通过，成功处理 %d 条并发反馈", expectedCount)
	}
}

// TestIntegration_Performance 测试性能
func TestIntegration_Performance(t *testing.T) {
	storage := feedback.NewMockStorage()
	collector := feedback.NewCollector(storage)
	s := sanitizer.NewSanitizer()

	// 测试脱敏性能
	t.Run("脱敏性能", func(t *testing.T) {
		testText := `联系: test@example.com, 电话: 13812345678,
		身份证: 110101199001011234, IP: 192.168.1.1
		SQL: SELECT * FROM users WHERE email = 'user@test.org' AND phone = '15987654321'`

		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			s.SanitizeAll(testText)
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		t.Logf("脱敏性能测试:")
		t.Logf("  总时间: %v", duration)
		t.Logf("  平均时间: %v/次", avgTime)

		// 验证性能要求：每次脱敏应该在1ms以内
		if avgTime > time.Millisecond {
			t.Errorf("脱敏性能不达标: 平均 %v，要求 < 1ms", avgTime)
		}
	})

	// 测试收集性能
	t.Run("收集性能", func(t *testing.T) {
		queryID := server.GenerateQueryID()
		queryContext := &feedback.QueryContext{
			QueryID:   queryID,
			Question: "测试性能查询",
			SQL:      "SELECT * FROM performance_test",
			Timestamp: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		storage.SetQueryContext(queryContext)

		feedbackReq := feedback.FeedbackRequest{
			QueryID:    queryID,
			IsPositive: true,
		}

		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			collector.Collect(feedbackReq)
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		t.Logf("收集性能测试:")
		t.Logf("  总时间: %v", duration)
		t.Logf("  平均时间: %v/次", avgTime)

		// 验证性能要求：每次收集应该在100ms以内
		if avgTime > 100*time.Millisecond {
			t.Errorf("收集性能不达标: 平均 %v，要求 < 100ms", avgTime)
		}
	})

	t.Logf("✅ 性能测试完成")
}

// TestIntegration_KnowledgeBaseLoading 测试知识库加载
func TestIntegration_KnowledgeBaseLoading(t *testing.T) {
	loader := knowledge.NewLoader()

	// 获取项目根目录（从测试文件位置向上查找）
	baseDir := "../.." // test/integration -> NLQ/
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		t.Fatalf("无法确定项目根目录: %v", err)
	}

	// 测试加载现有知识库目录
	t.Run("加载knowledge目录", func(t *testing.T) {
		knowledgeDir := filepath.Join(absBaseDir, "knowledge")
		docs, err := loader.LoadFromDirectory(knowledgeDir)
		if err != nil {
			t.Logf("加载失败: %v", err)
		}

		t.Logf("从 knowledge/ 加载了 %d 个文档", len(docs))

		// 验证文档内容
		for _, doc := range docs {
			if !doc.IsValid() {
				t.Errorf("文档无效: %s", doc.Title)
			}
			t.Logf("  - %s (%d 字符)", doc.Title, len(doc.Content))
		}
	})

	t.Run("加载positive目录", func(t *testing.T) {
		positiveDir := filepath.Join(absBaseDir, "knowledge/positive")
		docs, err := loader.LoadFromDirectory(positiveDir)
		if err != nil {
			t.Logf("加载失败: %v", err)
		}

		t.Logf("从 knowledge/positive 加载了 %d 个文档", len(docs))

		// 验证初始文件存在
		foundExamples := false
		foundPool := false
		for _, doc := range docs {
			if doc.Source == "positive_examples.md" {
				foundExamples = true
			}
			if doc.Source == "positive_pool.md" {
				foundPool = true
			}
		}

		if !foundExamples {
			t.Error("缺少 positive_examples.md")
		}
		if !foundPool {
			t.Error("缺少 positive_pool.md")
		}
	})

	t.Run("加载negative目录", func(t *testing.T) {
		negativeDir := filepath.Join(absBaseDir, "knowledge/negative")
		docs, err := loader.LoadFromDirectory(negativeDir)
		if err != nil {
			t.Logf("加载失败: %v", err)
		}

		t.Logf("从 knowledge/negative 加载了 %d 个文档", len(docs))

		// 验证初始文件存在
		foundExamples := false
		foundPool := false
		for _, doc := range docs {
			if doc.Source == "negative_examples.md" {
				foundExamples = true
			}
			if doc.Source == "negative_pool.md" {
				foundPool = true
			}
		}

		if !foundExamples {
			t.Error("缺少 negative_examples.md")
		}
		if !foundPool {
			t.Error("缺少 negative_pool.md")
		}
	})

	t.Logf("✅ 知识库加载测试完成")
}

// TestIntegration_QueryIDGeneration 测试QueryID生成
func TestIntegration_QueryIDGeneration(t *testing.T) {
	t.Run("生成多个QueryID验证唯一性", func(t *testing.T) {
		ids := make(map[string]bool)
		iterations := 1000

		for i := 0; i < iterations; i++ {
			id := server.GenerateQueryID()

			// 验证格式
			if !strings.HasPrefix(id, "qry_") {
				t.Errorf("QueryID应该以qry_开头: %s", id)
			}

			// 验证唯一性
			if ids[id] {
				t.Errorf("QueryID重复: %s", id)
			}

			ids[id] = true
		}

		t.Logf("✅ 生成了 %d 个唯一的QueryID", iterations)
	})

	t.Run("验证QueryID格式", func(t *testing.T) {
		id := server.GenerateQueryID()

		// 格式: qry_YYYYMMDD_XXXXXXXX
		parts := strings.Split(id, "_")
		if len(parts) != 3 {
			t.Errorf("QueryID格式错误，应该有3部分: %s", id)
		}

		if parts[0] != "qry" {
			t.Errorf("QueryID第一部分应该是qry: %s", parts[0])
		}

		// 验证日期部分 (8位数字)
		if len(parts[1]) != 8 {
			t.Errorf("日期部分应该是8位: %s", parts[1])
		}

		// 验证随机部分 (8位字符)
		if len(parts[2]) != 8 {
			t.Errorf("随机部分应该是8位: %s", parts[2])
		}

		t.Logf("✅ QueryID格式验证通过: %s", id)
	})
}

// TestIntegration_HTTPQueryWithFeedback 测试HTTP查询和反馈
func TestIntegration_HTTPQueryWithFeedback(t *testing.T) {
	// 创建mock存储
	storage := feedback.NewMockStorage()

	// 创建mock查询处理器
	mockHandler := &MockQueryHandlerWithStorage{storage: storage}
	queryHandler := server.NewQueryHandler(mockHandler)

	t.Run("查询请求包含反馈链接", func(t *testing.T) {
		reqBody := server.QueryRequest{
			Question: "查询所有用户",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/query", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		queryHandler.HandleQuery(w, req)

		// 检查响应
		if w.Code != http.StatusOK {
			t.Errorf("期望状态码200，实际%d", w.Code)
		}

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
			t.Logf("PositiveURL: %s", response.Feedback.PositiveURL)
			t.Logf("NegativeURL: %s", response.Feedback.NegativeURL)
			t.Logf("ExpiresAt: %d", response.Feedback.ExpiresAt)
		}

		// 验证QueryID已设置到mock处理器
		if mockHandler.lastQueryID == "" {
			t.Error("QueryID应该被设置")
		}
	})

	t.Run("反馈提交流程", func(t *testing.T) {
		// 先执行查询获取query_id
		queryID := server.GenerateQueryID()

		queryContext := &feedback.QueryContext{
			QueryID:   queryID,
			Question: "测试查询",
			SQL:      "SELECT * FROM test",
			Timestamp: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		storage.SetQueryContext(queryContext)

		// 创建反馈处理器
		feedbackHandler := server.NewFeedbackHandler(storage)

		reqBody := server.FeedbackRequest{
			QueryID:    queryID,
			IsPositive: true,
			UserComment: "测试反馈",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/feedback/submit", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		feedbackHandler.HandleFeedbackSubmit(w, req)

		// 检查响应
		if w.Code != http.StatusOK {
			t.Errorf("期望状态码200，实际%d", w.Code)
		}

		var response server.FeedbackResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if !response.Success {
			t.Error("反馈提交应该成功")
		}

		// 验证记录已存储
		records := storage.GetRecords()
		if len(records) != 1 {
			t.Fatalf("期望1条记录，实际%d条", len(records))
		}

		if !records[0].IsPositive {
			t.Error("记录应该是正面反馈")
		}

		t.Logf("✅ 反馈提交流程测试通过")
	})
}

// TestIntegration_EndToEnd 测试端到端场景
func TestIntegration_EndToEnd(t *testing.T) {
	t.Run("场景：用户查询并反馈正面评价", func(t *testing.T) {
		// 准备测试环境
		storage := feedback.NewMockStorage()
		queryHandler := &MockQueryHandlerWithStorage{storage: storage}
		serverHandler := server.NewQueryHandler(queryHandler)
		feedbackHandler := server.NewFeedbackHandler(storage)

		// 步骤1: 用户发起查询
		reqBody := server.QueryRequest{Question: "查询销售额大于10000的产品"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/query", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		serverHandler.HandleQuery(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("查询失败，状态码: %d", w.Code)
		}

		var queryResp server.QueryResponse
		if err := json.NewDecoder(w.Body).Decode(&queryResp); err != nil {
			t.Fatalf("解析查询响应失败: %v", err)
		}

		queryID := queryResp.QueryID
		t.Logf("步骤1: 查询成功，QueryID = %s", queryID)

		// 步骤2: 系统保存查询上下文
		queryContext := &feedback.QueryContext{
			QueryID:   queryID,
			Question: queryResp.Question,
			SQL:      queryResp.SQL,
			Timestamp: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		storage.SetQueryContext(queryContext)

		// 步骤3: 用户点击正面反馈链接
		feedbackReq := server.FeedbackRequest{
			QueryID:    queryID,
			IsPositive: true,
			UserComment: "查询结果准确",
		}
		feedbackBody, _ := json.Marshal(feedbackReq)

		feedbackReqHTTP := httptest.NewRequest("POST", "/feedback/submit", bytes.NewReader(feedbackBody))
		feedbackReqHTTP.Header.Set("Content-Type", "application/json")
		fw := httptest.NewRecorder()

		feedbackHandler.HandleFeedbackSubmit(fw, feedbackReqHTTP)

		if fw.Code != http.StatusOK {
			t.Fatalf("反馈提交失败，状态码: %d", fw.Code)
		}

		var feedbackResp server.FeedbackResponse
		if err := json.NewDecoder(fw.Body).Decode(&feedbackResp); err != nil {
			t.Fatalf("解析反馈响应失败: %v", err)
		}

		t.Logf("步骤2: 反馈提交成功，消息 = %s", feedbackResp.Message)

		// 步骤4: 验证反馈已正确存储
		records := storage.GetRecords()
		if len(records) != 1 {
			t.Fatalf("期望1条反馈记录，实际%d条", len(records))
		}

		record := records[0]
		if record.Question != "查询销售额大于10000的产品" {
			t.Errorf("问题不匹配: %s", record.Question)
		}
		if !record.IsPositive {
			t.Error("应该是正面反馈")
		}

		t.Logf("步骤3: 反馈已正确存储并脱敏")
		t.Logf("  问题: %s", record.Question)
		t.Logf("  SQL: %s", record.GeneratedSQL)
		t.Logf("  备注: %s", record.UserComment)

		t.Logf("✅ 端到端测试完成！")
	})
}

// ===== Mock 实现 =====

// MockQueryHandlerWithStorage Mock查询处理器（带存储）
type MockQueryHandlerWithStorage struct {
	storage      feedback.Storage
	lastQueryID string
}

func (m *MockQueryHandlerWithStorage) Handle(ctx context.Context, question string) (*handler.QueryResult, error) {
	// 生成queryID（模拟服务器行为）
	m.lastQueryID = server.GenerateQueryID()

	return &handler.QueryResult{
		Question: question,
		SQL:      "SELECT * FROM test",
		Duration: 100 * time.Millisecond,
	}, nil
}

func (m *MockQueryHandlerWithStorage) HandleWithSQL(ctx context.Context, sqlQuery string) (*handler.QueryResult, error) {
	return &handler.QueryResult{
		SQL:      sqlQuery,
		Duration: 50 * time.Millisecond,
	}, nil
}

func (m *MockQueryHandlerWithStorage) SetKnowledge(docs []knowledge.Document) error {
	return nil
}

// MockLLMClient Mock LLM客户端
type MockLLMClient struct{}

func (m *MockLLMClient) CheckDuplicate(newEntry string, existingEntries []string) (bool, error) {
	return false, nil
}

func (m *MockLLMClient) MergeEntries(existing, newEntry string) (string, error) {
	return existing + "\n" + newEntry, nil
}
