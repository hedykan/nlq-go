package feedback

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestMerger_CheckDuplicate 测试重复检测
func TestMerger_CheckDuplicate(t *testing.T) {
	tests := []struct {
		name           string
		newEntry       string
		existingEntries []string
		isDuplicate    bool
		llmResponse    string // Mock LLM响应
	}{
		{
			name:     "新条目不重复",
			newEntry: "问题: 查询销售额大于10000的产品\nSQL: SELECT * FROM products WHERE sales > 10000",
			existingEntries: []string{
				"问题: 查询用户信息\nSQL: SELECT * FROM users",
			},
			isDuplicate: false,
			llmResponse: "不重复",
		},
		{
			name:     "完全相同的条目",
			newEntry: "问题: 查询销售额大于10000的产品\nSQL: SELECT * FROM products WHERE sales > 10000",
			existingEntries: []string{
				"问题: 查询销售额大于10000的产品\nSQL: SELECT * FROM products WHERE sales > 10000",
			},
			isDuplicate: true,
			llmResponse: "重复",
		},
		{
			name:     "语义相同的条目（SQL不同但意义相同）",
			newEntry: "问题: 查找销售超过1万的产品\nSQL: SELECT * FROM products WHERE sales > 10000",
			existingEntries: []string{
				"问题: 查询销售额大于10000的产品\nSQL: SELECT * FROM products WHERE sales > 10000",
			},
			isDuplicate: true,
			llmResponse: "重复，语义相同",
		},
		{
			name:           "空知识库",
			newEntry:       "问题: 查询所有用户\nSQL: SELECT * FROM users",
			existingEntries: []string{},
			isDuplicate:    false,
			llmResponse:    "不重复",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Mock LLM客户端
			mockLLM := &MockLLMClient{
				response: tt.llmResponse,
			}

			merger := NewMerger(mockLLM)

			isDup, err := merger.CheckDuplicate(tt.newEntry, tt.existingEntries)
			if err != nil {
				t.Fatalf("CheckDuplicate() error = %v", err)
			}

			if isDup != tt.isDuplicate {
				t.Errorf("CheckDuplicate() = %v, want %v", isDup, tt.isDuplicate)
			}
		})
	}
}

// TestMerger_FormatEntry 测试条目格式化
func TestMerger_FormatEntry(t *testing.T) {
	tests := []struct {
		name     string
		record   *FeedbackRecord
		contains []string // 检查结果中是否包含这些字符串
	}{
		{
			name: "正面反馈条目",
			record: &FeedbackRecord{
				QueryID:      "qry_20250317_abc123",
				Question:     "查询销售额大于10000的产品",
				GeneratedSQL: "SELECT * FROM products WHERE sales > 10000",
				IsPositive:   true,
				UserComment:  "结果准确",
				Timestamp:    parseTime("2025-03-17T10:00:00Z"),
			},
			contains: []string{
				"## 示例",
				"**问题**: 查询销售额大于10000的产品",
				"**SQL**: SELECT * FROM products WHERE sales > 10000",
				"**说明**: 结果准确",
			},
		},
		{
			name: "负面反馈条目（有正确SQL）",
			record: &FeedbackRecord{
				QueryID:      "qry_20250317_def456",
				Question:     "查询最新订单",
				GeneratedSQL: "SELECT * FROM orders ORDER BY date DESC",
				IsPositive:   false,
				CorrectSQL:   "SELECT * FROM orders ORDER BY created_at DESC LIMIT 10",
				UserComment:  "使用了错误的日期字段",
			},
			contains: []string{
				"## 错误模式",
				"**问题**: 查询最新订单",
				"**错误SQL**: SELECT * FROM orders ORDER BY date DESC",
				"**正确SQL**: SELECT * FROM orders ORDER BY created_at DESC LIMIT 10",
				"**问题说明**: 使用了错误的日期字段",
			},
		},
		{
			name: "负面反馈条目（无正确SQL）",
			record: &FeedbackRecord{
				QueryID:      "qry_20250317_ghi789",
				Question:     "查询最近订单",
				GeneratedSQL: "SELECT * FROM orders",
				IsPositive:   false,
				UserComment:  "缺少排序和限制",
			},
			contains: []string{
				"**问题**: 查询最近订单",
				"**错误SQL**: SELECT * FROM orders",
				"**说明**: 缺少排序和限制",
			},
		},
	}

	merger := NewMerger(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merger.FormatEntry(tt.record)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatEntry() 结果应该包含 '%s'\n实际结果:\n%s", expected, result)
				}
			}
		})
	}
}

// TestMerger_MergePending 测试合并待处理反馈
func TestMerger_MergePending(t *testing.T) {
	tests := []struct {
		name            string
		pendingPath     string
		targetPath      string
		existingContent  string
		pendingRecords  []*FeedbackRecord
		shouldMerge     int // 期望合并的记录数
		llmResponse     string // Mock LLM响应
	}{
		{
			name:        "合并到空知识库",
			pendingPath: "/tmp/test_pending.md",
			targetPath:  "/tmp/test_target.md",
			existingContent: "",
			pendingRecords: []*FeedbackRecord{
				{
					Question:     "查询销售额大于10000的产品",
					GeneratedSQL: "SELECT * FROM products WHERE sales > 10000",
					IsPositive:   true,
				},
			},
			shouldMerge: 1,
			llmResponse: "不重复",
		},
		{
			name:        "合并到已有知识库（有重复）",
			pendingPath: "/tmp/test_pending2.md",
			targetPath:  "/tmp/test_target2.md",
			existingContent: `# 正面查询示例

## 示例 1
**问题**: 查询销售额大于10000的产品
**SQL**: SELECT * FROM products WHERE sales > 10000
**说明**: 简单数值比较
`,
			pendingRecords: []*FeedbackRecord{
				{
					Question:     "查询销售额大于10000的产品",
					GeneratedSQL: "SELECT * FROM products WHERE sales > 10000",
					IsPositive:   true,
				},
			},
			shouldMerge: 0, // 重复，不应该合并
			llmResponse: "重复",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Mock LLM
			mockLLM := &MockLLMClient{
				response: tt.llmResponse, // 使用测试用例的响应
			}

			merger := NewMerger(mockLLM)

			// 准备测试文件
			setupTestFiles(tt.targetPath, tt.existingContent)

			// 创建pending文件
			var pendingContent string
			for _, record := range tt.pendingRecords {
				pendingContent += merger.FormatEntry(record) + "\n\n---\n\n"
			}
			setupTestFiles(tt.pendingPath, pendingContent)

			// 执行合并
			err := merger.MergePending(tt.pendingPath, tt.targetPath)
			if err != nil {
				t.Fatalf("MergePending() error = %v", err)
			}

			// 清理测试文件
			os.Remove(tt.targetPath)
			os.Remove(tt.pendingPath)

			t.Log("合并完成")
		})
	}
}

// TestMerger_FormatEntry_Escaping 测试特殊字符转义
func TestMerger_FormatEntry_Escaping(t *testing.T) {
	record := &FeedbackRecord{
		Question:     "查询包含特殊字符: 'test' 和 \"quote\"",
		GeneratedSQL: "SELECT * FROM users WHERE name = 'test'",
		IsPositive:   true,
	}

	merger := NewMerger(nil)
	result := merger.FormatEntry(record)

	// 检查Markdown格式是否正确
	if !strings.Contains(result, "**问题**:") {
		t.Error("应该包含问题标题")
	}
	if !strings.Contains(result, "**SQL**:") {
		t.Error("应该包含SQL标题")
	}
}

// parseTime 辅助函数：解析时间字符串
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// ===== Mock LLM Client =====

// MockLLMClient 模拟LLM客户端
type MockLLMClient struct {
	response string
}

// CheckDuplicate 模拟重复检测
func (m *MockLLMClient) CheckDuplicate(newEntry string, existingEntries []string) (bool, error) {
	// 根据预设响应返回结果
	if m.response == "重复" || m.response == "重复，语义相同" {
		return true, nil
	}
	return false, nil
}

// MergeEntries 模拟合并条目
func (m *MockLLMClient) MergeEntries(existing, newEntry string) (string, error) {
	return existing + "\n" + newEntry, nil
}
