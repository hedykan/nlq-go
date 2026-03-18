package utils

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		maxLen    int
		expected  string
	}{
		{
			name:     "短字符串不截断",
			s:        "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "长字符串截断",
			s:        "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "正好等于最大长度",
			s:        "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "空字符串",
			s:        "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "中文字符串",
			s:        "你好世界这是一个很长的字符串",
			maxLen:   10,
			expected: "你好世界这是一...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.s, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q; 期望 %q", tt.s, tt.maxLen, result, tt.expected)
			}
		})
	}
}
