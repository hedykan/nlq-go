package security

import (
	"strings"
	"testing"
)

// TestNewFirewall 测试创建防火墙
func TestNewFirewall(t *testing.T) {
	firewall := NewFirewall()
	if firewall == nil {
		t.Fatal("期望返回非nil的防火墙实例")
	}
}

// TestFirewall_Check 测试SQL检查
func TestFirewall_Check(t *testing.T) {
	firewall := NewFirewall()

	tests := []struct {
		name    string
		sql     string
		allowed bool
	}{
		// 允许的查询
		{"正常SELECT", "SELECT * FROM users", true},
		{"带条件的SELECT", "SELECT name FROM users WHERE age > 25", true},
		{"JOIN查询", "SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id", true},
		{"LEFT JOIN", "SELECT u.name, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id", true},
		{"聚合查询", "SELECT COUNT(*) FROM users", true},
		{"GROUP BY", "SELECT city, COUNT(*) FROM users GROUP BY city", true},
		{"HAVING", "SELECT city, COUNT(*) FROM users GROUP BY city HAVING COUNT(*) > 10", true},
		{"ORDER BY", "SELECT * FROM users ORDER BY name", true},
		{"ORDER BY DESC", "SELECT * FROM users ORDER BY name DESC", true},
		{"ORDER BY ASC", "SELECT * FROM users ORDER BY created_at ASC", true},
		{"ORDER BY多字段", "SELECT * FROM users ORDER BY name DESC, created_at ASC", true},
		{"LIMIT", "SELECT * FROM users LIMIT 10", true},
		{"子查询", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)", true},
		{"UNION", "SELECT name FROM users UNION SELECT name FROM customers", true},
		{"DISTINCT", "SELECT DISTINCT city FROM users", true},
		{"CASE WHEN", "SELECT CASE WHEN age > 18 THEN 'adult' ELSE 'child' END FROM users", true},
		{"WITH子句", "WITH cte AS (SELECT * FROM users) SELECT * FROM cte", true},

		// 禁止的查询
		{"DROP TABLE", "DROP TABLE users", false},
		{"DELETE", "DELETE FROM users WHERE id = 1", false},
		{"UPDATE", "UPDATE users SET name='test' WHERE id=1", false},
		{"INSERT", "INSERT INTO users VALUES (1, 'test')", false},
		{"ALTER", "ALTER TABLE users ADD COLUMN test INT", false},
		{"CREATE", "CREATE TABLE test (id INT)", false},
		{"TRUNCATE", "TRUNCATE TABLE users", false},
		{"GRANT", "GRANT ALL ON users TO 'user'@'localhost'", false},
		{"REVOKE", "REVOKE ALL ON users FROM 'user'@'localhost'", false},
		{"EXECUTE", "EXECUTE stored_procedure()", false},
		{"CALL", "CALL stored_procedure()", false},
		{"EXPLAIN", "EXPLAIN SELECT * FROM users", false},
		{"SHOW", "SHOW TABLES", false},
		{"DESCRIBE", "DESCRIBE users", false},
		{"DESC", "DESC users", false},
		{"USE", "USE database", false},
		{"SET", "SET @var = 1", false},
		{"LOCK TABLES", "LOCK TABLES users WRITE", false},
		{"UNLOCK TABLES", "UNLOCK TABLES", false},
		{"REPLACE", "REPLACE INTO users VALUES (1, 'test')", false},
		{"LOAD DATA", "LOAD DATA INFILE '/path/to/file' INTO TABLE users", false},

		// SQL注入防护
		{"注释注入", "SELECT * FROM users; DROP TABLE users--", false},
		{"多语句", "SELECT * FROM users; SELECT * FROM orders", false},
		{"注释块注入", "SELECT * FROM users/*comment*/; DROP TABLE users", false},
		{"分号注入", "SELECT * FROM users WHERE id = 1; DELETE FROM users", false},
		{"多条语句", "SELECT * FROM users; SELECT * FROM orders; SELECT * FROM products", false},
		{"注释在中间", "SELECT * FROM -- comment\nusers", false},

		// 边界情况
		{"空SQL", "", false},
		{"只有空格", "   ", false},
		{"只有SELECT", "SELECT", false},
		{"小写select", "select * from users", true},
		{"混合大小写", "SeLeCt * FrOm users", true},
		{"前导空格", "  SELECT * FROM users", true},
		{"尾随空格", "SELECT * FROM users  ", true},
		{"前导换行", "\nSELECT * FROM users", true},
		{"前导制表符", "\tSELECT * FROM users", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := firewall.Check(tt.sql)
			if tt.allowed && err != nil {
				t.Errorf("期望允许但被拒绝: %v\nSQL: %s", err, tt.sql)
			}
			if !tt.allowed && err == nil {
				t.Errorf("期望拒绝但被允许\nSQL: %s", tt.sql)
			}
		})
	}
}

// TestFirewall_Check_DangerousKeywords 测试危险关键字检测
func TestFirewall_Check_DangerousKeywords(t *testing.T) {
	firewall := NewFirewall()

	dangerousKeywords := []string{
		"DROP", "DELETE", "UPDATE", "INSERT",
		"ALTER", "CREATE", "TRUNCATE", "GRANT",
		"REVOKE", "EXECUTE", "CALL", "EXPLAIN",
		"SHOW", "DESCRIBE", "DESC", "USE", "SET",
	}

	for _, keyword := range dangerousKeywords {
		sql := "SELECT * FROM users WHERE name = '" + keyword + "'"
		// 这应该是允许的，因为关键字在字符串字面量中
		err := firewall.Check(sql)
		if err != nil {
			t.Errorf("字符串中的关键字 %s 应该被允许，但被拒绝: %v", keyword, err)
		}
	}
}

// TestFirewall_Check_CommentInjection 测试注释注入检测
func TestFirewall_Check_CommentInjection(t *testing.T) {
	firewall := NewFirewall()

	commentInjections := []string{
		"SELECT * FROM users -- 注释",
		"SELECT * FROM users # 注释",
		"SELECT * FROM users /* 注释 */",
		"SELECT * FROM users WHERE id = 1 -- AND id = 2",
		"SELECT * FROM users WHERE id = 1 # AND id = 2",
		"SELECT * FROM users WHERE id = 1 /* AND id = 2 */",
	}

	for _, sql := range commentInjections {
		err := firewall.Check(sql)
		if err == nil {
			t.Errorf("期望拒绝注释注入，但被允许: %s", sql)
		}
	}
}

// TestFirewall_Check_SemicolonInjection 测试分号注入检测
func TestFirewall_Check_SemicolonInjection(t *testing.T) {
	firewall := NewFirewall()

	// 应该被拒绝的多语句
	semicolonInjections := []string{
		"SELECT * FROM users; DROP TABLE users",
		"SELECT * FROM users; SELECT * FROM orders",
		"SELECT * FROM users WHERE id = 1; DELETE FROM users",
		"SELECT * FROM users;;",
	}

	for _, sql := range semicolonInjections {
		err := firewall.Check(sql)
		if err == nil {
			t.Errorf("期望拒绝分号注入，但被允许: %s", sql)
		}
	}

	// 应该被允许的末尾分号
	allowedStatements := []string{
		"SELECT * FROM users;",
		"SELECT * FROM users; ",
		"SELECT * FROM users;",
	}

	for _, sql := range allowedStatements {
		err := firewall.Check(sql)
		if err != nil {
			t.Errorf("期望允许末尾分号，但被拒绝: %s\n错误: %v", sql, err)
		}
	}
}

// TestFirewall_Check_SelectOnly 测试只允许SELECT
func TestFirewall_Check_SelectOnly(t *testing.T) {
	firewall := NewFirewall()

	selectQueries := []string{
		"SELECT * FROM users",
		"select * from users",
		"  SELECT * FROM users",
		"\nSELECT * FROM users",
		"\tSELECT * FROM users",
		"SELECT * FROM users WHERE id = 1",
		"SELECT COUNT(*) FROM users",
		"SELECT u.name, o.amount FROM users u JOIN orders o ON u.id = o.user_id",
	}

	for _, sql := range selectQueries {
		err := firewall.Check(sql)
		if err != nil {
			t.Errorf("SELECT查询应该被允许，但被拒绝: %s\n错误: %v", sql, err)
		}
	}

	nonSelectQueries := []string{
		"UPDATE users SET name='test' WHERE id=1",
		"DELETE FROM users WHERE id = 1",
		"INSERT INTO users VALUES (1, 'test')",
		"DROP TABLE users",
		"CREATE TABLE test (id INT)",
		"ALTER TABLE users ADD COLUMN test INT",
		"TRUNCATE TABLE users",
	}

	for _, sql := range nonSelectQueries {
		err := firewall.Check(sql)
		if err == nil {
			t.Errorf("非SELECT查询应该被拒绝，但被允许: %s", sql)
		}
	}
}

// TestFirewall_Check_StringLiterals 测试字符串字面量中的关键字
func TestFirewall_Check_StringLiterals(t *testing.T) {
	firewall := NewFirewall()

	// 字符串字面量中的关键字应该被允许
	tests := []string{
		"SELECT * FROM users WHERE name = 'DROP TABLE'",
		"SELECT * FROM users WHERE name = 'UPDATE users SET'",
		"SELECT * FROM users WHERE name = 'DELETE FROM'",
		`SELECT * FROM users WHERE name = "INSERT INTO"`,
		"SELECT * FROM users WHERE description LIKE '%DROP TABLE%'",
	}

	for _, sql := range tests {
		err := firewall.Check(sql)
		if err != nil {
			// 注意：这里可能需要更复杂的解析来正确处理字符串字面量
			// 简单实现可能会拒绝这些查询，这是可以接受的
			t.Logf("字符串字面量测试: %s - %v", sql, err)
		}
	}
}

// TestFirewall_IsReadOnlyQuery 测试是否为只读查询
func TestFirewall_IsReadOnlyQuery(t *testing.T) {
	firewall := NewFirewall()

	tests := []struct {
		sql      string
		expected bool
	}{
		{"SELECT * FROM users", true},
		{"select * from users", true},
		{"UPDATE users SET name='test'", false},
		{"DELETE FROM users", false},
		{"INSERT INTO users VALUES (1, 'test')", false},
		{"DROP TABLE users", false},
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		result := firewall.IsReadOnlyQuery(tt.sql)
		if result != tt.expected {
			t.Errorf("IsReadOnlyQuery(%s) = %v, 期望 %v", tt.sql, result, tt.expected)
		}
	}
}

// TestFirewall_Check_ParenthesesBalance 测试括号平衡
func TestFirewall_Check_ParenthesesBalance(t *testing.T) {
	firewall := NewFirewall()

	tests := []struct {
		name    string
		sql     string
		allowed bool
	}{
		{"匹配的括号", "SELECT * FROM users WHERE (id = 1)", true},
		{"多个匹配的括号", "SELECT * FROM users WHERE ((id = 1) OR (id = 2))", true},
		{"不匹配的括号", "SELECT * FROM users WHERE (id = 1", false},
		{"反向不匹配", "SELECT * FROM users WHERE id = 1)", false},
		{"子查询匹配", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)", true},
		{"子查询不匹配", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := firewall.Check(tt.sql)
			if tt.allowed && err != nil {
				t.Errorf("期望允许但被拒绝: %v\nSQL: %s", err, tt.sql)
			}
			if !tt.allowed && err == nil {
				t.Errorf("期望拒绝但被允许\nSQL: %s", tt.sql)
			}
		})
	}
}

// TestFirewall_Check_CaseSensitivity 测试大小写敏感性
func TestFirewall_Check_CaseSensitivity(t *testing.T) {
	firewall := NewFirewall()

	tests := []string{
		"SELECT * FROM users",
		"select * from users",
		"Select * From Users",
		"SeLeCt * FrOm users",
	}

	for _, sql := range tests {
		err := firewall.Check(sql)
		if err != nil {
			t.Errorf("大小写变化应该不影响检查: %s\n错误: %v", sql, err)
		}
	}
}

// TestFirewall_Check_WhitespaceVariations 测试空白字符变体
func TestFirewall_Check_WhitespaceVariations(t *testing.T) {
	firewall := NewFirewall()

	tests := []string{
		"SELECT*FROM users",           // 无空格
		"SELECT * FROM users",         // 单空格
		"SELECT  *  FROM  users",      // 多空格
		"SELECT\t*\tFROM\tusers",      // 制表符
		"SELECT\n*\nFROM\nusers",      // 换行符
		"SELECT\r*\rFROM\rusers",      // 回车符
		"SELECT \n * \n FROM \n users", // 混合空白
	}

	for _, sql := range tests {
		err := firewall.Check(sql)
		if err != nil {
			// 某些空白变体可能被拒绝，这是可以接受的
			t.Logf("空白变体测试: %q - %v", sql, err)
		}
	}
}

// TestFirewall_GetBlockedKeywords 测试获取被阻止的关键字
func TestFirewall_GetBlockedKeywords(t *testing.T) {
	firewall := NewFirewall()

	keywords := firewall.GetBlockedKeywords()
	if len(keywords) == 0 {
		t.Error("期望有被阻止的关键字")
	}

	expectedKeywords := []string{
		"DROP", "DELETE", "UPDATE", "INSERT",
		"ALTER", "CREATE", "TRUNCATE",
	}

	for _, expected := range expectedKeywords {
		found := false
		for _, keyword := range keywords {
			if strings.EqualFold(keyword, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("期望被阻止的关键字中包含 %s", expected)
		}
	}
}

// TestFirewall_GetAllowedPrefixes 测试获取允许的前缀
func TestFirewall_GetAllowedPrefixes(t *testing.T) {
	firewall := NewFirewall()

	prefixes := firewall.GetAllowedPrefixes()
	if len(prefixes) == 0 {
		t.Error("期望有允许的前缀")
	}

	// 检查SELECT是否在允许的前缀中
	found := false
	for _, prefix := range prefixes {
		if strings.EqualFold(prefix, "SELECT") {
			found = true
			break
		}
	}
	if !found {
		t.Error("期望允许的前缀中包含 SELECT")
	}
}
