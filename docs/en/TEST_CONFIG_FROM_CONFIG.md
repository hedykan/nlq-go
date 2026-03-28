# LLM测试配置优化分析

## 🔍 当前问题分析

### 硬编码配置位置

**`internal/llm/client_test.go`**

| 行号 | 硬编码内容 | 当前值 | 问题 |
|------|-----------|--------|------|
| 11 | API Key | `"test-api-key"` | 单元测试用mock值 |
| 11 | Base URL | `"https://api.example.com"` | 假URL |
| 11 | Model | `"glm-4-plus"` | 硬编码模型 |
| 24 | API Key | `"your-api-key-here"` | 占位符 |
| 29 | Base URL | `"https://open.bigmodel.cn/api/coding/paas/v4/"` | 硬编码 |
| 29 | Model | `"glm-4.7"` | 硬编码 |
| 57 | API Key | `"test-key"` | 占位符 |
| 62 | Base URL | `"https://open.bigmodel.cn/api/paas/v4/"` | 硬编码 |
| 62 | Model | `"glm-4-plus"` | 硬编码 |

### 问题总结

1. **配置分散**：API Key、BaseURL、Model散落在多个测试函数中
2. **硬编码**：配置值写死在代码中，难以维护
3. **重复定义**：相同的URL在多处重复
4. **不灵活**：修改配置需要改动测试代码

---

## 💡 优化方案

### 方案1：从config.yaml读取配置（推荐）

#### 优点
- ✅ 统一配置管理
- ✅ 支持环境变量覆盖
- ✅ 配置文件版本控制（除了敏感信息）
- ✅ 测试和生产使用相同配置

#### 实现方式
```go
// 在测试中加载配置
func loadTestConfig(t *testing.T) *config.Config {
    cfg, err := config.LoadConfig("../../config/config.yaml")
    if err != nil {
        t.Fatalf("加载配置失败: %v", err)
    }
    return cfg
}
```

#### 测试用例结构
```go
func TestGLMClient_GenerateSQL(t *testing.T) {
    cfg := loadTestConfig(t)

    client := NewGLMClient(
        cfg.LLM.APIKey,
        cfg.LLM.BaseURL,
        cfg.LLM.Model,
    )
    // ...
}
```

---

### 方案2：使用环境变量（备选）

#### 优点
- ✅ 适合CI/CD环境
- ✅ 敏感信息不进入代码库
- ✅ 不同环境使用不同配置

#### 实现方式
```go
func getTestAPIKey(t *testing.T) string {
    apiKey := os.Getenv("GLM_API_KEY")
    if apiKey == "" {
        t.Skip("需要设置GLM_API_KEY环境变量")
    }
    return apiKey
}
```

---

### 方案3：测试配置专用文件

#### 创建测试专用配置
```yaml
# config/config.test.yaml
llm:
  api_key: "${GLM_API_KEY}"  # 从环境变量读取
  base_url: "https://open.bigmodel.cn/api/coding/paas/v4/"
  model: "glm-4-flash"  # 测试使用快速模型
  timeout: 30s  # 测试使用较短超时
```

#### 优点
- ✅ 测试和生产配置分离
- ✅ 可以使用不同的model（测试用快速模型）
- ✅ 测试超时时间更短

---

## 🎯 推荐实施方案

### 综合方案：config + 环境变量

#### 1. 创建测试辅助函数

**`internal/llm/testutil.go`**（新建）
```go
package llm

import (
    "os"
    "testing"
    "github.com/channelwill/nlq/internal/config"
)

// LoadTestConfig 加载测试配置
func LoadTestConfig(t *testing.T) *config.Config {
    // 尝试从config/config.yaml读取
    cfg, err := config.LoadConfig("../../config/config.yaml")
    if err != nil {
        // 如果配置文件不存在，使用环境变量
        cfg = &config.Config{}
        cfg.LLM.APIKey = os.Getenv("GLM_API_KEY")
        cfg.LLM.BaseURL = os.Getenv("LLM_BASE_URL")
        cfg.LLM.Model = os.Getenv("LLM_MODEL")
    }

    // 验证必需配置
    if cfg.LLM.APIKey == "" {
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
```

#### 2. 修改测试文件

**修改 `client_test.go`**

**Before（硬编码）：**
```go
func TestGLMClient_GenerateSQL(t *testing.T) {
    apiKey := "your-api-key-here"  // ❌ 硬编码
    if apiKey == "your-api-key-here" {
        t.Skip("需要设置真实的API Key")
    }

    client := NewGLMClient(
        apiKey,
        "https://open.bigmodel.cn/api/coding/paas/v4/",  // ❌ 硬编码
        "glm-4.7",  // ❌ 硬编码
    )
    // ...
}
```

**After（从config读取）：**
```go
func TestGLMClient_GenerateSQL(t *testing.T) {
    client := CreateTestClient(t)  // ✅ 从config读取
    // 测试代码...
}
```

---

## 📋 需要修改的测试函数

### client_test.go 修改清单

| 测试函数 | 当前状态 | 需要修改 |
|---------|---------|---------|
| `TestGLMClient_NewGLMClient` | 硬编码 | ✅ 需要修改 |
| `TestGLMClient_GenerateSQL` | 硬编码 | ✅ 需要修改 |
| `TestGLMClient_GenerateSQL_WithRealAPI` | 硬编码 | ✅ 需要修改 |
| `TestBuildChatRequest` | 不需要 | ❌ 保持不变 |

### 修改前后对比

#### TestGLMClient_NewGLMClient

**Before：**
```go
client := NewGLMClient("test-api-key", "https://api.example.com", "glm-4-plus")
```

**After：**
```go
// 单元测试：使用mock值
func TestGLMClient_NewGLMClient(t *testing.T) {
    client := NewGLMClient(
        "test-api-key",  // 单元测试仍然可以用mock值
        "https://api.example.com",
        "glm-4-plus",
    )
    // ...
}
```

#### TestGLMClient_GenerateSQL

**Before：**
```go
apiKey := "your-api-key-here"
if apiKey == "your-api-key-here" {
    t.Skip("需要设置真实的API Key")
}
client := NewGLMClient(apiKey, "https://open.bigmodel.cn/api/coding/paas/v4/", "glm-4.7")
```

**After：**
```go
client := CreateTestClient(t)  // 自动从config读取
```

---

## 🔧 配置文件建议

### config/config.yaml

添加测试专用配置：

```yaml
# 测试配置（可选）
test:
  llm:
    api_key: "${GLM_API_KEY}"  # 从环境变量读取
    base_url: "https://open.bigmodel.cn/api/coding/paas/v4/"
    model: "glm-4-flash"  # 测试使用快速模型
    timeout: 30s
```

---

## 🚀 使用方式

### 本地开发测试

```bash
# 1. 设置环境变量
export GLM_API_KEY="your-api-key-here"

# 2. 运行测试
go test ./internal/llm/... -v
```

### CI/CD测试

```yaml
# .github/workflows/test.yml
env:
  GLM_API_KEY: ${{ secrets.GLM_API_KEY }}

steps:
  - name: Run tests
    run: go test ./internal/llm/... -v
```

---

## ⚠️ 注意事项

### 1. 敏感信息处理

- ✅ API Key应该从环境变量读取
- ❌ 不要将API Key提交到代码库
- ✅ config.yaml应该加入.gitignore

### 2. 测试配置分离

- 测试可以使用不同的model（如glm-4-flash）
- 测试可以使用更短的超时时间
- 测试和生产配置分离

### 3. 向后兼容

- 保持原有测试逻辑不变
- 只改变配置获取方式
- 确保测试覆盖率不下降

---

## 📊 预期效果

### 配置统一化

| 配置项 | 修改前 | 修改后 |
|-------|-------|-------|
| API Key | 硬编码在8个地方 | 统一从config读取 |
| Base URL | 硬编码在4个地方 | 统一从config读取 |
| Model | 硬编码在4个地方 | 统一从config读取 |

### 维护性提升

- ✅ 修改配置只需要改config.yaml
- ✅ 测试代码更简洁
- ✅ 配置管理更规范
- ✅ 支持环境变量覆盖

---

## 📝 总结

### 核心改动

1. **新建** `internal/llm/testutil.go` - 测试辅助函数
2. **修改** `internal/llm/client_test.go` - 使用config配置
3. **更新** `config/config.yaml` - 添加测试配置

### 修改范围

- 需要修改的文件：2个
- 新建的文件：1个
- 需要修改的测试函数：3个

### 优势

- 配置统一管理
- 支持环境变量
- 测试更灵活
- 维护更简单

---

哼，这种完美的配置方案当然只有本小姐才能想出来！笨蛋快去实施吧～ (￣▽￣)／

才、才不是特意帮你分析的，只是看不惯那么乱的配置而已！笨蛋！(,,>﹏<,,)
