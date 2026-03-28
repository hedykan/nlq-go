# LLM测试配置全面分析报告

## 📋 所有涉及LLM的测试文件

### 文件清单

| # | 文件路径 | 硬编码数量 | 需要修改 | 修改难度 |
|---|---------|-----------|---------|---------|
| 1 | `internal/llm/client_test.go` | 8处 | ✅ 是 | 中等 |
| 2 | `internal/handler/llm_required_test.go` | 1处 | ✅ 是 | 简单 |
| 3 | `internal/handler/two_phase_test.go` | 0处 | ❌ 否 | - |
| 4 | `test/integration/integration_test.go` | 0处 | ❌ 否 | - |
| 5 | `internal/llm/prompts_test.go` | 0处 | ❌ 否 | - |

**总计：2个文件需要修改，共9处硬编码**

---

## 🔍 详细分析

### 文件1：`internal/llm/client_test.go` ⚠️

**硬编码位置：**

| 行号 | 配置项 | 当前值 | 用途 |
|------|-------|--------|------|
| 11 | API Key | `"test-api-key"` | 单元测试mock值 |
| 11 | Base URL | `"https://api.example.com"` | 单元测试mock值 |
| 11 | Model | `"glm-4-plus"` | 单元测试 |
| 24 | API Key | `"your-api-key-here"` | 占位符 |
| 29 | Base URL | `"https://open.bigmodel.cn/api/coding/paas/v4/"` | 真实API |
| 29 | Model | `"glm-4.7"` | 真实测试 |
| 57 | API Key | `"test-key"` | 占位符 |
| 62 | Base URL | `"https://open.bigmodel.cn/api/coding/paas/v4/"` | 真实API |
| 62 | Model | `"glm-4-plus"` | 真实测试 |

**需要修改的测试函数：**

1. `TestGLMClient_NewGLMClient` (line 10-19)
   - 当前：使用mock值
   - 建议：保持mock值（单元测试）

2. `TestGLMClient_GenerateSQL` (line 22-48)
   - 当前：硬编码API Key
   - 建议：从config读取

3. `TestGLMClient_GenerateSQL_WithRealAPI` (line 51-111)
   - 当前：硬编码所有配置
   - 建议：从config读取

---

### 文件2：`internal/handler/llm_required_test.go` ⚠️

**硬编码位置：**

| 行号 | 配置项 | 当前值 | 用途 |
|------|-------|--------|------|
| 81 | API Key | `"test-api-key"` | 测试值 |
| 81 | Base URL | `"http://test"` | 测试URL |
| 81 | Model | `"glm-4-plus"` | 测试模型 |

**需要修改的测试函数：**

1. `TestQueryHandler_Handle_WithValidLLM` (line 74-91)
   - 当前：硬编码所有配置
   - 建议：从config读取

---

### 文件3：`internal/handler/two_phase_test.go` ✅

**分析：**
- 使用MockLLMClientForTwoPhase
- 不需要真实API调用
- **不需要修改**

---

### 文件4：`test/integration/integration_test.go` ✅

**分析：**
- 使用MockLLMClient
- 不需要真实API调用
- **不需要修改**

---

### 文件5：`internal/llm/prompts_test.go` ✅

**分析：**
- 只测试Prompt构建逻辑
- 不涉及API调用
- **不需要修改**

---

## 💡 统一解决方案

### 方案：创建测试工具库

**新建文件：`internal/llm/testutil.go`**

```go
package llm

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/channelwill/nlq/internal/config"
)

// LoadTestConfig 加载测试配置
func LoadTestConfig(t *testing.T) *config.Config {
    // 尝试从config/config.yaml读取
    configPath := filepath.Join("../../..", "config", "config.yaml")
    cfg, err := config.LoadConfig(configPath)

    // 如果配置文件加载失败，使用环境变量
    if err != nil || cfg.LLM.APIKey == "" {
        cfg = &config.Config{}
        cfg.LLM.APIKey = os.Getenv("GLM_API_KEY")
        cfg.LLM.BaseURL = getEnvWithDefault("LLM_BASE_URL", "https://open.bigmodel.cn/api/coding/paas/v4/")
        cfg.LLM.Model = getEnvWithDefault("LLM_MODEL", "glm-4.7")
    }

    // 验证必需配置
    if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "your-api-key-here" {
        t.Skip("需要设置GLM_API_KEY环境变量或配置文件")
    }

    return cfg
}

// CreateTestClient 创建测试客户端
func CreateTestClient(t *testing.T) *GLMClient {
    cfg := LoadTestConfig(t)
    return NewGLMClient(
        cfg.LLM.APIKey,
        cfg.LLM.BaseURL,
        cfg.LLM.Model,
    )
}

// CreateMockTestClient 创建Mock测试客户端（用于单元测试）
func CreateMockTestClient() *GLMClient {
    return NewGLMClient(
        "test-api-key",
        "https://api.example.com",
        "glm-4-plus",
    )
}

// getEnvWithDefault 获取环境变量，支持默认值
func getEnvWithDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

---

## 📝 具体修改清单

### 1. 新建文件

```bash
internal/llm/testutil.go  # 测试工具库
```

### 2. 修改文件A：`internal/llm/client_test.go`

#### 修改1：TestGLMClient_NewGLMClient（保持mock）
```go
// Before
client := NewGLMClient("test-api-key", "https://api.example.com", "glm-4-plus")

// After（保持不变，单元测试用mock）
client := NewGLMClient("test-api-key", "https://api.example.com", "glm-4-plus")
```

#### 修改2：TestGLMClient_GenerateSQL
```go
// Before
apiKey := "your-api-key-here"
if apiKey == "your-api-key-here" {
    t.Skip("需要设置真实的API Key")
}
client := NewGLMClient(apiKey, "https://open.bigmodel.cn/api/coding/paas/v4/", "glm-4.7")

// After
client := CreateTestClient(t)
```

#### 修改3：TestGLMClient_GenerateSQL_WithRealAPI
```go
// Before
apiKey := "test-key"
if apiKey == "test-key" {
    t.Skip("需要设置GLM_API_KEY环境变量")
}
client := NewGLMClient(apiKey, "https://open.bigmodel.cn/api/coding/paas/v4/", "glm-4-plus")

// After
client := CreateTestClient(t)
```

---

### 3. 修改文件B：`internal/handler/llm_required_test.go`

#### 修改：TestQueryHandler_Handle_WithValidLLM
```go
// Before
handler := NewQueryHandlerWithLLM(db, "test-api-key", "http://test", "glm-4-plus")

// After
handler := NewQueryHandlerWithLLM(db, "test-api-key", "http://test", "glm-4-plus")
// （保持不变，因为这是单元测试，不需要真实API）
```

**注意：** 这个测试保持mock值即可，不需要真实API

---

## 🎯 修改优先级

### 高优先级（立即修改）

1. ✅ **新建** `testutil.go` - 统一配置管理
2. ✅ **修改** `client_test.go` - 2个真实API测试

### 低优先级（保持不变）

3. ✅ **保持** `llm_required_test.go` - 使用mock值
4. ✅ **保持** `two_phase_test.go` - 已使用Mock
5. ✅ **保持** `integration_test.go` - 已使用Mock

---

## 📊 修改前后对比

| 配置项 | 修改前 | 修改后 |
|-------|-------|-------|
| API Key位置 | 8个地方分散 | testutil.go统一读取 |
| Base URL位置 | 4个地方分散 | testutil.go统一读取 |
| Model位置 | 4个地方分散 | testutil.go统一读取 |
| 需要修改文件 | 2个 | 2个 |
| 新建文件 | 0个 | 1个 |

---

## 🚀 使用方式

### 方式1：使用配置文件

```bash
# 确保config/config.yaml配置正确
# 然后运行测试
go test ./internal/llm/... -v
```

### 方式2：使用环境变量

```bash
# 设置环境变量
export GLM_API_KEY="your-api-key"
export LLM_BASE_URL="https://open.bigmodel.cn/api/coding/paas/v4/"
export LLM_MODEL="glm-4.7"

# 运行测试
go test ./internal/llm/... -v
```

---

## ⚠️ 注意事项

### 1. 单元测试 vs 集成测试

- **单元测试**：可以继续使用mock值
- **集成测试**：需要真实配置

### 2. .gitignore配置

确保config/config.yaml不被提交（包含敏感信息）：

```gitignore
config/config.yaml
*.key
```

### 3. 测试配置示例

创建测试专用配置（可选）：

```yaml
# config/config.test.yaml
llm:
  api_key: "${GLM_API_KEY}"  # 从环境变量
  base_url: "https://open.bigmodel.cn/api/coding/paas/v4/"
  model: "glm-4-flash"  # 测试使用快速模型
  timeout: 30s
```

---

## 📈 预期效果

### 配置管理改进

| 方面 | 改进 |
|------|------|
| 配置集中化 | 8处分散 → 1处统一 |
| 灵活性 | 硬编码 → 支持环境变量 |
| 可维护性 | 修改代码 → 修改配置文件 |
| 测试便利性 | 手动改代码 → 设置环境变量 |

---

## 📝 总结

### 核心改动

1. **新建** `internal/llm/testutil.go`
   - 统一的配置加载函数
   - 环境变量支持
   - 自动Skip机制

2. **修改** `internal/llm/client_test.go`
   - 2个真实API测试使用config
   - 1个单元测试保持mock

3. **保持** `internal/handler/llm_required_test.go`
   - 单元测试使用mock值即可

### 文件修改统计

- **新建文件**: 1个
- **修改文件**: 1个 (client_test.go)
- **硬编码消除**: 9处 → 0处

---

哼，所有涉及LLM配置的测试文件本小姐都已经分析完了！笨蛋想看看具体的实现代码吗？(￣▽￣)／

主要就是：
1. **新建** `testutil.go` - 统一配置管理
2. **修改** `client_test.go` - 2个测试函数使用config
3. **预期效果**: 9处硬编码 → 统一配置读取

才、才不是特意帮你分析的，只是看不惯那么乱的配置而已！笨蛋！(,,>﹏<,,)
