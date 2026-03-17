package security

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Firewall SQL安全防火墙
type Firewall struct {
	blockedKeywords []string
	allowedPrefixes []string
	checkComments   bool
	checkSemicolon  bool
}

// NewFirewall 创建新的SQL防火墙
func NewFirewall() *Firewall {
	return &Firewall{
		blockedKeywords: []string{
			"DROP", "DELETE", "UPDATE", "INSERT",
			"ALTER", "CREATE", "TRUNCATE", "GRANT",
			"REVOKE", "EXECUTE", "CALL", "EXPLAIN",
			"SHOW", "DESCRIBE", "DESC", "USE", "SET",
			"LOCK", "UNLOCK", "REPLACE", "LOAD",
		},
		allowedPrefixes: []string{
			"SELECT", "WITH", // WITH用于CTE (Common Table Expressions)
		},
		checkComments:  true,
		checkSemicolon: true,
	}
}

// Check 严格检查SQL语句，只允许SELECT查询
func (f *Firewall) Check(sql string) error {
	// 1. 去除前后空格
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return errors.New("SQL语句不能为空")
	}

	// 2. 检查括号平衡
	if !f.hasBalancedParentheses(sql) {
		return errors.New("SQL语句括号不匹配")
	}

	// 3. 检查是否包含注释
	if f.checkComments && f.containsComments(sql) {
		return errors.New("安全检查失败：检测到SQL注释")
	}

	// 4. 检查是否包含多个语句（分号）
	if f.checkSemicolon && f.hasMultipleStatements(sql) {
		return errors.New("安全检查失败：检测到多语句执行")
	}

	// 5. 转换为大写进行关键字检查
	upperSQL := f.toUpperIgnoreStringLiterals(sql)

	// 6. 检查是否以允许的前缀开头
	if !f.startsWithAllowedPrefix(upperSQL) {
		return fmt.Errorf("安全检查失败：只允许SELECT查询语句")
	}

	// 7. 检查是否包含危险关键字
	if err := f.checkDangerousKeywords(upperSQL); err != nil {
		return err
	}

	return nil
}

// IsReadOnlyQuery 验证是否为只读查询
func (f *Firewall) IsReadOnlyQuery(sql string) bool {
	upperSQL := f.toUpperIgnoreStringLiterals(strings.TrimSpace(sql))
	return f.startsWithAllowedPrefix(upperSQL)
}

// GetBlockedKeywords 获取被阻止的关键字列表
func (f *Firewall) GetBlockedKeywords() []string {
	return f.blockedKeywords
}

// GetAllowedPrefixes 获取允许的前缀列表
func (f *Firewall) GetAllowedPrefixes() []string {
	return f.allowedPrefixes
}

// hasBalancedParentheses 检查SQL语句中的括号是否匹配
func (f *Firewall) hasBalancedParentheses(sql string) bool {
	count := 0
	inString := false
	stringChar := rune(0)

	for i, ch := range sql {
		// 处理字符串字面量
		if ch == '\'' || ch == '"' || ch == '`' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				// 检查是否是转义字符
				if i > 0 && sql[i-1] != '\\' {
					inString = false
					stringChar = rune(0)
				}
			}
		}

		// 在字符串字面量中跳过括号检查
		if inString {
			continue
		}

		if ch == '(' {
			count++
		} else if ch == ')' {
			count--
			if count < 0 {
				return false
			}
		}
	}

	return count == 0
}

// containsComments 检查是否包含SQL注释
func (f *Firewall) containsComments(sql string) bool {
	// 检查单行注释 --
	if strings.Contains(sql, "--") {
		return true
	}

	// 检查单行注释 #
	if strings.Contains(sql, "#") {
		return true
	}

	// 检查多行注释 /* */
	if strings.Contains(sql, "/*") {
		return true
	}

	return false
}

// hasMultipleStatements 检查是否包含多个语句
func (f *Firewall) hasMultipleStatements(sql string) bool {
	// 移除字符串字面量中的内容，避免误判
	cleanedSQL := f.removeStringLiterals(sql)

	// 检查分号数量
	semicolonCount := strings.Count(cleanedSQL, ";")

	// 如果有分号，检查是否有实际内容在分号后面
	if semicolonCount > 0 {
		// 分割SQL，检查每个部分
		parts := strings.Split(cleanedSQL, ";")

		// 如果只有一个分号，检查是否在末尾
		if len(parts) == 2 {
			// 第一部分应该有内容，第二部分应该只有空白字符
			firstPart := strings.TrimSpace(parts[0])
			secondPart := strings.TrimSpace(parts[1])

			// 如果第二部分不是空的（有实际内容），则拒绝
			if secondPart != "" {
				return true
			}

			// 如果第一部分是空的，也拒绝
			if firstPart == "" {
				return true
			}
		} else {
			// 多于一个分号，拒绝
			return true
		}
	}

	return false
}

// startsWithAllowedPrefix 检查是否以允许的前缀开头
func (f *Firewall) startsWithAllowedPrefix(sql string) bool {
	sql = strings.TrimSpace(sql)

	for _, prefix := range f.allowedPrefixes {
		if strings.HasPrefix(sql, prefix) {
			// 检查前缀后面是否是空白字符或括号
			remainder := strings.TrimPrefix(sql, prefix)
			if len(remainder) == 0 {
				// 只有前缀，不完整
				continue
			}

			nextChar := rune(remainder[0])
			if unicode.IsSpace(nextChar) || nextChar == '(' {
				return true
			}
		}
	}

	return false
}

// checkDangerousKeywords 检查是否包含危险关键字（智能上下文感知）
func (f *Firewall) checkDangerousKeywords(sql string) error {
	// 将SQL按空白字符分割成单词
	words := f.extractSQLWords(sql)

	for i, word := range words {
		upperWord := strings.ToUpper(word)
		for _, keyword := range f.blockedKeywords {
			if upperWord == keyword {
				// 检查是否在安全的上下文中
				if f.isSafeContext(word, words, i, sql) {
					continue // 在安全上下文中，跳过检查
				}
				return fmt.Errorf("安全检查失败：检测到危险关键字 '%s'", keyword)
			}
		}
	}

	return nil
}

// isSafeContext 检查关键字是否在安全的上下文中
func (f *Firewall) isSafeContext(word string, words []string, index int, sql string) bool {
	word = strings.ToUpper(word)

	// DESC或ASC在ORDER BY后面是安全的
	if word == "DESC" || word == "ASC" {
		// 向前搜索，查找是否有 ORDER BY ... DESC/ASC 模式
		// 例如: ORDER BY name DESC，DESC的index可能是7，需要向前找到ORDER和BY
		foundBy := false
		foundOrder := false

		for i := index - 1; i >= 0; i-- {
			prevWord := strings.ToUpper(words[i])
			if !foundBy && prevWord == "BY" {
				foundBy = true
			} else if foundBy && prevWord == "ORDER" {
				foundOrder = true
				break
			} else if prevWord != "BY" && foundBy {
				// 如果已经找到BY但前一个单词不是ORDER，说明不是ORDER BY模式
				// 但是可能有列名在中间，继续搜索
				continue
			}
		}

		if foundOrder && foundBy {
			return true // ORDER BY ... DESC/ASC 是安全的
		}
	}

	// SHOW TABLES在某些上下文中可能是安全的（但这里我们保持保守）
	// 暂时保持严格的检查，未来可以扩展

	return false
}

// extractSQLWords 从SQL中提取单词（忽略字符串字面量）
func (f *Firewall) extractSQLWords(sql string) []string {
	var words []string
	var currentWord strings.Builder
	inString := false
	stringChar := rune(0)

	for i, ch := range sql {
		// 处理字符串字面量
		if ch == '\'' || ch == '"' || ch == '`' {
			if !inString {
				// 保存当前单词
				if currentWord.Len() > 0 {
					words = append(words, currentWord.String())
					currentWord.Reset()
				}
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				// 检查是否是转义字符
				if i > 0 && sql[i-1] != '\\' {
					inString = false
					stringChar = rune(0)
				}
			}
		}

		// 在字符串字面量中跳过单词提取
		if inString {
			continue
		}

		// 检查是否是单词分隔符
		if unicode.IsSpace(ch) || ch == ',' || ch == '(' || ch == ')' || ch == ';' {
			// 保存当前单词
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		} else {
			currentWord.WriteRune(ch)
		}
	}

	// 保存最后一个单词
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// toUpperIgnoreStringLiterals 转换为大写，但忽略字符串字面量
func (f *Firewall) toUpperIgnoreStringLiterals(sql string) string {
	var result strings.Builder
	inString := false
	stringChar := rune(0)

	for i, ch := range sql {
		// 处理字符串字面量
		if ch == '\'' || ch == '"' || ch == '`' {
			if !inString {
				result.WriteString(strings.ToUpper(sql[i:result.Len()]))
				result.WriteRune(ch)
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				// 检查是否是转义字符
				if i > 0 && sql[i-1] != '\\' {
					result.WriteRune(ch)
					inString = false
					stringChar = rune(0)
				} else {
					result.WriteRune(ch)
				}
			} else {
				result.WriteRune(ch)
			}
		} else if inString {
			// 在字符串字面量中，保持原样
			result.WriteRune(ch)
		} else {
			// 不在字符串字面量中，转换为大写
			result.WriteRune(unicode.ToUpper(ch))
		}
	}

	// 处理最后的部分
	if inString {
		// 未闭合的字符串字面量，保持原样
		result.WriteString(sql[result.Len():])
	} else {
		// 转换剩余部分为大写
		result.WriteString(strings.ToUpper(sql[result.Len():]))
	}

	return result.String()
}

// removeStringLiterals 移除字符串字面量
func (f *Firewall) removeStringLiterals(sql string) string {
	// 使用正则表达式移除字符串字面量
	re := regexp.MustCompile(`'[^']*'|"[^"]*"|` + "`[^`]*`")
	return re.ReplaceAllString(sql, "")
}
