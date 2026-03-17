package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Document 表示一个知识库文档
type Document struct {
	Title     string // 文档标题
	Content   string // 文档内容
	FilePath  string // 文件路径
	Source    string // 来源文件名
}

// IsValid 验证文档是否有效
func (d *Document) IsValid() bool {
	return d.Title != "" && d.Content != ""
}

// Loader 知识库加载器
type Loader struct {
	extensions []string // 支持的文件扩展名
	recursive  bool     // 是否递归加载子文件夹
}

// NewLoader 创建新的知识库加载器
func NewLoader() *Loader {
	return &Loader{
		extensions: []string{".md"},
		recursive:  true,
	}
}

// LoadFromDirectory 从文件夹加载知识库文档
func (l *Loader) LoadFromDirectory(dirPath string) ([]Document, error) {
	// 检查文件夹是否存在
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("无法访问文件夹 %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s 不是文件夹", dirPath)
	}

	var documents []Document

	// 遍历文件夹
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果不是递归模式，跳过子文件夹
		if !l.recursive && info.IsDir() && path != dirPath {
			return filepath.SkipDir
		}

		// 只处理文件
		if info.IsDir() {
			return nil
		}

		// 检查文件扩展名
		if !l.isSupportedFile(path) {
			return nil
		}

		// 读取文件内容
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取文件 %s 失败: %w", path, err)
		}

		// 解析文档
		doc := l.parseDocument(string(content), path)
		if doc.IsValid() {
			documents = append(documents, doc)
		}
		return nil
	}

	if err := filepath.Walk(dirPath, walkFn); err != nil {
		return nil, fmt.Errorf("遍历文件夹失败: %w", err)
	}

	return documents, nil
}

// isSupportedFile 检查文件是否支持
func (l *Loader) isSupportedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, supportedExt := range l.extensions {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// parseDocument 解析文档内容
func (l *Loader) parseDocument(content, filePath string) Document {
	// 提取标题（假设第一行是Markdown标题）
	title := filepath.Base(filePath)
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "#") {
		// 移除 # 符号和空格
		title = strings.TrimSpace(strings.TrimPrefix(lines[0], "#"))
		title = strings.TrimSpace(title)
	}

	return Document{
		Title:    title,
		Content:  content,
		FilePath: filePath,
		Source:   filepath.Base(filePath),
	}
}

// SetExtensions 设置支持的文件扩展名
func (l *Loader) SetExtensions(extensions []string) {
	l.extensions = extensions
}

// SetRecursive 设置是否递归加载
func (l *Loader) SetRecursive(recursive bool) {
	l.recursive = recursive
}
