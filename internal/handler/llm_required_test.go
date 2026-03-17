package handler

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

// TestQueryHandler_Handle_RequiresLLM 测试查询处理器强制要求LLM
func TestQueryHandler_Handle_RequiresLLM(t *testing.T) {
	// 创建内存数据库
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 创建不带LLM的处理器（应该失败）
	queryHandler := NewQueryHandler(db)

	// 尝试执行查询
	ctx := context.Background()
	_, err = queryHandler.Handle(ctx, "测试问题")

	// 应该返回错误，要求配置API Key
	if err == nil {
		t.Error("期望返回错误，要求配置API Key")
	}

	// 检查错误消息包含API Key相关内容
	expectedError := "API Key"
	if err != nil && !containsError(err.Error(), expectedError) {
		t.Errorf("期望错误包含'%s'，实际: %v", expectedError, err)
	}
}

// TestNewQueryHandlerWithLLM_EmptyAPIKey 测试空API Key时的行为
func TestNewQueryHandlerWithLLM_EmptyAPIKey(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 空API Key应该创建一个不可用的处理器
	handler := NewQueryHandlerWithLLM(db, "", "")
	if handler == nil {
		t.Fatal("期望返回处理器实例，即使API Key为空")
	}

	// 验证处理器状态：空API Key时不应该启用真正的LLM
	if handler.useRealLLM {
		t.Error("空API Key时不应该设置useRealLLM为true")
	}

	// 验证LLM客户端为nil
	if handler.llmClient != nil {
		t.Error("空API Key时LLM客户端应该为nil")
	}

	// 验证尝试使用时会失败
	ctx := context.Background()
	_, err = handler.Handle(ctx, "测试问题")
	if err == nil {
		t.Error("空API Key时Handle应该返回错误")
	}

	expectedErrorMsg := "API Key"
	if !containsError(err.Error(), expectedErrorMsg) {
		t.Errorf("错误消息应该包含'%s'，实际: %v", expectedErrorMsg, err)
	}
}

// TestQueryHandler_Handle_WithValidLLM 测试有效LLM配置
func TestQueryHandler_Handle_WithValidLLM(t *testing.T) {
	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	// 使用Mock LLM客户端
	handler := NewQueryHandlerWithLLM(db, "test-api-key", "http://test")

	// 验证LLM被正确设置
	if !handler.useRealLLM {
		t.Error("有效API Key时应该设置useRealLLM为true")
	}

	if handler.llmClient == nil {
		t.Error("LLM客户端不应该为nil")
	}
}

// setupTestDB 创建测试数据库
func setupTestDB() (*gorm.DB, error) {
	// 返回nil表示使用现有数据库连接
	// 实际测试中应该使用mock数据库
	return nil, nil
}

// containsError 检查错误消息是否包含指定字符串
func containsError(errMsg, substr string) bool {
	return len(errMsg) > 0 && len(substr) > 0 &&
		   (errMsg == substr || len(errMsg) > len(substr) && findErrorSubstring(errMsg, substr))
}

func findErrorSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
