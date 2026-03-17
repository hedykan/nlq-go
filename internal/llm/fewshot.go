package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ExampleType 问题类型枚举
type ExampleType string

const (
	TimeSortType   ExampleType = "time_sort"   // 时间排序类查询
	FieldQueryType ExampleType = "field_query" // 字段查询类
	AggregationType ExampleType = "aggregation" // 聚合统计类
	JoinType       ExampleType = "join"        // 多表关联类
)

// EnhancedFewShotExample 增强的Few-shot示例结构
type EnhancedFewShotExample struct {
	ID         string     `json:"id"`
	Type       ExampleType `json:"type"`
	Question   string     `json:"question"`
	SQL        string     `json:"sql"`
	Tables     []string   `json:"tables"`
	FieldHints []string   `json:"field_hints"` // 涉及的字段提示
}

// ExampleRepository Few-shot示例仓库
type ExampleRepository struct {
	examples  []EnhancedFewShotExample
	mu        sync.RWMutex
	dataPath  string
}

// NewExampleRepository 创建示例仓库
func NewExampleRepository(dataPath string) *ExampleRepository {
	repo := &ExampleRepository{
		examples: []EnhancedFewShotExample{},
		dataPath: dataPath,
	}
	repo.loadExamples()
	return repo
}

// loadExamples 从文件加载示例
func (r *ExampleRepository) loadExamples() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 构建示例文件路径
	examplesFile := filepath.Join(r.dataPath, "examples.json")

	// 检查文件是否存在
	if _, err := os.Stat(examplesFile); os.IsNotExist(err) {
		// 文件不存在，使用默认示例
		r.examples = r.getDefaultExamples()
		return nil
	}

	// 读取文件
	data, err := os.ReadFile(examplesFile)
	if err != nil {
		r.examples = r.getDefaultExamples()
		return fmt.Errorf("读取示例文件失败: %w", err)
	}

	// 解析JSON
	var result struct {
		Examples []EnhancedFewShotExample `json:"examples"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		r.examples = r.getDefaultExamples()
		return fmt.Errorf("解析示例文件失败: %w", err)
	}

	r.examples = result.Examples
	return nil
}

// getDefaultExamples 获取默认示例（fallback）
func (r *ExampleRepository) getDefaultExamples() []EnhancedFewShotExample {
	return []EnhancedFewShotExample{
		{
			ID:       "time_sort_001",
			Type:     TimeSortType,
			Question: "查询100个最早的用户的username",
			SQL:      "SELECT username FROM boom_user ORDER BY created_at ASC LIMIT 100",
			Tables:   []string{"boom_user"},
			FieldHints: []string{"created_at", "username"},
		},
		{
			ID:       "field_query_001",
			Type:     FieldQueryType,
			Question: "查询100个最早的用户的shop_name",
			SQL:      "SELECT shop_name FROM boom_user ORDER BY created_at ASC LIMIT 100",
			Tables:   []string{"boom_user"},
			FieldHints: []string{"created_at", "shop_name"},
		},
		{
			ID:       "field_query_002",
			Type:     FieldQueryType,
			Question: "查询VIP用户的数量",
			SQL:      "SELECT COUNT(*) as total FROM boom_user WHERE level = 'C'",
			Tables:   []string{"boom_user"},
			FieldHints: []string{"level", "COUNT"},
		},
	}
}

// RetrieveExamples 根据问题类型检索相关示例
func (r *ExampleRepository) RetrieveExamples(question string, maxExamples int) []EnhancedFewShotExample {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 分析问题类型
	questionType := r.analyzeQuestionType(question)

	// 检索匹配类型的示例
	var matchedExamples []EnhancedFewShotExample
	for _, example := range r.examples {
		if example.Type == questionType {
			matchedExamples = append(matchedExamples, example)
		}
	}

	// 如果匹配的示例数量不足，添加通用示例
	if len(matchedExamples) < maxExamples {
		for _, example := range r.examples {
			if example.Type != questionType {
				matchedExamples = append(matchedExamples, example)
			}
			if len(matchedExamples) >= maxExamples {
				break
			}
		}
	}

	// 限制返回数量
	if len(matchedExamples) > maxExamples {
		matchedExamples = matchedExamples[:maxExamples]
	}

	return matchedExamples
}

// analyzeQuestionType 分析问题类型
func (r *ExampleRepository) analyzeQuestionType(question string) ExampleType {
	question = strings.ToLower(question)

	// 时间排序类
	if r.containsAnyKeyword(question, []string{"最早", "最晚", "最新", "最早", "first", "last", "recent", "oldest"}) {
		return TimeSortType
	}

	// 聚合统计类
	if r.containsAnyKeyword(question, []string{"多少", "数量", "总数", "统计", "平均", "sum", "count", "avg", "total"}) {
		return AggregationType
	}

	// 多表关联类
	if r.containsAnyKeyword(question, []string{"关联", "join", "结合", "包含"}) {
		return JoinType
	}

	// 默认为字段查询类
	return FieldQueryType
}

// containsAnyKeyword 检查是否包含任意关键词
func (r *ExampleRepository) containsAnyKeyword(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// FormatExamplesForPrompt 格式化示例为Prompt
func (r *ExampleRepository) FormatExamplesForPrompt(examples []EnhancedFewShotExample) string {
	var builder strings.Builder

	builder.WriteString("# 参考示例\n\n")

	for i, example := range examples {
		builder.WriteString(fmt.Sprintf("## 示例 %d\n", i+1))
		builder.WriteString(fmt.Sprintf("问题: %s\n", example.Question))
		builder.WriteString(fmt.Sprintf("SQL: %s\n", example.SQL))
		if len(example.Tables) > 0 {
			builder.WriteString(fmt.Sprintf("涉及表: %s\n", strings.Join(example.Tables, ", ")))
		}
		if len(example.FieldHints) > 0 {
			builder.WriteString(fmt.Sprintf("关键字段: %s\n", strings.Join(example.FieldHints, ", ")))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// GetAllExamples 获取所有示例
func (r *ExampleRepository) GetAllExamples() []EnhancedFewShotExample {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]EnhancedFewShotExample, len(r.examples))
	copy(result, r.examples)
	return result
}

// AddExample 添加新示例
func (r *ExampleRepository) AddExample(example EnhancedFewShotExample) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.examples = append(r.examples, example)

	// 持久化到文件
	return r.saveExamples()
}

// saveExamples 保存示例到文件
func (r *ExampleRepository) saveExamples() error {
	examplesFile := filepath.Join(r.dataPath, "examples.json")

	// 确保目录存在
	if err := os.MkdirAll(r.dataPath, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建JSON结构
	result := struct {
		Examples []EnhancedFewShotExample `json:"examples"`
	}{
		Examples: r.examples,
	}

	// 序列化JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化示例失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(examplesFile, data, 0644); err != nil {
		return fmt.Errorf("写入示例文件失败: %w", err)
	}

	return nil
}
