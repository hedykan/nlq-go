package utils

import (
	"unicode/utf8"
)

// TruncateString 截断字符串到指定长度（按字符数，支持多字节字符）
// 如果字符串长度超过 maxLen，则截断并添加 "..." 后缀
func TruncateString(s string, maxLen int) string {
	// 获取字符串的字符数（rune count）
	runeCount := utf8.RuneCountInString(s)
	if runeCount <= maxLen {
		return s
	}

	// 计算需要保留的字符数（为...预留3个字符位置）
	truncateTo := maxLen - 3
	if truncateTo <= 0 {
		return "..."
	}

	// 按字符截断
	count := 0
	var result []rune
	for _, r := range s {
		if count >= truncateTo {
			break
		}
		result = append(result, r)
		count++
	}

	return string(result) + "..."
}
