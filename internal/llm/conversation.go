package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

// Conversation 对话上下文管理器
// 维护 LLM 多轮对话的消息历史，每轮调用后自动累积
type Conversation struct {
	messages []llms.MessageContent
}

// NewConversation 创建新对话（预置 system prompt）
func NewConversation(systemPrompt string) *Conversation {
	c := &Conversation{
		messages: make([]llms.MessageContent, 0),
	}
	if systemPrompt != "" {
		c.messages = append(c.messages, llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt))
	}
	return c
}

// AddUser 追加用户消息
func (c *Conversation) AddUser(content string) {
	c.messages = append(c.messages, llms.TextParts(llms.ChatMessageTypeHuman, content))
}

// AddAssistant 追加助手回复
func (c *Conversation) AddAssistant(content string) {
	c.messages = append(c.messages, llms.TextParts(llms.ChatMessageTypeAI, content))
}

// Messages 获取完整消息列表（传给 LLM API）
func (c *Conversation) Messages() []llms.MessageContent {
	result := make([]llms.MessageContent, len(c.messages))
	copy(result, c.messages)
	return result
}

// LastAssistantContent 获取最后一次助手回复内容
func (c *Conversation) LastAssistantContent() string {
	for i := len(c.messages) - 1; i >= 0; i-- {
		msg := c.messages[i]
		if msg.Role == llms.ChatMessageTypeAI {
			for _, part := range msg.Parts {
				if text, ok := part.(llms.TextContent); ok {
					return text.Text
				}
			}
		}
	}
	return ""
}

// TurnCount 当前对话轮次数（不含 system）
func (c *Conversation) TurnCount() int {
	count := 0
	for _, msg := range c.messages {
		if msg.Role == llms.ChatMessageTypeHuman {
			count++
		}
	}
	return count
}

// TotalContentLength 当前所有消息内容总字符数
func (c *Conversation) TotalContentLength() int {
	total := 0
	for _, msg := range c.messages {
		for _, part := range msg.Parts {
			if text, ok := part.(llms.TextContent); ok {
				total += len(text.Text)
			}
		}
	}
	return total
}

// ReplaceLastUser 替换最后一条用户消息（用于自检修正时更新 prompt）
func (c *Conversation) ReplaceLastUser(content string) error {
	for i := len(c.messages) - 1; i >= 0; i-- {
		if c.messages[i].Role == llms.ChatMessageTypeHuman {
			c.messages[i] = llms.TextParts(llms.ChatMessageTypeHuman, content)
			return nil
		}
	}
	return fmt.Errorf("no user message found to replace")
}

// String 返回对话摘要（用于日志）
func (c *Conversation) String() string {
	return fmt.Sprintf("Conversation{turns=%d, messages=%d, totalChars=%d}",
		c.TurnCount(), len(c.messages), c.TotalContentLength())
}
