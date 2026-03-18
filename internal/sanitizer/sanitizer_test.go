package sanitizer

import (
	"testing"
)

// TestSanitizer_SanitizeEmail 测试邮箱脱敏
func TestSanitizer_SanitizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通邮箱",
			input:    "contact@example.com",
			expected: "***@***.***",
		},
		{
			name:     "带数字邮箱",
			input:    "user123@163.com",
			expected: "***@***.***",
		},
		{
			name:     "带下划线邮箱",
			input:    "test_user@qq.com",
			expected: "***@***.***",
		},
		{
			name:     "无邮箱",
			input:    "这是普通文本",
			expected: "这是普通文本",
		},
		{
			name:     "多个邮箱",
			input:    "联系: a@test.com 或 b@example.com",
			expected: "联系: ***@***.*** 或 ***@***.***",
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeEmail(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeEmail() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizer_SanitizePhone 测试手机号脱敏
func TestSanitizer_SanitizePhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通手机号",
			input:    "13812345678",
			expected: "138****5678",
		},
		{
			name:     "15开头手机号",
			input:    "15987654321",
			expected: "159****4321",
		},
		{
			name:     "带区号手机号",
			input:    "+86 13812345678",
			expected: "+86 138****5678",
		},
		{
			name:     "无手机号",
			input:    "这是普通文本",
			expected: "这是普通文本",
		},
		{
			name:     "多个手机号",
			input:    "联系: 13812345678 或 15987654321",
			expected: "联系: 138****5678 或 159****4321",
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizePhone(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizePhone() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizer_SanitizeIDCard 测试身份证号脱敏
func TestSanitizer_SanitizeIDCard(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "18位身份证",
			input:    "110101199001011234",
			expected: "************1234",
		},
		{
			name:     "18位身份证带X",
			input:    "11010119900101123X",
			expected: "************123X",
		},
		{
			name:     "无身份证",
			input:    "这是普通文本",
			expected: "这是普通文本",
		},
		{
			name:     "多个身份证",
			input:    "证件1: 110101199001011234, 证件2: 310101199001011234",
			expected: "证件1: ************1234, 证件2: ************1234",
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeIDCard(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeIDCard() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizer_SanitizeIP 测试IP地址脱敏
func TestSanitizer_SanitizeIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通IP",
			input:    "192.168.1.1",
			expected: "***.***.1.1",
		},
		{
			name:     "公网IP",
			input:    "8.8.8.8",
			expected: "*.*.8.8",
		},
		{
			name:     "无IP",
			input:    "这是普通文本",
			expected: "这是普通文本",
		},
		{
			name:     "多个IP",
			input:    "服务器: 192.168.1.1, 10.0.0.1",
			expected: "服务器: ***.***.1.1, ***.*.0.1",
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeIP(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeIP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizer_SanitizeSQL 测试SQL语句脱敏
func TestSanitizer_SanitizeSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "简单SELECT",
			input:    "SELECT * FROM users WHERE email = 'test@example.com'",
			expected: "SELECT * FROM users WHERE email = '***@***.***'",
		},
		{
			name:     "带手机号SQL",
			input:    "SELECT * FROM users WHERE phone = '13812345678'",
			expected: "SELECT * FROM users WHERE phone = '138****5678'",
		},
		{
			name:     "带身份证SQL",
			input:    "SELECT * FROM users WHERE id_card = '110101199001011234'",
			expected: "SELECT * FROM users WHERE id_card = '************1234'",
		},
		{
			name:     "带IP的SQL",
			input:    "SELECT * FROM logs WHERE ip = '192.168.1.1'",
			expected: "SELECT * FROM logs WHERE ip = '***.***.1.1'",
		},
		{
			name:     "多种敏感信息",
			input:    "SELECT * FROM users WHERE email = 'a@b.com' AND phone = '13812345678'",
			expected: "SELECT * FROM users WHERE email = '***@***.***' AND phone = '138****5678'",
		},
		{
			name:     "无敏感信息",
			input:    "SELECT name, age FROM users WHERE age > 18",
			expected: "SELECT name, age FROM users WHERE age > 18",
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeSQL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeSQL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizer_SanitizeQuestion 测试问题文本脱敏
func TestSanitizer_SanitizeQuestion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "带邮箱问题",
			input:    "查询邮箱为test@example.com的用户信息",
			expected: "查询邮箱为***@***.***的用户信息",
		},
		{
			name:     "带手机号问题",
			input:    "查找手机号是13812345678的客户",
			expected: "查找手机号是138****5678的客户",
		},
		{
			name:     "带身份证问题",
			input:    "查询身份证号110101199001011234的用户",
			expected: "查询身份证号************1234的用户",
		},
		{
			name:     "无敏感信息问题",
			input:    "查询销售额大于10000的产品",
			expected: "查询销售额大于10000的产品",
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeQuestion(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeQuestion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizer_SanitizeAll 测试综合脱敏
func TestSanitizer_SanitizeAll(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // 检查结果中是否包含这些字符串
	}{
		{
			name:     "综合敏感信息",
			input:    "联系: test@example.com, 电话: 13812345678, 身份证: 110101199001011234, IP: 192.168.1.1",
			contains: []string{"***@***.***", "138****5678", "************1234", "***.***.1.1"},
		},
		{
			name:     "纯文本无敏感信息",
			input:    "查询所有销售额大于10000的产品",
			contains: []string{"查询所有销售额大于10000的产品"},
		},
	}

	s := NewSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeAll(tt.input)
			for _, expected := range tt.contains {
				if !contains(result, expected) {
					t.Errorf("SanitizeAll() result = %v, should contain %v", result, expected)
				}
			}
		})
	}
}

// contains 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
