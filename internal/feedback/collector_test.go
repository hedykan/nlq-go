package feedback

import (
	"testing"
	"time"
)

// TestCollector_Validate 测试反馈验证
func TestCollector_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request FeedbackRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的正面反馈",
			request: FeedbackRequest{
				QueryID:    "qry_20250317_abc123",
				IsPositive: true,
			},
			wantErr: false,
		},
		{
			name: "有效的负面反馈",
			request: FeedbackRequest{
				QueryID:    "qry_20250317_abc123",
				IsPositive: false,
			},
			wantErr: false,
		},
		{
			name: "带备注的反馈",
			request: FeedbackRequest{
				QueryID:     "qry_20250317_abc123",
				IsPositive:  true,
				UserComment: "结果很准确",
			},
			wantErr: false,
		},
		{
			name: "负面反馈带正确SQL",
			request: FeedbackRequest{
				QueryID:    "qry_20250317_abc123",
				IsPositive: false,
				CorrectSQL: "SELECT * FROM users ORDER BY created_at DESC",
			},
			wantErr: false,
		},
		{
			name: "空的QueryID",
			request: FeedbackRequest{
				QueryID:    "",
				IsPositive: true,
			},
			wantErr: true,
			errMsg:  "query_id不能为空",
		},
		{
			name: "无效的QueryID格式",
			request: FeedbackRequest{
				QueryID:    "invalid_id",
				IsPositive: true,
			},
			wantErr: true,
			errMsg:  "query_id格式无效",
		},
	}

	// 创建临时存储用于测试
	storage := NewMockStorage()
	collector := NewCollector(storage)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.Validate(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestCollector_Collect_Positive 测试收集正面反馈
func TestCollector_Collect_Positive(t *testing.T) {
	storage := NewMockStorage()
	collector := NewCollector(storage)

	request := FeedbackRequest{
		QueryID:     "qry_20250317_abc123",
		IsPositive:  true,
		UserComment: "查询结果准确",
	}

	// 设置查询上下文
	queryContext := &QueryContext{
		QueryID:  "qry_20250317_abc123",
		Question: "查询所有用户",
		SQL:      "SELECT * FROM users",
	}
	storage.SetQueryContext(queryContext)

	err := collector.Collect(request)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	// 验证存储
	records := storage.GetRecords()
	if len(records) != 1 {
		t.Fatalf("期望1条记录，实际获得%d条", len(records))
	}

	record := records[0]
	if record.IsPositive != true {
		t.Errorf("期望IsPositive=true, 实际%v", record.IsPositive)
	}
	if record.UserComment != "查询结果准确" {
		t.Errorf("期望备注='查询结果准确', 实际'%s'", record.UserComment)
	}
}

// TestCollector_Collect_Negative 测试收集负面反馈
func TestCollector_Collect_Negative(t *testing.T) {
	storage := NewMockStorage()
	collector := NewCollector(storage)

	request := FeedbackRequest{
		QueryID:    "qry_20250317_abc123",
		IsPositive: false,
		CorrectSQL: "SELECT * FROM users ORDER BY created_at DESC",
	}

	queryContext := &QueryContext{
		QueryID:  "qry_20250317_abc123",
		Question: "查询最新用户",
		SQL:      "SELECT * FROM users",
	}
	storage.SetQueryContext(queryContext)

	err := collector.Collect(request)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	records := storage.GetRecords()
	if len(records) != 1 {
		t.Fatalf("期望1条记录，实际获得%d条", len(records))
	}

	record := records[0]
	if record.IsPositive != false {
		t.Errorf("期望IsPositive=false, 实际%v", record.IsPositive)
	}
	if record.CorrectSQL != "SELECT * FROM users ORDER BY created_at DESC" {
		t.Errorf("期望CorrectSQL包含正确SQL")
	}
}

// TestCollector_Collect_Sanitization 测试脱敏功能
func TestCollector_Collect_Sanitization(t *testing.T) {
	storage := NewMockStorage()
	collector := NewCollector(storage)

	request := FeedbackRequest{
		QueryID:     "qry_20250317_abc123",
		IsPositive:  true,
		UserComment: "用户邮箱是test@example.com，手机13812345678",
	}

	queryContext := &QueryContext{
		QueryID:  "qry_20250317_abc123",
		Question: "联系: test@example.com, 13812345678",
		SQL:      "SELECT * FROM users WHERE email = 'test@example.com'",
	}
	storage.SetQueryContext(queryContext)

	err := collector.Collect(request)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	records := storage.GetRecords()
	if len(records) != 1 {
		t.Fatalf("期望1条记录，实际获得%d条", len(records))
	}

	record := records[0]
	// 验证敏感信息已被脱敏
	if contains(record.Question, "test@example.com") {
		t.Error("问题中的邮箱应该被脱敏")
	}
	if contains(record.Question, "13812345678") {
		t.Error("问题中的手机号应该被脱敏")
	}
	if contains(record.GeneratedSQL, "test@example.com") {
		t.Error("SQL中的邮箱应该被脱敏")
	}
}

// contains 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestCollector_Collect_QueryNotFound 测试查询上下文不存在
func TestCollector_Collect_QueryNotFound(t *testing.T) {
	storage := NewMockStorage()
	collector := NewCollector(storage)

	request := FeedbackRequest{
		QueryID:    "qry_20250317_notfound",
		IsPositive: true,
	}

	err := collector.Collect(request)
	if err == nil {
		t.Error("期望返回错误，因为查询上下文不存在")
	}
}

// TestFeedbackRecord_IsExpired 测试记录过期检查
func TestFeedbackRecord_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		timestamp time.Time
		expired   bool
	}{
		{
			name:      "未过期的记录",
			timestamp: time.Now().Add(-1 * time.Hour),
			expired:   false,
		},
		{
			name:      "过期的记录（超过24小时）",
			timestamp: time.Now().Add(-25 * time.Hour),
			expired:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &FeedbackRecord{
				Timestamp: tt.timestamp,
			}
			if got := record.IsExpired(24 * time.Hour); got != tt.expired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expired)
			}
		})
	}
}
