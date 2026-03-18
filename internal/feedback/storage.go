package feedback

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Storage 反馈存储接口
type Storage interface {
	// Save 保存反馈记录
	Save(record *FeedbackRecord) error

	// Get 获取反馈记录
	Get(id string) (*FeedbackRecord, error)

	// GetByQueryID 根据查询ID获取反馈记录
	GetByQueryID(queryID string) ([]*FeedbackRecord, error)

	// SetQueryContext 存储查询上下文（临时）
	SetQueryContext(context *QueryContext) error

	// GetQueryContext 获取查询上下文
	GetQueryContext(queryID string) (*QueryContext, error)

	// Delete 删除反馈记录
	Delete(id string) error
}

// MockStorage 内存存储实现（用于测试）
type MockStorage struct {
	mu            sync.RWMutex
	records       map[string]*FeedbackRecord
	queryContexts map[string]*QueryContext
}

// NewMockStorage 创建Mock存储
func NewMockStorage() *MockStorage {
	return &MockStorage{
		records:       make(map[string]*FeedbackRecord),
		queryContexts: make(map[string]*QueryContext),
	}
}

// Save 保存反馈记录
func (m *MockStorage) Save(record *FeedbackRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if record.ID == "" {
		record.ID = generateID()
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	m.records[record.ID] = record
	return nil
}

// Get 获取反馈记录
func (m *MockStorage) Get(id string) (*FeedbackRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	record, exists := m.records[id]
	if !exists {
		return nil, fmt.Errorf("记录不存在: %s", id)
	}
	return record, nil
}

// GetByQueryID 根据查询ID获取反馈记录
func (m *MockStorage) GetByQueryID(queryID string) ([]*FeedbackRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*FeedbackRecord
	for _, record := range m.records {
		if record.QueryID == queryID {
			results = append(results, record)
		}
	}
	return results, nil
}

// SetQueryContext 存储查询上下文
func (m *MockStorage) SetQueryContext(context *QueryContext) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 设置默认过期时间为24小时
	if context.ExpiresAt.IsZero() {
		context.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	if context.Timestamp.IsZero() {
		context.Timestamp = time.Now()
	}

	m.queryContexts[context.QueryID] = context
	return nil
}

// GetQueryContext 获取查询上下文
func (m *MockStorage) GetQueryContext(queryID string) (*QueryContext, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	context, exists := m.queryContexts[queryID]
	if !exists {
		return nil, fmt.Errorf("查询上下文不存在: %s", queryID)
	}

	// 检查是否过期
	if time.Now().After(context.ExpiresAt) {
		return nil, fmt.Errorf("查询上下文已过期: %s", queryID)
	}

	return context, nil
}

// Delete 删除反馈记录
func (m *MockStorage) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.records, id)
	return nil
}

// GetRecords 获取所有记录（测试辅助方法）
func (m *MockStorage) GetRecords() []*FeedbackRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*FeedbackRecord
	for _, record := range m.records {
		results = append(results, record)
	}
	return results
}

// Clear 清空所有记录（测试辅助方法）
func (m *MockStorage) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.records = make(map[string]*FeedbackRecord)
	m.queryContexts = make(map[string]*QueryContext)
}

// generateID 生成唯一ID（使用加密随机数确保唯一性）
func generateID() string {
	// 生成8字节的随机数
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，回退到时间戳+计数器的方式
		return fmt.Sprintf("fb_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
	}
	// 转换为16字符的十六进制字符串
	return fmt.Sprintf("fb_%s", hex.EncodeToString(b))
}
