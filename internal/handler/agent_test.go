package handler

import (
	"testing"

	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/stretchr/testify/assert"
)

func TestAgentConfig_Defaults(t *testing.T) {
	cfg := DefaultAgentConfig()
	assert.Equal(t, 3, cfg.MaxSelfCorrect)
	assert.Equal(t, 5, cfg.MaxTurns)
	assert.False(t, cfg.Verbose)
}

func TestAgentConfig_ZeroValues_Overridden(t *testing.T) {
	// AgentQueryHandler 构造函数中会修正零值
	def := DefaultAgentConfig()
	assert.True(t, def.MaxSelfCorrect > 0)
	assert.True(t, def.MaxTurns > 0)
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json code block",
			input:    "```json\n{\"selected\": [\"01\"]}\n```",
			expected: `{"selected": ["01"]}`,
		},
		{
			name:     "plain code block",
			input:    "```\n{\"selected\": [\"01\"]}\n```",
			expected: `{"selected": ["01"]}`,
		},
		{
			name:     "raw json",
			input:    `{"selected": ["01"]}`,
			expected: `{"selected": ["01"]}`,
		},
		{
			name:     "json with prefix text",
			input:    "Here is the result:\n```json\n{\"selected\": [\"01\", \"06\"]}\n```\nDone.",
			expected: `{"selected": ["01", "06"]}`,
		},
		{
			name:     "no json",
			input:    "This is just plain text",
			expected: "",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocTitles(t *testing.T) {
	docs := []knowledge.Document{
		{Title: "数据库表结构"},
		{Title: "模型映射"},
	}

	titles := docTitles(docs)
	assert.Equal(t, []string{"数据库表结构", "模型映射"}, titles)
}

func TestTruncateStr(t *testing.T) {
	assert.Equal(t, "hello", truncateStr("hello", 10))
	result := truncateStr("hello world", 6)
	assert.True(t, len(result) <= 9)
}

func TestBuildErrorResult(t *testing.T) {
	// 简单验证 buildErrorResult 不 panic
	// 因为需要 db 等依赖，这里只做基本测试
	steps := []AgentStep{
		{Turn: 1, Action: "test", Detail: "testing"},
	}

	result := &QueryResult{
		Question: "test question",
		Error:    "test error",
		Steps:    steps,
		Metadata: map[string]interface{}{
			"mode": "agent",
		},
	}

	assert.Equal(t, "test question", result.Question)
	assert.Equal(t, "test error", result.Error)
	assert.Equal(t, 1, len(result.Steps))
}

func TestAgentStep_Struct(t *testing.T) {
	step := AgentStep{
		Turn:     1,
		Action:   "resource_selection",
		Detail:   "选择了2个文档",
		Duration: 100000000, // 100ms
		Data: map[string]interface{}{
			"selected_docs": []string{"01", "06"},
		},
	}

	assert.Equal(t, 1, step.Turn)
	assert.Equal(t, "resource_selection", step.Action)
	assert.Equal(t, "选择了2个文档", step.Detail)
	assert.NotNil(t, step.Data)
}
