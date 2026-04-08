package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestNewConversation(t *testing.T) {
	c := NewConversation("You are a helpful assistant.")
	assert.Equal(t, 1, len(c.Messages()))
	assert.Equal(t, 0, c.TurnCount())
}

func TestConversation_AddUser(t *testing.T) {
	c := NewConversation("system")
	c.AddUser("hello")

	assert.Equal(t, 1, c.TurnCount())
	assert.Equal(t, 2, len(c.Messages()))
}

func TestConversation_AddAssistant(t *testing.T) {
	c := NewConversation("system")
	c.AddUser("hello")
	c.AddAssistant("hi there")

	assert.Equal(t, 1, c.TurnCount())
	assert.Equal(t, 3, len(c.Messages()))
	assert.Equal(t, "hi there", c.LastAssistantContent())
}

func TestConversation_LastAssistantContent_Empty(t *testing.T) {
	c := NewConversation("system")
	assert.Equal(t, "", c.LastAssistantContent())
}

func TestConversation_ReplaceLastUser(t *testing.T) {
	c := NewConversation("system")
	c.AddUser("original question")
	c.AddAssistant("some response")
	c.AddUser("updated question")

	err := c.ReplaceLastUser("final question")
	assert.NoError(t, err)

	msgs := c.Messages()
	lastUser := msgs[len(msgs)-1]
	assert.Equal(t, llms.ChatMessageTypeHuman, lastUser.Role)
}

func TestConversation_ReplaceLastUser_NoUser(t *testing.T) {
	c := NewConversation("system")
	err := c.ReplaceLastUser("something")
	assert.Error(t, err)
}

func TestConversation_Messages(t *testing.T) {
	c := NewConversation("system prompt")
	c.AddUser("user message")

	msgs := c.Messages()
	assert.Equal(t, 2, len(msgs))
	// 验证返回的是副本
	msgs[0] = llms.TextParts(llms.ChatMessageTypeHuman, "modified")
	assert.Equal(t, 2, len(c.Messages()))
	assert.Equal(t, llms.ChatMessageTypeSystem, c.Messages()[0].Role)
}

func TestConversation_TotalContentLength(t *testing.T) {
	c := NewConversation("system prompt here")
	c.AddUser("user message")

	length := c.TotalContentLength()
	assert.Equal(t, len("system prompt here")+len("user message"), length)
}

func TestConversation_String(t *testing.T) {
	c := NewConversation("system")
	c.AddUser("hello")
	c.AddAssistant("hi")

	s := c.String()
	assert.Contains(t, s, "turns=1")
	assert.Contains(t, s, "messages=3")
}

func TestConversation_EmptySystem(t *testing.T) {
	c := NewConversation("")
	assert.Equal(t, 0, len(c.Messages()))
	assert.Equal(t, 0, c.TurnCount())
}
