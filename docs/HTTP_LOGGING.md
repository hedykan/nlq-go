# HTTP请求日志功能说明

## 概述

本小姐已经为NLQ服务器添加了完整的HTTP请求日志记录功能！这个功能可以记录每个HTTP请求的详细信息，包括请求运行的各个阶段。

## 日志功能特性

### 1. HTTP请求日志中间件

每个HTTP请求都会被自动记录，包括以下信息：

- **请求ID**：每个请求都有唯一的标识符（REQ-1, REQ-2, ...）
- **请求方法**：GET, POST等
- **请求路径**：API端点路径
- **客户端地址**：请求来源IP和端口
- **User-Agent**：客户端标识
- **执行时长**：请求处理时间（毫秒）
- **状态码**：HTTP响应状态码

### 2. 阶段日志记录

对于复杂的查询请求，系统会记录以下阶段的详细信息：

#### HandleQuery处理阶段：

1. **阶段1.请求解析** - 解析HTTP请求体
2. **阶段2.请求验证** - 验证请求参数
3. **阶段3.开始查询** - 开始执行查询处理
4. **阶段4.查询成功/失败** - 查询处理结果
5. **阶段5.构建响应** - 构建HTTP响应

#### TwoPhaseQueryHandler处理阶段：

1. **阶段1.选择相关表** - 从数据库中选择相关表
2. **阶段2-3.构建Schema并生成SQL** - 根据选定的表构建Schema并生成SQL
3. **阶段4.验证SQL** - 验证生成的SQL语法
4. **阶段5.执行SQL** - 执行SQL并返回结果

### 3. 日志级别

系统支持以下日志级别：

- **DEBUG** 🔍 - 调试信息（默认不显示）
- **INFO** ℹ️ - 一般信息
- **WARN** ⚠️ - 警告信息
- **ERROR** ❌ - 错误信息

可以通过 `utils.SetLogLevel(level)` 调整日志级别。

## 日志输出示例

### 成功的查询请求：

```
2026-03-18 19:21:18.760 ℹ️  [INFO] ════════════════════════════════════════════════════════════════
2026-03-18 19:21:18.760 ℹ️  [INFO] 📨 [HTTP请求] ID=REQ-1 | Method=POST | Path=/api/v1/query?verbose=true
2026-03-18 19:21:18.760 ℹ️  [INFO] 📍 [客户端地址] RemoteAddr=127.0.0.1:12345
2026-03-18 19:21:18.760 ℹ️  [INFO] 🔑 [User-Agent] Mozilla/5.0
2026-03-18 19:21:18.761 ℹ️  [INFO] 🔄 [阶段-1.请求解析] ID=REQ-1
2026-03-18 19:21:18.761 ℹ️  [INFO]    ├─ handler: QueryHandler.HandleQuery
2026-03-18 19:21:18.762 ℹ️  [INFO] 🔄 [阶段-2.请求验证] ID=REQ-1
2026-03-18 19:21:18.762 ℹ️  [INFO]    ├─ question: 查询所有用户
2026-03-18 19:21:18.762 ℹ️  [INFO]    ├─ knowledge_base:
2026-03-18 19:21:18.762 ℹ️  [INFO]    ├─ verbose: true
2026-03-18 19:21:18.763 ℹ️  [INFO] 🔄 [阶段-3.开始查询] ID=REQ-1
2026-03-18 19:21:18.763 ℹ️  [INFO]    ├─ question: 查询所有用户
2026-03-18 19:21:18.764 ℹ️  [INFO]    └─ 📊 [阶段1] 开始选择相关表...
2026-03-18 19:21:18.770 ℹ️  [INFO]    └─ ✅ [阶段1] 表选择完成 | 主要表: [users] | 次要表: []
2026-03-18 19:21:18.771 ℹ️  [INFO]    └─ 📝 [阶段2-3] 构建Schema并生成SQL...
2026-03-18 19:21:18.775 ℹ️  [INFO]    └─ ✅ [阶段2-3] SQL生成成功 | SQL: SELECT * FROM users
2026-03-18 19:21:18.776 ℹ️  [INFO]    └─ 🔍 [阶段4] 验证SQL...
2026-03-18 19:21:18.777 ℹ️  [INFO]    └─ ✅ [阶段5] SQL执行成功 | 返回 10 行
2026-03-18 19:21:18.778 ℹ️  [INFO] 🔄 [阶段-4.查询成功] ID=REQ-1
2026-03-18 19:21:18.778 ℹ️  [INFO]    ├─ sql: SELECT * FROM users
2026-03-18 19:21:18.778 ℹ️  [INFO]    ├─ duration_ms: 15
2026-03-18 19:21:18.778 ℹ️  [INFO]    ├─ row_count: 10
2026-03-18 19:21:18.779 ℹ️  [INFO] 🔄 [阶段-5.构建响应] ID=REQ-1
2026-03-18 19:21:18.779 ℹ️  [INFO]    ├─ query_id: qry_20260318_a0452143
2026-03-18 19:21:18.779 ℹ️  [INFO]    ├─ success: true
2026-03-18 19:21:18.780 ℹ️  [INFO] ✅ [请求成功] ID=REQ-1 | Status=200 | Duration=20ms
2026-03-18 19:21:18.780 ℹ️  [INFO] ════════════════════════════════════════════════════════════════
```

### 失败的查询请求：

```
2026-03-18 19:21:18.760 ℹ️  [INFO] ════════════════════════════════════════════════════════════════
2026-03-18 19:21:18.760 ℹ️  [INFO] 📨 [HTTP请求] ID=REQ-2 | Method=POST | Path=/api/v1/query
2026-03-18 19:21:18.760 ℹ️  [INFO] 📍 [客户端地址] RemoteAddr=127.0.0.1:54321
2026-03-18 19:21:18.761 ℹ️  [INFO] 🔄 [阶段-1.请求解析] ID=REQ-2
2026-03-18 19:21:18.761 ℹ️  [INFO]    ├─ handler: QueryHandler.HandleQuery
2026-03-18 19:21:18.762 ℹ️  [INFO] 🔄 [阶段-2.请求验证] ID=REQ-2
2026-03-18 19:21:18.762 ℹ️  [INFO]    ├─ question: 错误的查询
2026-03-18 19:21:18.763 ℹ️  [INFO] 🔄 [阶段-3.开始查询] ID=REQ-2
2026-03-18 19:21:18.764 ℹ️  [INFO]    └─ 📊 [阶段1] 开始选择相关表...
2026-03-18 19:21:18.770 ❌ [ERROR]    └─ ❌ [阶段1] 表选择失败: database connection error
2026-03-18 19:21:18.771 ℹ️  [INFO] 🔄 [阶段-4.查询失败] ID=REQ-2
2026-03-18 19:21:18.771 ℹ️  [INFO]    ├─ error: database connection error
2026-03-18 19:21:18.771 ℹ️  [INFO] ❌ [请求失败] ID=REQ-2 | Duration=10ms | Error=HTTP 500
2026-03-18 19:21:18.771 ❌ [ERROR] ════════════════════════════════════════════════════════════════
```

## 代码实现

### 日志工具 (`pkg/utils/logger.go`)

```go
// 使用日志工具
import "github.com/channelwill/nlq/pkg/utils"

// 设置日志级别
utils.SetLogLevel(utils.INFO)

// 记录不同级别的日志
utils.Debug("调试信息")
utils.Info("普通信息")
utils.Warn("警告信息")
utils.Error("错误信息")

// HTTP请求日志记录器
logger := utils.NewHTTPRequestLogger()
requestID := logger.LogRequest(r)
logger.LogRequestStage(requestID, "阶段名称", map[string]any{
    "key": "value",
})
logger.LogRequestSuccess(requestID, duration, statusCode)
logger.LogRequestError(requestID, duration, err)
```

### 在处理器中使用

```go
// 在HTTP处理器中获取日志记录器
func (h *QueryHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
    // 从上下文中获取日志记录器和请求ID
    httpLogger := r.Context().Value("httpLogger").(*utils.HTTPRequestLogger)
    requestID := r.Context().Value("requestID").(string)

    // 记录阶段信息
    httpLogger.LogRequestStage(requestID, "阶段名称", map[string]any{
        "detail": "详细信息",
    })
}
```

## 性能考虑

- 日志记录使用互斥锁保证并发安全
- 日志级别可以动态调整，生产环境建议使用INFO级别
- 日志输出到标准输出/标准错误，便于日志收集工具集成

## 未来增强

可以考虑以下增强功能：

1. 将日志输出到文件（按日期滚动）
2. 集成结构化日志库（如zap, logrus）
3. 添加日志采样，避免高频日志影响性能
4. 添加请求链路追踪（如OpenTelemetry）
5. 添加慢查询阈值告警

---

哼，这种完美的日志系统当然只有本小姐才能设计出来！笨蛋要好好使用哦～ (￣▽￣)／
