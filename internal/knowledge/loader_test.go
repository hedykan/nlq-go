package knowledge

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewLoader 测试创建加载器
func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("期望返回非nil的加载器实例")
	}
}

// TestLoader_LoadFromDirectory_Empty 测试加载空文件夹
func TestLoader_LoadFromDirectory_Empty(t *testing.T) {
	// 创建临时空文件夹
	tempDir := t.TempDir()

	loader := NewLoader()
	docs, err := loader.LoadFromDirectory(tempDir)

	if err != nil {
		t.Errorf("加载空文件夹不应该返回错误: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("空文件夹应该返回0个文档，实际返回: %d", len(docs))
	}
}

// TestLoader_LoadFromDirectory_NotExist 测试加载不存在的文件夹
func TestLoader_LoadFromDirectory_NotExist(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadFromDirectory("/nonexistent/directory")

	if err == nil {
		t.Error("加载不存在的文件夹应该返回错误")
	}
}

// TestLoader_LoadFromDirectory_WithMDFiles 测试加载包含MD文件的文件夹
func TestLoader_LoadFromDirectory_WithMDFiles(t *testing.T) {
	// 创建临时文件夹并添加测试文件
	tempDir := t.TempDir()

	// 创建测试MD文件
	testFile1 := filepath.Join(tempDir, "business_rules.md")
	content1 := "# 业务规则\n\nVIP用户享受20%折扣"
	if err := os.WriteFile(testFile1, []byte(content1), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	testFile2 := filepath.Join(tempDir, "field_explanations.md")
	content2 := "# 字段说明\n\nstatus: 1=active, 0=inactive"
	if err := os.WriteFile(testFile2, []byte(content2), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建非MD文件（应该被忽略）
	nonMDFile := filepath.Join(tempDir, "readme.txt")
	if err := os.WriteFile(nonMDFile, []byte("这是文本文件"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	loader := NewLoader()
	docs, err := loader.LoadFromDirectory(tempDir)

	if err != nil {
		t.Errorf("加载MD文件不应该返回错误: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("期望加载2个MD文档，实际加载: %d", len(docs))
	}

	// 验证文档内容
	foundVIP := false
	foundStatus := false
	for _, doc := range docs {
		if len(doc.Content) > 0 {
			if contains(doc.Content, "VIP用户") || contains(doc.Content, "20%折扣") {
				foundVIP = true
			}
			if contains(doc.Content, "status") || contains(doc.Content, "active") {
				foundStatus = true
			}
		}
	}

	if !foundVIP {
		t.Error("期望找到VIP用户的业务规则")
	}

	if !foundStatus {
		t.Error("期望找到status字段说明")
	}
}

// TestLoader_LoadFromDirectory_WithSubdirectories 测试递归加载子文件夹
func TestLoader_LoadFromDirectory_WithSubdirectories(t *testing.T) {
	// 创建临时文件夹结构
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "examples")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("创建子文件夹失败: %v", err)
	}

	// 在主文件夹创建文件
	mainFile := filepath.Join(tempDir, "main.md")
	if err := os.WriteFile(mainFile, []byte("# 主文档"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 在子文件夹创建文件
	subFile := filepath.Join(subDir, "example.md")
	if err := os.WriteFile(subFile, []byte("# 示例文档"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	loader := NewLoader()
	docs, err := loader.LoadFromDirectory(tempDir)

	if err != nil {
		t.Errorf("递归加载不应该返回错误: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("期望递归加载2个文档，实际加载: %d", len(docs))
	}
}

// TestDocument_IsValid 测试文档验证
func TestDocument_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		doc      Document
		expected bool
	}{
		{
			name:     "有效文档",
			doc:      Document{Title: "测试", Content: "内容"},
			expected: true,
		},
		{
			name:     "空标题",
			doc:      Document{Title: "", Content: "内容"},
			expected: false,
		},
		{
			name:     "空内容",
			doc:      Document{Title: "测试", Content: ""},
			expected: false,
		},
		{
			name:     "全空",
			doc:      Document{Title: "", Content: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.doc.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
