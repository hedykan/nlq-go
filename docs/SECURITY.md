# 🔒 NLQ 安全性分析报告

## 概述

本文档详细说明NLQ（Natural Language Query）项目的安全机制，以及为什么该系统在设计上是安全的。本系统通过多层安全防护，确保LLM无法直接访问数据库业务数据，并严格防止不可逆的数据操作。

---

## 核心安全问题解答

### ❓ LLM会直接读取数据库数据吗？

**答案：不会！**

#### 数据流程说明

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│  用户问题    │ ──→ │   LLM服务    │ ──→ │  SQL防火墙   │ ──→ │  数据库执行   │
└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
                           ↑
                    只接收Schema元数据
                    (表名、列名、类型等)
                    不包含任何业务数据！
```

#### 详细分析

1. **Schema数据来源** (`internal/database/schema.go`)
   - 从MySQL的`information_schema`系统表获取元数据
   - 包含内容：表名、列名、数据类型、是否可空、列注释
   - **完全不含任何业务数据**

2. **发送给LLM的内容** (`internal/llm/prompts.go:32-40`)
   ```go
   SQLGenerationPromptTemplate = `你是一个专业的SQL专家。根据数据库Schema和用户问题，生成准确的SQL查询语句。

   {{.Schema}}  // ← 这里只有表结构，没有实际数据！

   用户问题: {{.Question}}

   请只返回SQL语句，不要包含任何解释或注释。确保SQL语法正确且符合MySQL规范。`
   ```

3. **LLM的作用范围**
   - ✅ 接收：数据库Schema（表结构）
   - ✅ 接收：用户的自然语言问题
   - ✅ 生成：SQL查询语句
   - ❌ 不访问：数据库中的实际业务数据

#### 关键代码位置

| 文件 | 行号 | 说明 |
|------|------|------|
| `internal/database/schema.go` | 168-191 | Schema解析，只读取表结构元数据 |
| `internal/handler/query.go` | 124 | 格式化Schema为Prompt发送给LLM |
| `internal/llm/client.go` | 74-127 | LLM只接收Schema和问题，生成SQL |

---

### ❓ 系统会执行不可逆操作吗？

**答案：绝对不会！**

系统实现了**多层安全防护机制**，确保只执行安全的SELECT查询。

---

## 🛡️ 安全防护机制

### 第一层：SQL防火墙 (`pkg/security/firewall.go`)

防火墙在SQL执行前进行严格检查，确保只有安全的查询能被执行。

#### 阻止的危险关键字

```go
blockedKeywords: []string{
    // 数据修改操作
    "DROP", "DELETE", "UPDATE", "INSERT",
    // DDL操作
    "ALTER", "CREATE", "TRUNCATE",
    // 权限管理
    "GRANT", "REVOKE",
    // 执行和调用
    "EXECUTE", "CALL",
    // 信息泄露
    "EXPLAIN", "SHOW", "DESCRIBE", "DESC",
    // 系统操作
    "USE", "SET", "LOCK", "UNLOCK",
    // 数据替换
    "REPLACE", "LOAD",
}
```

#### 只允许的查询类型

```go
allowedPrefixes: []string{
    "SELECT",  // 标准查询
    "WITH",    // CTE（公用表表达式）
}
```

#### 防火墙功能清单

| 功能 | 说明 | 代码位置 |
|------|------|----------|
| 注释注入防护 | 阻止`--`、`#`、`/* */`注释 | `firewall.go:131-149` |
| 多语句防护 | 阻止分号分隔的多条语句 | `firewall.go:151-186` |
| 括号平衡检查 | 确保SQL括号匹配 | `firewall.go:92-129` |
| 智能上下文 | 允许`ORDER BY name DESC`中的DESC | `firewall.go:211-266` |
| 前缀检查 | 只允许SELECT/WITH开头 | `firewall.go:188-209` |

#### 防火墙检查流程

```go
// pkg/security/firewall.go:38-74
func (f *Firewall) Check(sql string) error {
    // 1. 去除前后空格
    // 2. 检查括号平衡
    // 3. 检查是否包含注释
    // 4. 检查是否包含多个语句（分号）
    // 5. 转换为大写进行关键字检查
    // 6. 检查是否以允许的前缀开头（SELECT/WITH）
    // 7. 检查是否包含危险关键字
    return nil
}
```

---

### 第二层：Prompt约束 (`internal/llm/prompts.go`)

系统Prompt明确指示LLM只生成安全的SELECT查询。

```go
// internal/llm/prompts.go:286-302
func GenerateSystemPrompt() string {
    return `你是一个专业的SQL助手，专门负责将自然语言问题转换为准确的SQL查询语句。

    你的职责：
    1. 理解用户的自然语言问题
    2. 根据数据库Schema生成正确的SQL查询
    3. 确保SQL语法正确且符合MySQL规范
    4. 只返回SQL语句，不要包含任何解释或注释

    注意事项：
    - 只使用SELECT查询，不要使用DELETE、UPDATE、INSERT等修改数据的语句
    - 确保列名和表名与Schema中定义的完全一致
    - 使用适当的JOIN来关联多个表
    - 使用WHERE、GROUP BY、ORDER BY等子句来精确查询
    - 当需要聚合时，使用COUNT、SUM、AVG等聚合函数
    - 使用LIMIT来限制返回的结果数量`
}
```

---

### 第三层：LLM参数配置 (`internal/llm/client.go`)

使用低温度参数确保输出的确定性和安全性。

```go
// internal/llm/client.go:94-95
request := GLMRequest{
    Model: c.model,
    Messages: []GLMMessage{...},
    Temperature: 0.1,  // ← 极低温度，确保输出确定性
    MaxTokens:   1000,  // ← 限制输出长度
}
```

**参数说明：**
- `Temperature: 0.1` - 低温度确保LLM输出稳定、可预测
- `MaxTokens: 1000` - 限制生成的SQL长度，防止复杂注入

---

### 第四层：执行前验证 (`internal/sql/executor.go`)

在SQL真正执行数据库前，再次进行防火墙验证。

```go
// internal/sql/executor.go:34-47
func (e *Executor) Execute(ctx context.Context, sqlQuery string) (*ExecuteResult, error) {
    // 1. 安全检查
    if err := e.firewall.Check(sqlQuery); err != nil {
        return nil, fmt.Errorf("SQL安全检查失败: %w", err)
    }

    // 2. 执行查询（只有通过防火墙检查的SQL才会执行）
    rows, err := e.db.WithContext(ctx).Raw(sqlQuery).Rows()
    // ...
}
```

---

## 🔐 安全架构总结

### 数据安全层

| 层级 | 机制 | 防护内容 |
|------|------|----------|
| L1 | Schema隔离 | LLM只接收表结构，不接触业务数据 |
| L2 | Prompt约束 | 系统Prompt明确禁止生成修改性SQL |
| L3 | LLM参数 | 低温度确保输出稳定 |

### 执行安全层

| 层级 | 机制 | 防护内容 |
|------|------|----------|
| L1 | 关键字过滤 | 阻止所有数据修改和DDL操作 |
| L2 | 前缀检查 | 只允许SELECT/WITH开头 |
| L3 | 注入防护 | 阻止注释和多语句注入 |
| L4 | 括号检查 | 防止括号注入攻击 |
| L5 | 执行前验证 | 最后的安全检查点 |

---

## ⚠️ 潜在安全风险与建议

### 当前存在的风险

| 风险 | 严重程度 | 说明 |
|------|----------|------|
| API Key泄露 | 中 | 需确保配置文件安全 |
| 错误信息泄露 | 低 | SQL错误可能暴露表结构 |
| 缺少访问控制 | 中 | 当前无用户认证机制 |
| 缺少审计日志 | 低 | 无法追踪查询历史 |

### 生产环境建议

#### 1. 数据库连接安全
```go
// 使用只读数据库账户
db.User = "nlq_readonly"
db.Password = "strong_password_here"
db.Host = "localhost"
// 赋予只读权限：GRANT SELECT ON database.* TO 'nlq_readonly'@'localhost'
```

#### 2. API Key管理
```bash
# 使用环境变量存储敏感配置
export GLM_API_KEY="your-actual-api-key"
export DB_PASSWORD="your-db-password"
```

#### 3. 错误信息处理
```go
// 生产环境隐藏详细错误信息
if isProduction {
    return fmt.Errorf("查询失败，请联系管理员")
}
// 开发环境返回详细错误用于调试
return fmt.Errorf("执行SQL失败: %w", err)
```

#### 4. 添加请求限流
```go
// 限制单用户请求频率
rateLimiter := NewRateLimiter(10, time.Minute) // 每分钟最多10次
```

#### 5. 查询审计日志
```go
// 记录所有生成的SQL（用于审计）
logger.Info("SQL Generated",
    "question", question,
    "sql", generatedSQL,
    "user", userID,
    "timestamp", time.Now(),
)
```

---

## 🧪 安全测试建议

### 单元测试覆盖

```go
// pkg/security/firewall_test.go
func TestFirewall_BlocksDangerousKeywords(t *testing.T) {
    dangerousQueries := []string{
        "DELETE FROM users",
        "DROP TABLE users",
        "UPDATE users SET name='test'",
        "INSERT INTO users VALUES (1, 'test')",
    }

    for _, query := range dangerousQueries {
        err := firewall.Check(query)
        assert.Error(t, err)
    }
}

func TestFirewall_AllowsSafeQueries(t *testing.T) {
    safeQueries := []string{
        "SELECT * FROM users",
        "WITH cte AS (SELECT * FROM users) SELECT * FROM cte",
        "SELECT name FROM users ORDER BY created_at DESC",
    }

    for _, query := range safeQueries {
        err := firewall.Check(query)
        assert.NoError(t, err)
    }
}
```

### 渗透测试

尝试以下攻击方式，确保防火墙能有效阻止：

1. **SQL注入尝试**
   ```sql
   SELECT * FROM users WHERE name = 'admin' --'
   SELECT * FROM users WHERE name = 'admin'; DROP TABLE users--'
   ```

2. **注释注入**
   ```sql
   SELECT * FROM users/* comment */WHERE id = 1
   ```

3. **关键字伪装**
   ```sql
   SELECT * FROM users ORDER BY DESC  -- 尝试误用DESC
   ```

---

## 📊 安全性检查清单

部署前检查：

- [ ] API Key已通过安全方式配置（非硬编码）
- [ ] 数据库使用只读账户
- [ ] 防火墙规则已测试并验证
- [ ] 错误信息在生产环境已隐藏
- [ ] 已添加请求频率限制
- [ ] 已配置查询审计日志
- [ ] 已进行渗透测试
- [ ] 已配置HTTPS（生产环境）
- [ ] 已添加用户认证机制（可选）

---

## 📞 安全问题报告

如发现安全问题，请：
1. 不要公开issue
2. 私密联系项目维护者
3. 详细描述复现步骤
4. 等待确认后再公开

---

**文档版本**: 1.0.0
**最后更新**: 2025-03-17
**维护者**: NLQ开发团队
