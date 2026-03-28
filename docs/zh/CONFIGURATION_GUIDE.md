# 配置文件详解

本文详细介绍 `config.yaml` 中的所有配置项。

---

## 完整配置示例

```yaml
database:
  driver: mysql
  host: localhost              # 数据库地址
  port: 3306                  # 数据库端口
  database: your_database     # 数据库名
  username: root              # 用户名
  password: your_password     # 密码
  readonly: true              # 只读模式
  read_timeout: 30s          # 读取超时
  connect_timeout: 10s        # 连接超时

  # SSH隧道配置（可选，用于连接远程数据库）
  ssh_enabled: false          # 是否启用SSH隧道
  ssh_host: ""               # SSH服务器地址
  ssh_port: 22                # SSH端口
  ssh_user: ""                # SSH用户名
  ssh_password: ""             # SSH密码（使用私钥认证时留空）
  ssh_private_key_file: ""    # SSH私钥路径
  ssh_key_passphrase: ""      # 私钥密码短语

llm:
  provider: zhipuai           # 提供商: zhipuai, minimax, openai, azure, ollama
  model: glm-4.7              # 模型名称
  default_model: glm-4-flash   # 默认模型
  api_key: "${LLM_API_KEY}"   # 从环境变量读取，或直接填入API Key
  base_url: ""                # API地址（为空则使用默认值）
  max_retries: 3              # 最大重试次数
  timeout: 60s                # 超时时间
  temperature: 0.0            # 温度参数，越低越确定（0.0-1.0）
  max_tokens: 2048            # 最大生成token数

security:
  mode: strict               # 严格模式：只允许SELECT语句
  check_comments: true       # 检查SQL注释注入
  check_semicolon: true      # 检查多语句执行

server:
  host: "0.0.0.0"            # 监听地址
  port: 8080                  # 监听端口
  query_timeout: 300s         # 查询超时时间
  read_timeout: 300s          # 读取超时
  write_timeout: 300s         # 写入超时
  enable_cors: true           # 是否启用CORS

field_aliases:
  name: ["username", "shop_name", "customer_name"]
  user: ["username", "customer_name", "user_name"]
  customer: ["customer_name", "client_name"]
  email: ["email_address", "mail"]
  phone: ["mobile", "telephone", "contact_number"]
  time: ["created_at", "updated_at", "timestamp"]
  status: ["state", "level", "condition"]
  price: ["amount", "cost", "total"]

query:
  mode: simple               # 查询模式: simple, two_phase, auto
  table_count_threshold: 50  # 自动模式切换阈值

logging:
  level: info                 # 日志级别: debug, info, warn, error
  format: text               # 日志格式: text, json
```

---

## 数据库配置

### 基本字段

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `driver` | string | `mysql` | 数据库驱动 |
| `host` | string | `localhost` | 数据库地址 |
| `port` | int | `3306` | 数据库端口 |
| `database` | string | - | 数据库名 |
| `username` | string | - | 数据库用户名 |
| `password` | string | - | 数据库密码 |
| `readonly` | bool | `true` | 只读模式 |
| `read_timeout` | duration | `30s` | 读取超时 |
| `connect_timeout` | duration | `10s` | 连接超时 |

### SSH隧道

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `ssh_enabled` | bool | `false` | 是否启用SSH隧道 |
| `ssh_host` | string | - | SSH服务器地址 |
| `ssh_port` | int | `22` | SSH端口 |
| `ssh_user` | string | - | SSH用户名 |
| `ssh_password` | string | - | SSH密码 |
| `ssh_private_key_file` | string | - | 私钥路径 |
| `ssh_key_passphrase` | string | - | 私钥密码短语 |

---

## LLM配置

### 支持的提供商

| Provider | 模型示例 | Base URL | 描述 |
|----------|----------|----------|------|
| `zhipuai` | glm-4, glm-4-plus, glm-4-flash | `https://open.bigmodel.cn/api/paas/v4/` | 智谱AI |
| `minimax` | M2-her, M2.5, M2.7 | `https://api.minimaxi.com` | MiniMax |
| `openai` | gpt-4, gpt-3.5-turbo | `https://api.openai.com/v1` | OpenAI |
| `azure` | gpt-4, gpt-35-turbo | Azure endpoint | Azure OpenAI |
| `ollama` | llama2, mistral | `http://localhost:11434/v1` | Ollama本地 |

### LLM字段

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `provider` | string | `zhipuai` | LLM提供商 |
| `model` | string | `glm-4.7` | 模型名称 |
| `default_model` | string | `glm-4-flash` | 默认模型 |
| `api_key` | string | - | API密钥（支持环境变量）|
| `base_url` | string | - | API地址 |
| `max_retries` | int | `3` | 最大重试次数 |
| `timeout` | duration | `60s` | 请求超时 |
| `temperature` | float | `0.0` | 温度参数（0.0-1.0）|
| `max_tokens` | int | `2048` | 最大输出token数 |

### Temperature 说明

- **0.0** - 最确定性输出，适合SQL生成
- **0.1-0.3** - 略有随机性
- **0.5-0.7** - 中等随机性
- **0.8-1.0** - 高随机性，高创造性

---

## 安全配置

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `mode` | string | `strict` | 安全模式 |
| `check_comments` | bool | `true` | 拦截SQL注释（`--`, `#`, `/* */`）|
| `check_semicolon` | bool | `true` | 拦截多语句执行 |

### 安全模式

| 模式 | 描述 |
|------|------|
| `strict` | 只允许SELECT，拦截危险关键字 |
| `normal` | 基础SQL注入防护 |
| `off` | 关闭检查（不推荐）|

---

## 服务器配置

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `host` | string | `0.0.0.0` | 监听地址 |
| `port` | int | `8080` | 监听端口 |
| `query_timeout` | duration | `300s` | 查询执行超时 |
| `read_timeout` | duration | `300s` | HTTP读取超时 |
| `write_timeout` | duration | `300s` | HTTP写入超时 |
| `enable_cors` | bool | `true` | 启用CORS |

---

## 查询配置

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `mode` | string | `simple` | 查询模式 |
| `table_count_threshold` | int | `50` | 自动模式切换阈值 |

### 查询模式

| 模式 | 适用场景 |
|------|----------|
| `simple` | 单步SQL生成，表数量 < 50 |
| `two_phase` | 两阶段查询（表筛选+SQL生成），大型数据库 |
| `auto` | 根据表数量自动选择 |

---

## 字段别名

用于两阶段查询中的字段匹配，提高查询准确性。

```yaml
field_aliases:
  name: ["username", "shop_name", "customer_name"]
  user: ["username", "customer_name", "user_name"]
  email: ["email_address", "mail"]
  phone: ["mobile", "telephone"]
  time: ["created_at", "updated_at"]
  status: ["state", "level"]
  price: ["amount", "cost", "total"]
```

---

## 日志配置

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `level` | string | `info` | 日志级别 |
| `format` | string | `text` | 日志格式 |

### 日志级别

| 级别 | 描述 |
|------|------|
| `debug` | 详细调试信息 |
| `info` | 一般信息 |
| `warn` | 警告 |
| `error` | 仅错误 |

### 日志格式

| 格式 | 描述 |
|------|------|
| `text` | 人类可读文本 |
| `json` | JSON格式，用于日志收集 |

---

## 环境变量

在 `config.yaml` 中使用 `${VAR_NAME}` 语法

| 变量 | 描述 |
|------|------|
| `LLM_API_KEY` | LLM API密钥 |
| `DATABASE_PASSWORD` | 数据库密码 |
| `SSH_PASSWORD` | SSH密码 |

---

## 快速开始

参考 [config.yaml.example](../config/config.yaml.example) 获取可直接使用的模板。
