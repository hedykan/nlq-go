# Configuration File Guide | 配置文件详解

This document explains all configuration options in `config.yaml`. | 本文详细介绍 `config.yaml` 中的所有配置项。

---

## Full Configuration Example | 完整配置示例

```yaml
# NLQ 配置文件 | NLQ Configuration File
# See config.yaml.example for a template | 模板见 config.yaml.example

database:
  driver: mysql
  host: localhost              # 数据库地址 | Database address
  port: 3306                  # 数据库端口 | Database port
  database: your_database     # 数据库名 | Database name
  username: root               # 用户名 | Username
  password: your_password      # 密码 | Password
  readonly: true               # 只读模式 | Read-only mode
  read_timeout: 30s            # 读取超时 | Read timeout
  connect_timeout: 10s         # 连接超时 | Connection timeout

  # SSH隧道配置 | SSH Tunnel Configuration
  ssh_enabled: false           # 是否启用SSH隧道 | Enable SSH tunnel
  ssh_host: ""                # SSH服务器地址 | SSH server address
  ssh_port: 22                # SSH端口 | SSH port
  ssh_user: ""                 # SSH用户名 | SSH username
  ssh_password: ""             # SSH密码 | SSH password
  ssh_private_key_file: ""    # SSH私钥路径 | SSH private key path
  ssh_key_passphrase: ""      # 私钥密码短语 | Private key passphrase

llm:
  provider: zhipuai           # 提供商: zhipuai, minimax, openai, azure, ollama | Provider
  model: glm-4.7            # 模型名称 | Model name
  default_model: glm-4-flash  # 默认模型 | Default model
  api_key: "${LLM_API_KEY}"  # API密钥 | API key
  base_url: ""                # API地址（为空使用默认值）| API URL
  max_retries: 3             # 最大重试次数 | Max retries
  timeout: 60s               # 超时时间 | Timeout
  temperature: 0.0           # 温度参数（越低越确定，0.0-1.0）| Temperature
  max_tokens: 2048           # 最大生成token数 | Max tokens

security:
  mode: strict               # 严格模式：只允许SELECT | Strict mode: SELECT only
  check_comments: true        # 检查SQL注释注入 | Check SQL comment injection
  check_semicolon: true      # 检查多语句执行 | Check multi-statement execution

server:
  host: "0.0.0.0"            # 监听地址 | Listen address
  port: 8080                 # 监听端口 | Listen port
  query_timeout: 300s        # 查询超时 | Query timeout
  read_timeout: 300s         # 读取超时 | Read timeout
  write_timeout: 300s        # 写入超时 | Write timeout
  enable_cors: true          # 启用CORS | Enable CORS

field_aliases:
  name: ["username", "shop_name"]  # 字段别名映射 | Field alias mapping
  user: ["username", "customer_name"]
  customer: ["customer_name", "client_name"]
  email: ["email_address", "mail"]
  phone: ["mobile", "telephone"]
  time: ["created_at", "updated_at"]
  status: ["state", "level"]
  price: ["amount", "cost", "total"]

query:
  mode: simple               # 查询模式: simple, two_phase, auto
  table_count_threshold: 50  # 自动模式切换阈值

logging:
  level: info                # 日志级别: debug, info, warn, error
  format: text               # 日志格式: text, json
```

---

## Database Configuration | 数据库配置

### Basic Fields | 基本字段

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `driver` | string | `mysql` | Database driver | 数据库驱动 |
| `host` | string | `localhost` | Database host | 数据库地址 |
| `port` | int | `3306` | Database port | 数据库端口 |
| `database` | string | - | Database name | 数据库名 |
| `username` | string | - | Database username | 数据库用户名 |
| `password` | string | - | Database password | 数据库密码 |
| `readonly` | bool | `true` | Read-only mode | 只读模式 |
| `read_timeout` | duration | `30s` | Read timeout | 读取超时 |
| `connect_timeout` | duration | `10s` | Connection timeout | 连接超时 |

### SSH Tunnel | SSH隧道

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `ssh_enabled` | bool | `false` | Enable SSH tunnel | 启用SSH隧道 |
| `ssh_host` | string | - | SSH server address | SSH服务器地址 |
| `ssh_port` | int | `22` | SSH port | SSH端口 |
| `ssh_user` | string | - | SSH username | SSH用户名 |
| `ssh_password` | string | - | SSH password | SSH密码 |
| `ssh_private_key_file` | string | - | Private key path | 私钥路径 |
| `ssh_key_passphrase` | string | - | Key passphrase | 私钥密码短语 |

---

## LLM Configuration | LLM配置

### Supported Providers | 支持的提供商

| Provider | Models | Base URL | Description | 描述 |
|----------|--------|----------|-------------|------|
| `zhipuai` | glm-4, glm-4-plus, glm-4-flash | `https://open.bigmodel.cn/api/paas/v4/` | 智谱AI | ZhipuAI |
| `minimax` | M2-her, M2.5, M2.7 | `https://api.minimaxi.com` | MiniMax | MiniMax |
| `openai` | gpt-4, gpt-3.5-turbo | `https://api.openai.com/v1` | OpenAI | OpenAI |
| `azure` | gpt-4, gpt-35-turbo | Azure endpoint | Azure OpenAI | Azure OpenAI |
| `ollama` | llama2, mistral | `http://localhost:11434/v1` | Ollama本地 | Ollama local |

### LLM Fields | LLM字段

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `provider` | string | `zhipuai` | LLM provider | LLM提供商 |
| `model` | string | `glm-4.7` | Model name | 模型名称 |
| `default_model` | string | `glm-4-flash` | Default model | 默认模型 |
| `api_key` | string | - | API key (supports env var) | API密钥（支持环境变量）|
| `base_url` | string | - | API base URL | API地址 |
| `max_retries` | int | `3` | Max retry attempts | 最大重试次数 |
| `timeout` | duration | `60s` | Request timeout | 请求超时 |
| `temperature` | float | `0.0` | Temperature (0.0-1.0) | 温度参数 |
| `max_tokens` | int | `2048` | Max output tokens | 最大输出token数 |

### Temperature 说明

- **0.0** - 最确定性输出，适合SQL生成 | Most deterministic, good for SQL generation
- **0.1-0.3** - 略有随机性 | Slightly random
- **0.5-0.7** - 中等随机性 | Medium randomness
- **0.8-1.0** - 高随机性，高创造性 | High randomness, creative

---

## Security Configuration | 安全配置

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `mode` | string | `strict` | Security mode (`strict` only) | 安全模式 |
| `check_comments` | bool | `true` | Block SQL comments (`--`, `#`, `/* */`) | 拦截SQL注释 |
| `check_semicolon` | bool | `true` | Block multi-statements | 拦截多语句 |

### Security Modes | 安全模式

| Mode | Description | 描述 |
|------|-------------|------|
| `strict` | Only SELECT allowed, blocks dangerous keywords | 只允许SELECT，拦截危险关键字 |
| `normal` | Basic SQL injection protection | 基础SQL注入防护 |
| `off` | No SQL checking (not recommended) | 关闭检查（不推荐）|

---

## Server Configuration | 服务器配置

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `host` | string | `0.0.0.0` | Listen address | 监听地址 |
| `port` | int | `8080` | Listen port | 监听端口 |
| `query_timeout` | duration | `300s` | Query execution timeout | 查询执行超时 |
| `read_timeout` | duration | `300s` | HTTP read timeout | HTTP读取超时 |
| `write_timeout` | duration | `300s` | HTTP write timeout | HTTP写入超时 |
| `enable_cors` | bool | `true` | Enable CORS | 启用CORS |

---

## Query Configuration | 查询配置

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `mode` | string | `simple` | Query mode | 查询模式 |
| `table_count_threshold` | int | `50` | Auto mode switch threshold | 自动模式切换阈值 |

### Query Modes | 查询模式

| Mode | Description | 适用场景 | Use Case |
|------|-------------|----------|----------|
| `simple` | Single-step SQL generation | 表数量 < 50 | Tables < 50 |
| `two_phase` | Table selection + SQL generation | 大型数据库 | Large databases |
| `auto` | Auto-select based on table count | 自动选择 | Automatic |

---

## Field Aliases | 字段别名

用于两阶段查询中的字段匹配，提高查询准确性。 | Used in two-phase query for field matching.

```yaml
field_aliases:
  name: ["username", "shop_name", "customer_name"]
  user: ["username", "customer_name", "user_name"]
  email: ["email_address", "mail", "email"]
  phone: ["mobile", "telephone", "contact_number"]
  time: ["created_at", "updated_at", "timestamp"]
  status: ["state", "level", "condition"]
  price: ["amount", "cost", "total"]
```

---

## Logging Configuration | 日志配置

| Field | Type | Default | Description | 描述 |
|-------|------|---------|-------------|------|
| `level` | string | `info` | Log level | 日志级别 |
| `format` | string | `text` | Log format | 日志格式 |

### Log Levels | 日志级别

| Level | Description | 描述 |
|-------|-------------|------|
| `debug` | Detailed debugging info | 详细调试信息 |
| `info` | General information | 一般信息 |
| `warn` | Warnings | 警告 |
| `error` | Errors only | 仅错误 |

### Log Formats | 日志格式

| Format | Description | 描述 |
|--------|-------------|------|
| `text` | Human-readable text | 人类可读文本 |
| `json` | JSON format for log aggregation | JSON格式，用于日志收集 |

---

## Environment Variables | 环境变量

Use `${VAR_NAME}` syntax in config.yaml | 在 config.yaml 中使用 `${VAR_NAME}` 语法

| Variable | Description | 描述 |
|---------|-------------|------|
| `LLM_API_KEY` | LLM API key | LLM API密钥 |
| `DATABASE_PASSWORD` | Database password | 数据库密码 |
| `SSH_PASSWORD` | SSH password | SSH密码 |

---

## Quick Start Template | 快速开始模板

See [config.yaml.example](../config/config.yaml.example) for a ready-to-use template. |  готовый шаблон见 [config.yaml.example](../config/config.yaml.example)。
