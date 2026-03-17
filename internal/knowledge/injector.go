package knowledge

import (
	"fmt"
	"strings"
)

// Injector 知识库注入器
type Injector struct {
	maxTokens int  // 最大Token数限制
	enabled   bool // 是否启用注入
}

// NewInjector 创建新的知识库注入器
func NewInjector() *Injector {
	return &Injector{
		maxTokens: 4000, // 默认最大Token数
		enabled:   true,
	}
}

// Inject 将知识库文档注入到基础Prompt中
func (i *Injector) Inject(basePrompt string, documents []Document) string {
	if len(documents) == 0 {
		return basePrompt
	}

	// 构建知识库上下文
	knowledgeContext := i.BuildKnowledgeContext(documents)

	// 如果基础Prompt为空，直接返回知识库上下文
	if basePrompt == "" {
		return knowledgeContext
	}

	// 组合基础Prompt和知识库上下文
	return fmt.Sprintf("%s\n\n%s\n\n%s",
		i.buildSectionHeader("数据库Schema"),
		basePrompt,
		knowledgeContext,
	)
}

// InjectWithMaxTokens 使用Token限制注入知识库
func (i *Injector) InjectWithMaxTokens(basePrompt string, documents []Document, maxTokens int) string {
	// 设置临时Token限制
	originalMaxTokens := i.maxTokens
	i.maxTokens = maxTokens
	defer func() { i.maxTokens = originalMaxTokens }()

	// 如果没有Token限制，使用普通注入
	if maxTokens <= 0 {
		return i.Inject(basePrompt, documents)
	}

	// 简单估算：假设1个Token约等于4个字符（粗略估计）
	maxChars := maxTokens * 4

	// 构建知识库上下文，限制长度
	knowledgeContext := i.BuildKnowledgeContextWithLimit(documents, maxChars)

	// 组合Prompt
	if basePrompt == "" {
		return knowledgeContext
	}

	result := fmt.Sprintf("%s\n\n%s\n\n%s",
		i.buildSectionHeader("数据库Schema"),
		basePrompt,
		knowledgeContext,
	)

	// 如果结果超过限制，进行截断
	if len(result) > maxChars {
		// 优先保留知识库内容
		result = basePrompt + "\n\n" + knowledgeContext
		if len(result) > maxChars {
			result = basePrompt + "\n\n" + truncateString(knowledgeContext, maxChars-len(basePrompt)-20)
		}
	}

	return result
}

// BuildKnowledgeContext 构建知识库上下文字符串
func (i *Injector) BuildKnowledgeContext(documents []Document) string {
	if len(documents) == 0 {
		return ""
	}

	var builder strings.Builder

	// 添加知识库标题
	builder.WriteString(i.buildSectionHeader("业务知识库"))
	builder.WriteString("\n")

	// 添加每个文档的内容
	for i, doc := range documents {
		if !doc.IsValid() {
			continue
		}

		// 添加文档标题
		builder.WriteString(fmt.Sprintf("### %s\n", doc.Title))

		// 添加文档内容
		builder.WriteString(doc.Content)

		// 文档之间添加分隔（除了最后一个）
		if i < len(documents)-1 {
			builder.WriteString("\n\n")
		}
	}

	return builder.String()
}

// BuildKnowledgeContextWithLimit 构建限制长度的知识库上下文
func (i *Injector) BuildKnowledgeContextWithLimit(documents []Document, maxChars int) string {
	var builder strings.Builder

	builder.WriteString(i.buildSectionHeader("业务知识库"))
	builder.WriteString("\n")

	currentLength := 0
	for i, doc := range documents {
		if !doc.IsValid() {
			continue
		}

		// 估算当前文档的长度
		docLength := len(doc.Title) + len(doc.Content) + 10 // +10 用于格式化

		// 如果添加此文档会超过限制，停止添加
		if currentLength+docLength > maxChars {
			if i == 0 {
				// 至少添加第一个文档的部分内容
				remainingChars := maxChars - currentLength - 20
				if remainingChars > 0 {
					builder.WriteString(fmt.Sprintf("### %s\n", doc.Title))
					builder.WriteString(truncateString(doc.Content, remainingChars))
				}
			}
			break
		}

		// 添加文档
		builder.WriteString(fmt.Sprintf("### %s\n", doc.Title))
		builder.WriteString(doc.Content)

		if i < len(documents)-1 && currentLength+docLength < maxChars {
			builder.WriteString("\n\n")
		}

		currentLength += docLength
	}

	return builder.String()
}

// buildSectionHeader 构建节标题
func (i *Injector) buildSectionHeader(title string) string {
	return fmt.Sprintf("━━━━━ %s ━━━━━", title)
}

// SetMaxTokens 设置最大Token数
func (i *Injector) SetMaxTokens(maxTokens int) {
	i.maxTokens = maxTokens
}

// SetEnabled 设置是否启用注入
func (i *Injector) SetEnabled(enabled bool) {
	i.enabled = enabled
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
