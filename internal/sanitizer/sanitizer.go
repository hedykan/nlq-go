package sanitizer

import (
	"fmt"
	"regexp"
	"strings"
)

// Sanitizer 数据脱敏器
type Sanitizer struct {
	// 预编译正则表达式（性能优化）
	emailRegex   *regexp.Regexp
	phoneRegex   *regexp.Regexp
	idCardRegex  *regexp.Regexp
	ipRegex      *regexp.Regexp
}

// NewSanitizer 创建新的脱敏器
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		emailRegex:  regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phoneRegex:  regexp.MustCompile(`1[3-9]\d{9}`),
		idCardRegex: regexp.MustCompile(`\d{17}[\dXx]`),
		ipRegex:     regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`),
	}
}

// SanitizeEmail 脱敏邮箱地址
// 规则: 将邮箱替换为 ***@***.***
func (s *Sanitizer) SanitizeEmail(text string) string {
	return s.emailRegex.ReplaceAllString(text, "***@***.***")
}

// SanitizePhone 脱敏手机号
// 规则: 保留前3位和后4位，中间用****替换
// 例如: 13812345678 -> 138****5678
func (s *Sanitizer) SanitizePhone(text string) string {
	return s.phoneRegex.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) == 11 {
			return match[:3] + "****" + match[7:]
		}
		return match
	})
}

// SanitizeIDCard 脱敏身份证号
// 规则: 只保留后4位，前面用*替换
// 例如: 110101199001011234 -> ************1234
func (s *Sanitizer) SanitizeIDCard(text string) string {
	return s.idCardRegex.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) == 18 {
			return "************" + match[14:]
		}
		return match
	})
}

// SanitizeIP 脱敏IP地址
// 规则: 保留前缀中每段的第一位和完整后两位
// 例如: 192.168.1.1 -> ***.***.1.1
func (s *Sanitizer) SanitizeIP(text string) string {
	return s.ipRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := strings.Split(match, ".")
		if len(parts) == 4 {
			// 第一段和第二段脱敏
			part1 := maskIPPart(parts[0])
			part2 := maskIPPart(parts[1])
			return fmt.Sprintf("%s.%s.%s.%s", part1, part2, parts[2], parts[3])
		}
		return match
	})
}

// maskIPPart 脱敏IP的一段（用于前两段）
// 规则: 长度>=2时完全脱敏为***，长度==1时脱敏为*
// 例如: 192 -> ***, 8 -> *
func maskIPPart(part string) string {
	if len(part) > 1 {
		return "***"
	}
	if len(part) == 1 {
		return "*"
	}
	return part
}

// sanitizeAll 内部方法：按优先级应用所有脱敏规则
// 顺序: 身份证 > 手机号 > 邮箱 > IP（避免互相干扰）
func (s *Sanitizer) sanitizeAll(text string) string {
	result := s.SanitizeIDCard(text)
	result = s.SanitizePhone(result)
	result = s.SanitizeEmail(result)
	result = s.SanitizeIP(result)
	return result
}

// SanitizeSQL 脱敏SQL语句中的敏感信息
func (s *Sanitizer) SanitizeSQL(sql string) string {
	return s.sanitizeAll(sql)
}

// SanitizeQuestion 脱敏问题文本中的敏感信息
func (s *Sanitizer) SanitizeQuestion(question string) string {
	return s.sanitizeAll(question)
}

// SanitizeAll 综合脱敏（应用所有脱敏规则）
func (s *Sanitizer) SanitizeAll(text string) string {
	return s.sanitizeAll(text)
}
