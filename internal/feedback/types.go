package feedback

import (
	"time"
)

// FeedbackRequest 反馈提交请求
type FeedbackRequest struct {
	QueryID     string `json:"query_id"`     // 查询唯一标识
	IsPositive  bool   `json:"is_positive"`  // true=符合预期, false=不符合预期
	UserComment string `json:"user_comment"` // 用户备注（可选）
	CorrectSQL  string `json:"correct_sql"`  // 正确的SQL（负面反馈时可选）
}

// FeedbackResponse 反馈提交响应
type FeedbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// QueryContext 查询上下文（临时存储，供反馈时使用）
type QueryContext struct {
	QueryID   string                 `json:"query_id"`
	Question  string                 `json:"question"`
	SQL       string                 `json:"sql"`
	Result    []map[string]any `json:"result,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	ExpiresAt time.Time              `json:"expires_at"`
}

// FeedbackRecord 反馈记录（存储到知识库）
type FeedbackRecord struct {
	ID           string    `json:"id"`            // 记录唯一ID
	QueryID      string    `json:"query_id"`      // 关联的查询ID
	Question     string    `json:"question"`      // 原始问题（已脱敏）
	GeneratedSQL string    `json:"generated_sql"` // 生成的SQL（已脱敏）
	IsPositive   bool      `json:"is_positive"`   // 反馈类型
	UserComment  string    `json:"user_comment"`  // 用户备注
	CorrectSQL   string    `json:"correct_sql"`   // 用户提供的正确SQL（如果有）
	Timestamp    time.Time `json:"timestamp"`     // 提交时间
}

// IsExpired 检查记录是否过期
func (r *FeedbackRecord) IsExpired(maxAge time.Duration) bool {
	return time.Since(r.Timestamp) > maxAge
}

// IsValid 验证反馈记录是否有效
func (r *FeedbackRecord) IsValid() bool {
	return r.ID != "" && r.QueryID != "" && r.Question != ""
}
