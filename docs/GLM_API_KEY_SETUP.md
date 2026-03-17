# GLM4.7 API Key配置说明

## 📝 配置GLM4.7 API Key

### 方法1：使用配置文件（推荐）

1. 编辑 `config/config.yaml` 文件：

```yaml
llm:
  provider: zhipuai
  model: glm-4-plus
  api_key: "your-api-key-here"  # 替换为你的真实API Key
  base_url: https://open.bigmodel.cn/api/paas/v4/
  max_retries: 3
  timeout: 30s
  temperature: 0.1
```

2. 使用配置文件运行：

```bash
./bin/nlq -c config/config.yaml query "boom_user有多少条数据？"
```

### 方法2：使用环境变量

1. 设置环境变量：

```bash
export GLM_API_KEY="your-api-key-here"
```

2. 在配置文件中使用环境变量：

```yaml
llm:
  api_key: ${GLM_API_KEY}  # 从环境变量读取
```

3. 运行程序：

```bash
./bin/nlq query "boom_user有多少条数据？"
```

### 方法3：直接命令行（临时）

```bash
GLM_API_KEY="your-api-key-here" ./bin/nlq query "boom_user有多少条数据？"
```

---

## 🔑 获取GLM4.7 API Key

### 步骤：

1. 访问智谱AI开放平台：https://open.bigmodel.cn/
2. 注册/登录账号
3. 进入API Key管理页面
4. 创建新的API Key
5. 复制API Key

### 注意事项：

- ⚠️ **不要将API Key提交到代码仓库**
- ⚠️ **妥善保管API Key，避免泄露**
- ⚠️ **定期轮换API Key**

---

## 🎯 验证API Key配置

### 测试命令：

```bash
# 使用详细模式查看是否使用了真实LLM
./bin/nlq query "boom_user有多少条数据？" -v
```

### 预期输出：

**如果API Key配置正确**：
```
🤖 使用GLM4.7 LLM: glm-4-plus
```

**如果API Key未配置**：
```
💡 使用简化SQL生成（未配置API Key）
```

---

## 📊 功能对比

### 使用真实GLM4.7 LLM：

- ✅ **自然语言理解**：真正理解中文问题
- ✅ **复杂SQL生成**：支持多表关联、子查询
- ✅ **上下文记忆**：可以记住对话历史
- ✅ **智能修正**：自动修正SQL语法错误

### 使用简化SQL生成：

- ⚠️ **模式匹配**：仅支持简单的关键词匹配
- ⚠️ **有限支持**：只能处理特定的表和问题
- ⚠️ **无上下文**：每次查询都是独立的

---

## 💡 推荐配置

**开发环境**：使用简化模式（无需API Key）
```bash
./bin/nlq query "boom_user有多少条数据？"
```

**生产环境**：使用真实GLM4.7
```bash
export GLM_API_KEY="your-api-key-here"
./bin/nlq -c config/config.yaml query "去年销售额最高的员工是谁？"
```

---

## 🚨 常见问题

### Q1: API调用失败

**错误信息**：`LLM生成SQL失败: API返回错误状态码: 401`

**解决方案**：
- 检查API Key是否正确
- 确认API Key是否有效（未过期）
- 检查账户余额是否充足

### Q2: 网络超时

**错误信息**：`LLM生成SQL失败: 发送HTTP请求失败: ...`

**解决方案**：
- 检查网络连接
- 增加超时时间（在配置文件中设置 `timeout: 60s`）
- 检查API服务是否可用

### Q3: 生成SQL质量差

**解决方案**：
- 调整 `temperature` 参数（0.0-1.0，越低越确定）
- 在问题中提供更多上下文信息
- 使用更明确的问题描述

---

## 📖 示例配置文件

完整示例配置文件 `config/config.yaml`：

```yaml
# NLQ配置文件

# 数据库配置
database:
  driver: mysql
  host: localhost
  port: 3306
  database: loloyal
  username: root
  password: root
  readonly: true

# LLM配置（GLM4.7）
llm:
  provider: zhipuai
  model: glm-4-plus
  api_key: ${GLM_API_KEY}  # 从环境变量读取
  base_url: https://open.bigmodel.cn/api/paas/v4/
  max_retries: 3
  timeout: 30s
  temperature: 0.1  # 温度参数，越低越确定

# 安全配置
security:
  mode: strict  # 严格模式：只允许SELECT语句
  check_comments: true  # 检查SQL注释注入
  check_semicolon: true  # 检查多语句执行

# 服务器配置
server:
  port: 8080
  query_timeout: 10s

# 日志配置
logging:
  level: info  # debug, info, warn, error
  format: text  # text, json
```

---

**配置完成后，你就可以使用真实的GLM4.7 LLM来进行自然语言查询了！** 🎉
