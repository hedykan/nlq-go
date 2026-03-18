package feedback

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/channelwill/nlq/internal/sanitizer"
	"github.com/channelwill/nlq/pkg/utils"
)

// Collector 反馈收集器
type Collector struct {
	storage   Storage
	sanitizer *sanitizer.Sanitizer
}

// NewCollector 创建新的反馈收集器
func NewCollector(storage Storage) *Collector {
	return &Collector{
		storage:   storage,
		sanitizer: sanitizer.NewSanitizer(),
	}
}

// Validate 验证反馈请求
func (c *Collector) Validate(req FeedbackRequest) error {
	// 验证QueryID
	if req.QueryID == "" {
		return fmt.Errorf("query_id不能为空")
	}

	// 验证QueryID格式（以qry_开头）
	if !isValidQueryID(req.QueryID) {
		return fmt.Errorf("query_id格式无效")
	}

	return nil
}

// Collect 收集反馈
func (c *Collector) Collect(req FeedbackRequest) error {
	// 验证请求
	if err := c.Validate(req); err != nil {
		return err
	}

	// 获取查询上下文
	context, err := c.storage.GetQueryContext(req.QueryID)
	if err != nil {
		return fmt.Errorf("无法获取查询上下文: %w", err)
	}

	// 创建反馈记录
	record := &FeedbackRecord{
		ID:           utils.GenerateFeedbackID(),
		QueryID:      req.QueryID,
		Question:     c.sanitizeQuestion(context.Question),
		GeneratedSQL: c.sanitizeSQL(context.SQL),
		IsPositive:   req.IsPositive,
		UserComment:  c.sanitizeQuestion(req.UserComment),
		CorrectSQL:   c.sanitizeSQL(req.CorrectSQL),
		Timestamp:    time.Now(),
	}

	// 保存记录
	if err := c.storage.Save(record); err != nil {
		return fmt.Errorf("保存反馈记录失败: %w", err)
	}

	return nil
}

// sanitizeQuestion 脱敏问题文本
func (c *Collector) sanitizeQuestion(question string) string {
	return c.sanitizer.SanitizeQuestion(question)
}

// sanitizeSQL 脱敏SQL语句
func (c *Collector) sanitizeSQL(sql string) string {
	return c.sanitizer.SanitizeSQL(sql)
}

// isValidQueryID 验证QueryID格式
// 格式: qry_{YYYYMMDD}_{随机字符串}
func isValidQueryID(queryID string) bool {
	// 检查是否以qry_开头
	if !strings.HasPrefix(queryID, "qry_") {
		return false
	}

	// 检查格式: qry_日期_随机字符
	pattern := regexp.MustCompile(`^qry_\d{8}_[a-zA-Z0-9]+$`)
	return pattern.MatchString(queryID)
}
