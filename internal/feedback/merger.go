package feedback

import (
	"fmt"
	"os"
	"strings"
)

// LLMClient LLM客户端接口
type LLMClient interface {
	// CheckDuplicate 使用LLM检查是否重复
	CheckDuplicate(newEntry string, existingEntries []string) (bool, error)

	// MergeEntries 使用LLM合并条目
	MergeEntries(existing, newEntry string) (string, error)
}

// Merger 知识库合并器
type Merger struct {
	llmClient LLMClient
}

// NewMerger 创建新的合并器
func NewMerger(llmClient LLMClient) *Merger {
	return &Merger{
		llmClient: llmClient,
	}
}

// CheckDuplicate 使用LLM检查新条目是否与现有条目重复
func (m *Merger) CheckDuplicate(newEntry string, existingEntries []string) (bool, error) {
	if m.llmClient == nil {
		// 如果没有LLM客户端，使用简单的字符串匹配
		return m.simpleDuplicateCheck(newEntry, existingEntries), nil
	}

	return m.llmClient.CheckDuplicate(newEntry, existingEntries)
}

// simpleDuplicateCheck 简单的重复检测（无LLM时使用）
func (m *Merger) simpleDuplicateCheck(newEntry string, existingEntries []string) bool {
	newEntryNormalized := normalizeEntry(newEntry)

	for _, existing := range existingEntries {
		existingNormalized := normalizeEntry(existing)
		if newEntryNormalized == existingNormalized {
			return true
		}
	}
	return false
}

// normalizeEntry 规范化条目用于比较
func normalizeEntry(entry string) string {
	// 移除多余空格和换行
	entry = strings.TrimSpace(entry)
	entry = strings.ReplaceAll(entry, " ", "")
	entry = strings.ReplaceAll(entry, "\n", "")
	entry = strings.ToLower(entry)
	return entry
}

// FormatEntry 格式化反馈记录为Markdown条目
func (m *Merger) FormatEntry(record *FeedbackRecord) string {
	var builder strings.Builder

	if record.IsPositive {
		// 正面反馈格式
		builder.WriteString("## 示例\n")
		builder.WriteString(fmt.Sprintf("**问题**: %s\n", record.Question))
		builder.WriteString(fmt.Sprintf("**SQL**: %s\n", record.GeneratedSQL))
		if record.UserComment != "" {
			builder.WriteString(fmt.Sprintf("**说明**: %s\n", record.UserComment))
		}
	} else {
		// 负面反馈格式
		builder.WriteString("## 错误模式\n")
		builder.WriteString(fmt.Sprintf("**问题**: %s\n", record.Question))
		builder.WriteString(fmt.Sprintf("**错误SQL**: %s\n", record.GeneratedSQL))
		if record.CorrectSQL != "" {
			builder.WriteString(fmt.Sprintf("**正确SQL**: %s\n", record.CorrectSQL))
		}
		if record.UserComment != "" {
			if record.CorrectSQL != "" {
				builder.WriteString(fmt.Sprintf("**问题说明**: %s\n", record.UserComment))
			} else {
				builder.WriteString(fmt.Sprintf("**说明**: %s\n", record.UserComment))
			}
		}
	}

	return builder.String()
}

// MergePending 合并待处理反馈到知识库
func (m *Merger) MergePending(pendingPath, targetPath string) error {
	// 读取现有知识库
	existingContent, err := os.ReadFile(targetPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取目标文件失败: %w", err)
	}

	// 读取待处理反馈
	pendingContent, err := os.ReadFile(pendingPath)
	if err != nil {
		return fmt.Errorf("读取待处理文件失败: %w", err)
	}

	// 解析待处理条目
	pendingEntries := m.parseEntries(string(pendingContent))

	// 解析现有条目
	existingEntries := m.parseEntries(string(existingContent))

	// 合并非重复条目
	var mergedEntries []string
	mergedEntries = append(mergedEntries, existingEntries...)

	for _, pending := range pendingEntries {
		// 检查是否重复
		isDuplicate, err := m.CheckDuplicate(pending, existingEntries)
		if err != nil {
			return fmt.Errorf("检查重复失败: %w", err)
		}

		if !isDuplicate {
			mergedEntries = append(mergedEntries, pending)
		}
	}

	// 写入目标文件
	output := m.formatDocument(mergedEntries)
	if err := os.WriteFile(targetPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("写入目标文件失败: %w", err)
	}

	// 清空待处理文件
	if err := os.WriteFile(pendingPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("清空待处理文件失败: %w", err)
	}

	return nil
}

// parseEntries 解析Markdown文件中的条目
func (m *Merger) parseEntries(content string) []string {
	if content == "" {
		return []string{}
	}

	// 检查是什么格式
	if strings.Contains(content, "## ") {
		// 标准格式：按 "## " 分割条目
		parts := strings.Split(content, "## ")
		var entries []string

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			entries = append(entries, "## "+part)
		}

		return entries
	} else {
		// 简化格式：按 "---" 分割条目（我们的 pending pool 格式）
		parts := strings.Split(content, "---")
		var entries []string

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" || strings.HasPrefix(part, "*") {
				// 跳过空内容和注释行
				continue
			}
			if strings.HasPrefix(part, "#") {
				// 跳过标题行
				continue
			}
			entries = append(entries, part)
		}

		return entries
	}
}

// formatDocument 格式化完整文档
func (m *Merger) formatDocument(entries []string) string {
	if len(entries) == 0 {
		return ""
	}

	return strings.Join(entries, "\n\n---\n\n")
}

// setupTestFiles 设置测试文件（测试辅助函数）
func setupTestFiles(path, content string) error {
	if content == "" {
		content = "# 测试知识库\n\n"
	}
	return os.WriteFile(path, []byte(content), 0644)
}
