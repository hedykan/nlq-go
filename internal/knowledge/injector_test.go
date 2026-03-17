package knowledge

import (
	"strings"
	"testing"
)

// TestNewInjector 测试创建注入器
func TestNewInjector(t *testing.T) {
	injector := NewInjector()
	if injector == nil {
		t.Fatal("期望返回非nil的注入器实例")
	}
}

// TestInjector_Inject_EmptyDocuments 测试注入空文档列表
func TestInjector_Inject_EmptyDocuments(t *testing.T) {
	injector := NewInjector()
	basePrompt := "根据Schema生成SQL"

	result := injector.Inject(basePrompt, []Document{})

	if result != basePrompt {
		t.Errorf("空文档不应该修改原始Prompt\n期望: %s\n实际: %s", basePrompt, result)
	}
}

// TestInjector_Inject_SingleDocument 测试注入单个文档
func TestInjector_Inject_SingleDocument(t *testing.T) {
	injector := NewInjector()
	basePrompt := "根据Schema生成SQL"

	docs := []Document{
		{Title: "业务规则", Content: "VIP用户享受20%折扣"},
	}

	result := injector.Inject(basePrompt, docs)

	// 验证原始Prompt被保留
	if !strings.Contains(result, basePrompt) {
		t.Error("注入后的Prompt应该包含原始Prompt")
	}

	// 验证文档内容被注入
	if !strings.Contains(result, "VIP用户") {
		t.Error("注入后的Prompt应该包含文档内容")
	}

	if !strings.Contains(result, "20%折扣") {
		t.Error("注入后的Prompt应该包含完整的文档内容")
	}
}

// TestInjector_Inject_MultipleDocuments 测试注入多个文档
func TestInjector_Inject_MultipleDocuments(t *testing.T) {
	injector := NewInjector()
	basePrompt := "根据Schema生成SQL"

	docs := []Document{
		{Title: "业务规则", Content: "VIP用户享受20%折扣"},
		{Title: "字段说明", Content: "status字段: 1=active, 0=inactive"},
		{Title: "表关系", Content: "users表通过user_id关联orders表"},
	}

	result := injector.Inject(basePrompt, docs)

	// 验证所有文档都被注入
	expectedContent := []string{
		"VIP用户", "20%折扣",
		"status字段", "active", "inactive",
		"users表", "user_id", "orders表",
	}

	for _, content := range expectedContent {
		if !strings.Contains(result, content) {
			t.Errorf("注入后的Prompt应该包含: %s", content)
		}
	}
}

// TestInjector_Inject_LongContent 测试注入长内容
func TestInjector_Inject_LongContent(t *testing.T) {
	injector := NewInjector()
	basePrompt := "根据Schema生成SQL"

	// 创建长内容文档
	longContent := strings.Repeat("这是一个很长的业务规则说明。", 100)
	docs := []Document{
		{Title: "详细规则", Content: longContent},
	}

	result := injector.Inject(basePrompt, docs)

	// 验证长内容被正确处理（可能被截断）
	if !strings.Contains(result, basePrompt) {
		t.Error("即使文档很长，也应该保留原始Prompt")
	}

	// 验证文档标题被注入
	if !strings.Contains(result, "详细规则") {
		t.Error("应该包含文档标题")
	}
}

// TestInjector_Inject_EmptyBasePrompt 测试空基础Prompt
func TestInjector_Inject_EmptyBasePrompt(t *testing.T) {
	injector := NewInjector()
	basePrompt := ""

	docs := []Document{
		{Title: "业务规则", Content: "VIP用户享受20%折扣"},
	}

	result := injector.Inject(basePrompt, docs)

	// 验证即使基础Prompt为空，也能正常注入
	if !strings.Contains(result, "VIP用户") {
		t.Error("即使基础Prompt为空，也应该注入文档内容")
	}
}

// TestInjector_Inject_Formatting 测试注入格式化
func TestInjector_Inject_Formatting(t *testing.T) {
	injector := NewInjector()
	basePrompt := "根据Schema生成SQL"

	docs := []Document{
		{Title: "业务规则", Content: "VIP用户享受20%折扣"},
	}

	result := injector.Inject(basePrompt, docs)

	// 验证格式：应该有清晰的分隔
	lines := strings.Split(result, "\n")

	// 验证有知识库相关的标题或标记
	hasKnowledgeHeader := false
	for _, line := range lines {
		if strings.Contains(line, "知识库") || strings.Contains(line, "Knowledge") ||
		   strings.Contains(line, "业务规则") || strings.Contains(line, "=== ") {
			hasKnowledgeHeader = true
			break
		}
	}

	if !hasKnowledgeHeader {
		t.Error("注入的内容应该有清晰的标题或标记")
	}
}

// TestInjector_InjectWithMaxTokens 测试使用最大Token限制
func TestInjector_InjectWithMaxTokens(t *testing.T) {
	injector := NewInjector()
	basePrompt := "根据Schema生成SQL"

	// 创建多个文档
	docs := []Document{
		{Title: "规则1", Content: strings.Repeat("内容1 ", 100)},
		{Title: "规则2", Content: strings.Repeat("内容2 ", 100)},
		{Title: "规则3", Content: strings.Repeat("内容3 ", 100)},
	}

	// 使用较小的Token限制进行注入
	result := injector.InjectWithMaxTokens(basePrompt, docs, 500)

	// 验证结果被限制在合理范围内
	// 注意：这里只是验证函数能正常工作，实际的Token计算可能需要更复杂的实现
	if len(result) == 0 {
		t.Error("即使有Token限制，也应该返回有效的内容")
	}

	if !strings.Contains(result, basePrompt) {
		t.Error("即使有Token限制，也应该保留原始Prompt")
	}
}

// TestInjector_BuildKnowledgeContext 测试构建知识库上下文
func TestInjector_BuildKnowledgeContext(t *testing.T) {
	injector := NewInjector()

	docs := []Document{
		{Title: "业务规则", Content: "VIP用户享受20%折扣"},
		{Title: "字段说明", Content: "status: 1=active"},
	}

	context := injector.BuildKnowledgeContext(docs)

	// 验证上下文包含所有文档信息
	if !strings.Contains(context, "业务规则") {
		t.Error("上下文应该包含文档标题")
	}

	if !strings.Contains(context, "VIP用户") {
		t.Error("上下文应该包含文档内容")
	}

	if !strings.Contains(context, "字段说明") {
		t.Error("上下文应该包含所有文档")
	}
}
