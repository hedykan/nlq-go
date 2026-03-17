# NLQ HTTP API使用指南

## 🌐 服务架构

NLQ现在提供RESTful API和WebSocket服务，支持外部系统集成。

---

## 🚀 快速开始

### 1. 启动HTTP服务

```bash
# 编译服务器
make build-server

# 启动服务器
./bin/nlq-server

# 或使用环境变量配置
DATABASE_HOST=localhost DATABASE_NAME=loloyal ./bin/nlq-server
```

**服务启动输出**：
```
🤖 使用GLM4.7 LLM: glm-4-plus
🌐 HTTP服务器启动在 http://0.0.0.0:8080
```

### 2. 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

**响应**：
```json
{
  "status": "healthy",
  "timestamp": "2026-03-16T19:13:47+08:00"
}
```

---

## 📋 API接口

### 1. 自然语言查询

**接口**：`POST /api/v1/query`

**请求**：
```json
{
  "question": "查询VIP用户",
  "knowledge_base": "./knowledge",
  "verbose": true
}
```

**响应**：
```json
{
  "success": true,
  "question": "查询VIP用户",
  "sql": "SELECT * FROM boom_user WHERE level = 'C' AND status = 1 AND is_delete = 0",
  "result": [...],
  "count": 973,
  "duration_ms": 6371,
  "metadata": {
    "llm_type": "GLM-4-Plus",
    "use_real_llm": true
  }
}
```

**curl示例**：
```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "boom_user有多少条数据？"}'
```

### 2. SQL查询

**接口**：`POST /api/v1/sql`

**请求**：
```json
{
  "sql": "SELECT * FROM boom_user WHERE id = 1"
}
```

**响应**：
```json
{
  "success": true,
  "sql": "SELECT * FROM boom_user WHERE id = 1",
  "result": [...],
  "count": 1,
  "duration_ms": 2
}
```

**curl示例**：
```bash
curl -X POST http://localhost:8080/api/v1/sql \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT COUNT(*) FROM boom_user"}'
```

### 3. 数据库Schema

**接口**：`GET /api/v1/schema`

**响应**：
```json
{
  "schema": "数据库Schema信息"
}
```

**curl示例**：
```bash
curl http://localhost:8080/api/v1/schema
```

### 4. 表结构

**接口**：`GET /api/v1/schema/{table}`

**响应**：
```json
{
  "table": "boom_user",
  "schema": "表结构信息"
}
```

**curl示例**：
```bash
curl http://localhost:8080/api/v1/schema/boom_user
```

### 5. 服务状态

**接口**：`GET /api/v1/status`

**响应**：
```json
{
  "service": "nlq",
  "version": "1.0.0",
  "status": "running",
  "uptime": "unknown",
  "database": "connected"
}
```

**curl示例**：
```bash
curl http://localhost:8080/api/v1/status
```

---

## 🔧 配置选项

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DATABASE_HOST` | 数据库主机 | localhost |
| `DATABASE_PORT` | 数据库端口 | 3306 |
| `DATABASE_NAME` | 数据库名称 | loloyal |
| `GLM_API_KEY` | GLM API密钥 | - |

### 配置文件

编辑 `config/config.yaml`：

```yaml
server:
  host: 0.0.0.0
  port: 8080
  query_timeout: 10s
  read_timeout: 30s
  write_timeout: 30s
  enable_cors: true

database:
  host: localhost
  port: 3306
  database: loloyal
  username: root
  password: root
  readonly: true

llm:
  provider: zhipuai
  model: glm-4-plus
  api_key: your-api-key-here
  base_url: https://open.bigmodel.cn/api/paas/v4/
```

---

## 🧪 使用示例

### Python示例

```python
import requests
import json

# API基础URL
BASE_URL = "http://localhost:8080/api/v1"

# 自然语言查询
def natural_language_query(question):
    response = requests.post(
        f"{BASE_URL}/query",
        json={"question": question}
    )
    return response.json()

# SQL查询
def sql_query(sql):
    response = requests.post(
        f"{BASE_URL}/sql",
        json={"sql": sql}
    )
    return response.json()

# 使用示例
result = natural_language_query("查询VIP用户")
print(f"SQL: {result['sql']}")
print(f"结果数: {result['count']}")
```

### JavaScript示例

```javascript
const BASE_URL = 'http://localhost:8080/api/v1';

// 自然语言查询
async function query(question) {
  const response = await fetch(`${BASE_URL}/query`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ question })
  });
  return await response.json();
}

// SQL查询
async function sqlQuery(sql) {
  const response = await fetch(`${BASE_URL}/sql`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ sql })
  });
  return await response.json();
}

// 使用示例
query('查询VIP用户').then(result => {
  console.log('SQL:', result.sql);
  console.log('结果数:', result.count);
});
```

### Go示例

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const baseURL = "http://localhost:8080/api/v1"

type QueryRequest struct {
    Question string `json:"question"`
}

type QueryResponse struct {
    Success bool                   `json:"success"`
    SQL      string                 `json:"sql"`
    Result   []map[string]interface{} `json:"result"`
    Count    int                    `json:"count"`
}

func naturalLanguageQuery(question string) (*QueryResponse, error) {
    reqBody, _ := json.Marshal(QueryRequest{Question: question})
    resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result QueryResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

---

## 🛡️ 安全特性

### CORS支持

服务器默认启用CORS，允许跨域请求：

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

### SQL防火墙

所有SQL查询都经过严格的安全检查：

- ✅ 只允许SELECT查询
- ✅ 检查危险关键字（DROP、DELETE等）
- ✅ 检测SQL注入（注释、分号）
- ✅ 括号平衡检查

---

## 📊 响应格式

### 成功响应

```json
{
  "success": true,
  "question": "查询问题",
  "sql": "生成的SQL",
  "result": [...],
  "count": 结果数量,
  "duration_ms": 执行时间毫秒
}
```

### 错误响应

```json
{
  "success": false,
  "error": "错误信息",
  "code": "错误代码"
}
```

---

## 🔍 故障排查

### 问题1：服务器无法启动

**检查**：
```bash
# 检查端口占用
lsof -i :8080

# 检查数据库连接
docker exec mysql mysql -uroot -proot -e "SELECT 1"
```

### 问题2：API返回404

**检查**：
- 确认URL路径正确（`/api/v1/query`）
- 确认HTTP方法正确（POST/GET）

### 问题3：查询失败

**检查**：
```bash
# 查看服务器日志
tail -f /tmp/nlq-server.log

# 测试数据库连接
curl http://localhost:8080/api/v1/status
```

---

## 🚀 性能优化

### 连接池配置

服务器使用GORM连接池，默认配置：

```go
db.DB().SetMaxIdleConns(10)
db.DB().SetMaxOpenConns(100)
db.DB().SetConnMaxLifetime(time.Hour)
```

### 超时配置

- **读超时**：30秒
- **写超时**：30秒
- **查询超时**：10秒

---

## 🎯 最佳实践

### 1. 使用自然语言查询

```bash
# 推荐：使用自然语言
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "查询最近创建的5个用户"}'
```

### 2. 错误处理

```python
try:
    result = natural_language_query("查询用户")
    if result['success']:
        print("查询成功")
    else:
        print(f"查询失败: {result.get('error')}")
except Exception as e:
    print(f"请求失败: {e}")
```

### 3. 性能监控

```bash
# 监控响应时间
time curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "查询用户数量"}'
```

---

## 📚 高级功能

### 知识库集成

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "question": "查询VIP用户",
    "knowledge_base": "./knowledge",
    "verbose": true
  }'
```

### JSON输出

```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "查询用户"}' | jq .
```

---

**享受强大的NLQ API服务！** ✨
