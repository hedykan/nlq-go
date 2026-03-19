# TDD (测试驱动开发) 强制规则

## 📋 规则概述

**本项目强制要求使用TDD（Test-Driven Development）开发方式。**

所有新功能的开发必须遵循 **Red-Green-Refactor** 循环：

1. 🔴 **Red** - 先写一个失败的测试
2. 🟢 **Green** - 编写最简单的代码让测试通过
3. 🔵 **Refactor** - 重构代码，保持测试通过

---

## 🎯 核心原则

### 1. 测试先行原则

**禁止**：先写功能代码，再补测试代码。

**必须**：先写测试，再写功能代码。

**工作流程：**
```
1. 理解需求
2. 编写失败的测试 (Red)
3. 编写最少代码让测试通过 (Green)
4. 重构优化代码 (Refactor)
5. 重复以上步骤
```

### 2. 测试覆盖率要求

**最低覆盖率标准：**
- **整体覆盖率**：≥ 70%
- **核心业务逻辑**：≥ 90%
- **API端点**：≥ 80%
- **数据处理**：≥ 85%

**检查命令：**
```bash
# 生成覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 查看覆盖率
go tool cover -func=coverage.out
```

### 3. 测试文件组织

**测试文件命名：**
```
源文件: internal/server/handlers.go
测试文件: internal/server/handlers_test.go
```

**包结构：**
```go
package server  // 与源文件在同一个包

import (
    "testing"
    "github.com/stretchr/testify/assert"  // 使用断言库
)
```

---

## 📝 编写规范

### 1. 测试函数命名

**格式：** `Test<FunctionName>_<Scenario>`

**示例：**
```go
// ✅ 好的命名
func TestGLMClient_GenerateSQL_Success(t *testing.T)
func TestGLMClient_GenerateSQL_EmptyQuestion(t *testing.T)
func TestQueryHandler_Handle_InvalidInput(t *testing.T)
func TestQueryHandler_Handle_DatabaseError(t *testing.T)

// ❌ 不好的命名
func TestClient1(t *testing.T)
func TestQuery(t *testing.T)
func TestError(t *testing.T)
```

### 2. 测试结构（表驱动测试）

**优先使用表驱动测试：**

```go
func TestGenerateSQL(t *testing.T) {
    tests := []struct {
        name      string
        question  string
        expected  string
        wantError bool
    }{
        {
            name:      "简单查询",
            question:  "查询用户总数",
            expected:  "SELECT COUNT(*) FROM users",
            wantError: false,
        },
        {
            name:      "空问题",
            question:  "",
            expected:  "",
            wantError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sql, err := GenerateSQL(tt.question)
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, sql)
            }
        })
    }
}
```

### 3. 断言使用规范

**使用 `testify/assert` 库：**

```go
import "github.com/stretchr/testify/assert"

// ✅ 推荐用法
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.NotNil(t, obj)
assert.True(t, condition)
assert.Contains(t, str, substring)
assert.Len(t, slice, expectedLength)

// ❌ 避免使用
if actual != expected {
    t.Errorf("expected %v, got %v", expected, actual)
}
```

### 4. Mock和测试辅助函数

**使用 `mock` 前缀命名测试辅助函数：**

```go
// 创建Mock测试客户端
func createMockClient(t *testing.T) *GLMClient {
    return NewGLMClient("test-key", "http://test", "glm-4-plus")
}

// 创建Mock数据库
func setupMockDB(t *testing.T) *gorm.DB {
    // 返回测试数据库
}

// 清理函数
func cleanup(t *testing.T) {
    // 清理测试数据
}
```

---

## 🚫 禁止行为

### 1. 禁止先写代码后补测试

```go
// ❌ 错误做法
// Step 1: 先写功能代码
func GenerateSQL(question string) string {
    // ... 实现代码 ...
}

// Step 2: 再补测试
func TestGenerateSQL(t *testing.T) {
    // ... 测试代码 ...
}

// ✅ 正确做法
// Step 1: 先写测试
func TestGenerateSQL(t *testing.T) {
    result := GenerateSQL("查询用户")
    assert.Contains(t, result, "SELECT")
}

// Step 2: 再写功能代码
func GenerateSQL(question string) string {
    return "SELECT * FROM users WHERE ..."
}
```

### 2. 禁止提交没有测试的代码

**代码审查检查项：**
- [ ] 新功能是否有对应的测试？
- [ ] 测试覆盖率是否达标？
- [ ] 所有测试是否通过？

**Pre-commit Hook（推荐）：**
```bash
# .git/hooks/pre-commit
#!/bin/bash
go test ./...
if [ $? -ne 0 ]; then
    echo "❌ 测试失败，禁止提交"
    exit 1
fi
go test ./... -coverprofile=coverage.out
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$coverage < 70" | bc -l) )); then
    echo "❌ 覆盖率 ${coverage}% 低于70%，禁止提交"
    exit 1
fi
```

### 3. 禁止在测试中使用真实的外部依赖

**使用Mock替代：**
```go
// ❌ 错误：直接调用真实API
func TestGenerateSQL(t *testing.T) {
    sql, _ := GenerateSQL("查询")
    // 如果API Key无效，测试会失败
}

// ✅ 正确：使用Mock或测试工具
func TestGenerateSQL(t *testing.T) {
    client := createMockClient(t)
    sql, err := client.GenerateSQL("查询")
    // 可预测的测试结果
}
```

---

## 📊 测试分类

### 1. 单元测试

**目标：** 测试单个函数或方法

**示例：**
```go
func TestParseSQLFromResponse_ValidSQL(t *testing.T) {
    response := "SELECT * FROM users"
    sql, err := ParseSQLFromResponse(response)
    assert.NoError(t, err)
    assert.Equal(t, "SELECT * FROM users", sql)
}
```

**运行：**
```bash
go test ./internal/llm -v
```

### 2. 集成测试

**目标：** 测试多个组件协作

**示例：**
```go
func TestQueryAPI_Integration(t *testing.T) {
    // 启动测试服务器
    server := startTestServer(t)
    defer server.Close()

    // 发送HTTP请求
    resp := testRequest(server, "POST", "/api/v1/query", payload)

    // 验证响应
    assert.Equal(t, 200, resp.StatusCode)
}
```

**运行：**
```bash
go test ./test/integration -v
```

### 3. 端到端测试

**目标：** 测试完整的用户场景

**示例：**
```go
func TestE2E_UserQueryFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过E2E测试")
    }

    // 1. 用户打开页面
    // 2. 输入问题
    // 3. 提交查询
    // 4. 验证结果
}
```

**运行：**
```bash
go test ./test/e2e -v
```

---

## 🎯 项目特定规则

### NLQ项目特殊要求

#### 1. SQL生成测试

**必须包含：**
- [ ] 正常SQL生成
- [ ] SQL注入防护
- [ ] 空问题处理
- [ ] 特殊字符处理

**示例：**
```go
func TestGenerateSQL_SQLInjection(t *testing.T) {
    tests := []struct {
        name     string
        question string
        safe     bool
    }{
        {
            name:     "正常查询",
            question: "查询用户总数",
            safe:     true,
        },
        {
            name:     "包含DROP TABLE",
            question: "'; DROP TABLE users; --",
            safe:     false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sql, err := GenerateSQL(tt.question)
            if tt.safe {
                assert.NoError(t, err)
                assert.True(t, ValidateSQLQuery(sql))
            } else {
                // 应该被防火墙拦截
            }
        })
    }
}
```

#### 2. LLM客户端测试

**必须使用Mock API：**
```go
func TestGLMClient_GenerateSQL_MockSuccess(t *testing.T) {
    // 使用Mock服务器或测试工具
    client := CreateMockTestClient()

    sql, err := client.GenerateSQL("查询用户")
    assert.NoError(t, err)
    assert.NotEmpty(t, sql)
}
```

#### 3. API端点测试

**必须测试：**
- [ ] 正常请求
- [ ] 参数验证
- [ ] 错误处理
- [ ] CORS头设置
- [ ] 响应格式

**示例：**
```go
func TestHandleQuery_ValidRequest(t *testing.T) {
    handler := NewQueryHandler(mockDB)

    body := `{"question": "查询用户总数"}`
    req := httptest.NewRequest("POST", "/api/v1/query", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    handler.HandleQuery(w, req)

    assert.Equal(t, 200, w.Code)
    assert.Contains(t, w.Body.String(), "success")
}
```

---

## 🔧 工具和辅助

### 1. 测试辅助函数位置

**统一放在：** `internal/<module>/testutil.go`

**示例：**
```
internal/llm/testutil.go
internal/server/testutil.go
internal/handler/testutil.go
```

**提供函数：**
- `CreateMockClient(t *testing.T) *GLMClient`
- `CreateTestDB(t *testing.T) *gorm.DB`
- `CreateTestConfig(t *testing.T) *config.Config`
- `CleanupTestData(t *testing.T)`

### 2. 常用测试命令

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/llm -v

# 运行特定测试
go test ./internal/llm -run TestGenerateSQL

# 运行测试并生成覆盖率
go test ./... -coverprofile=coverage.out

# 查看HTML覆盖率报告
go tool cover -html=coverage.out

# 运行基准测试
go test -bench=. -benchmem

# 跳过慢速测试
go test ./... -short

# 详细输出
go test ./... -v
```

### 3. Makefile集成

**添加到 Makefile：**
```makefile
# 测试相关
.PHONY: test test-cover test-integration test-e2e

test:
	go test ./... -v

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out
	open coverage.html

test-integration:
	go test ./test/integration/... -v

test-e2e:
	go test ./test/e2e/... -v

# 检查测试覆盖率是否达标
test-coverage-check:
	go test ./... -coverprofile=coverage.out
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$${coverage}" -lt 70 ]; then \
		echo "❌ 覆盖率 $${coverage}% 低于70%"; \
		exit 1; \
	fi
```

---

## 📋 代码审查清单

### PR/MR 提交前检查

**必须确认：**

- [ ] 所有测试通过 (`go test ./...`)
- [ ] 测试覆盖率达标 (`go test -coverprofile=coverage.out`)
- [ ] 新功能有对应测试
- [ ] Mock使用正确
- [ ] 没有skip的测试（除非有充分理由）
- [ ] 测试命名清晰
- [ ] 断言使用正确

**审查时会被拒绝的情况：**

1. ❌ 新功能没有测试
2. ❌ 测试覆盖率低于70%
3. ❌ 测试全部skip
4. ❌ 使用真实的外部依赖（API密钥、数据库等）
5. ❌ 测试命名不规范
6. ❌ 硬编码测试数据

---

## 🎓 学习资源

### 推荐阅读

1. **《测试驱动开发》** by Kent Beck
2. **《Go语言圣经》- Testing章节**
3. **Testify Go** 文档

### 实用链接

- Go Testing: https://golang.org/pkg/testing/
- Testify: https://github.com/stretchr/testify
- Table Driven Tests: https://dave.cheney.net/2019/05/27/table-driven-tests-in-go/

---

## 🚀 快速开始

### 开发新功能的TDD流程

```bash
# 1. 创建测试文件
touch internal/server/myfeature_test.go

# 2. 编写失败的测试（Red）
# 在myfeature_test.go中添加测试函数

# 3. 运行测试，确认失败（Red）
go test ./internal/server -run TestMyFeature -v
# 预期：测试失败 ❌

# 4. 编写最简单的代码让测试通过（Green）
# 在myfeature.go中添加功能代码

# 5. 运行测试，确认通过（Green）
go test ./internal/server -run TestMyFeature -v
# 预期：测试通过 ✅

# 6. 重构代码，保持测试通过（Refactor）
# 优化代码结构，同时保持测试通过

# 7. 重复2-6，直到功能完成
```

---

## ⚖️ 例外情况

**可以不遵循TDD的情况：**

1. **UI/前端代码**：可以使用手动测试
2. **配置文件**：可以不写测试
3. **文档代码**：可以不写测试
4. **示例代码**：可以不写测试
5. **紧急修复**：可以先修复，事后补测试

**但必须记录：**
```go
// TODO: 需要添加测试
// Reason: 紧急修复生产环境bug
// Date: 2026-03-19
// Author: <your-name>
```

---

## 📝 总结

### TDD的三个步骤

1. 🔴 **Red** - 先写一个会失败的测试
2. 🟢 **Green** - 写最简单的代码让测试通过
3. 🔵 **Refactor** - 重构代码，保持测试通过

### 记住

- **测试先行**，测试驱动开发
- **小步快跑**，频繁重构
- **覆盖率达标**，质量保证
- **持续集成**，自动化检查

---

**强制执行！** 🚨

所有不符合此规则的代码将被拒绝合并！

---

*最后更新：2026-03-19*
*维护者：ChannelWill开发团队*
