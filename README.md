# NLQ - Natural Language Query

> A powerful natural language database query tool that uses LLM to convert natural language into SQL queries

[![Go Report Card](https://goreportcard.com/badge/github.com/channelwill/nlq)](https://goreportcard.com/report/github.com/channelwill/nlq)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## 项目简介 | Project Introduction

NLQ是一个创新的自然语言查询工具，允许用户使用自然语言提问，自动转换为SQL查询并返回人类可读的结果。

NLQ is an innovative natural language query tool that allows users to ask questions in natural language, automatically converts them to SQL queries, and returns human-readable results.

**核心问题解决 | Key Problem Solved**：
- ❌ 过去：写SQL需要掌握数据库结构、SQL语法
- ✅ 现在：直接用中文或英文提问，如"去年销售额前三的客户是谁？"

---

## 🌟 核心特性 | Core Features

### 1. 🗣️ 自然语言接口 | Natural Language Interface
- 支持**中文**和**英文**提问
- 自动转换为精确的SQL查询
- 示例 | Examples：
  - `"有多少个用户？" → SELECT COUNT(*) FROM boom_user`
  - `"查询VIP用户的数量"` → 自动识别level='C'条件

### 2. 📚 智能知识库系统 | Intelligent Knowledge Base System
- 提供业务规则、字段说明文档
- LLM自动学习业务上下文
- 支持Markdown格式文档
- 目录结构 | Directory Structure：
```
knowledge/
├── business_rules.md       # 业务规则 | Business Rules
├── field_explanations.md  # 字段说明 | Field Explanations
├── positive/              # 正面示例 | Positive Examples
│   ├── positive_examples.md
│   └── positive_pool.md
└── negative/              # 错误模式 | Error Patterns
    ├── negative_examples.md
    └── negative_pool.md
```

### 3. 🔄 反馈机制与自学习 | Feedback Mechanism & Self-Learning

**独创的反馈回路系统 | Unique Feedback Loop System**：

```
┌─────────────┐      ✅ 同意       ┌─────────────┐
│  用户查询    │ ───────────────→  │ 正面知识库   │
│  User Query │                  │Positive Pool│
└─────────────┘                  └──────┬──────┘
       │                                   │
       │  ❌ 反对                          │ 自动合并
       ↓                                   ↓
┌─────────────┐                  ┌─────────────┐
│ 错误模式    │ ←─────────────── │ Merger合并器 │
│Negative Pool│    纠正SQL      │   Merger    │
└─────────────┘                  └─────────────┘
```

**反馈类型 | Feedback Types**：
- ✅ **同意 (Positive)**：查询结果符合预期 → 自动加入正面知识库
- ❌ **反对 (Negative)**：查询结果不符合预期 → 自动加入错误模式库
- 🔧 **纠正 (Correction)**：用户提供正确的SQL → 合并到知识库

**反馈API | Feedback API**：
```bash
# 同意
curl "http://localhost:8080/feedback/positive/{query_id}"

# 反对
curl "http://localhost:8080/feedback/negative/{query_id}"

# 提交纠正
curl -X POST "http://localhost:8080/feedback/submit" \
  -d '{"query_id":"xxx","correct_sql":"SELECT ..."}'
```

### 4. 🔒 严格安全防护 | Strict Security
- **SELECT-only策略**：只允许查询操作
- **多层SQL注入防护**：
  - 危险关键字拦截（DROP, DELETE, UPDATE等）
  - SQL注释检测（--, #, /* */）
  - 分号多语句检测
  - 括号平衡检查
- **LLM数据隔离**：只接收Schema元数据，不接触业务数据
- **只读数据库连接**

### 5. 🧪 TDD开发 | Test-Driven Development
- 85%+ 测试覆盖率
- 100+ 安全模块测试用例
- 严格的测试驱动开发流程

### 6. ⚡ 智能Schema解析 | Intelligent Schema Parsing
- 自动识别126+数据库表
- 智能字段映射（name→username/shop_name等）
- 两阶段查询优化（适用于大型数据库）
- 支持SSH隧道连接远程数据库

### 7. 🤖 GLM4.7驱动 | Powered by GLM4.7
- 使用智谱AI最新大语言模型
- 支持流式输出
- 120秒HTTP超时保护
- 自动重试机制

---

## 🏗️ 系统架构 | System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户请求 | User Request                   │
│                   (自然语言问题 / Natural Language)               │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     QueryHandler (查询处理器)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Schema     │  │  Knowledge  │  │      Feedback          │  │
│  │  Parser     │◄─│  Loader     │  │      Collector         │  │
│  │  (Schema解析)│  │  (知识库加载)│  │      (反馈收集)        │  │
│  └─────────────┘  └─────────────┘  └───────────┬─────────────┘  │
└────────────────────────────┬───────────────────┼────────────────┘
                             │                   │
          ┌──────────────────┼───────────────────┘
          │                  │        ▲
          ▼                  ▼        │
┌─────────────┐      ┌─────────────┐   │
│   GLM4.7    │      │    SQL      │   │ 自动合并
│   LLM       │      │  Firewall   │   │ Feedback
│  (SQL生成)  │      │  (安全检查)  │   │
└──────┬──────┘      └──────┬──────┘   │
       │                    │          │
       │  SQL               │ 验证通过  │
       ▼                    ▼          │
┌─────────────┐      ┌─────────────┐   │
│   MySQL     │      │   Result    │   │
│  Database   │      │  Formatter  │   │
│   (执行)    │      │   (格式化)  │   │
└─────────────┘      └─────────────┘   │
                                       │
          ┌────────────────────────────┘
          │ 自动记录失败SQL
          ▼
┌─────────────────┐
│  Merger         │ ────► Knowledge (自学习循环)
│ (自动合并反馈)  │
└─────────────────┘
```

---

## 🚀 快速开始 | Quick Start

### 环境要求 | Requirements

- Go 1.21+
- MySQL 8.0+
- 智谱AI API Key

### 安装 | Installation

```bash
# 克隆项目
git clone https://github.com/channelwill/nlq.git
cd nlq

# 安装依赖
go mod download

# 编译
make build
```

### 配置 | Configuration

创建 `config/config.yaml`：

```yaml
database:
  driver: mysql
  host: localhost
  port: 3306
  database: your_database
  username: root
  password: root
  readonly: true

llm:
  provider: zhipuai
  model: glm-4.7
  api_key: ${GLM_API_KEY}
  base_url: https://open.bigmodel.cn/api/paas/v4/

security:
  mode: strict
```

### 使用 | Usage

```bash
# CLI查询
./bin/nlq query "有多少个客户？"
./bin/nlq query "查询VIP用户" --json

# 启动HTTP服务
./bin/nlq-server

# API查询
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "有多少个用户？"}'
```

---

## 📚 知识库使用 | Knowledge Base Usage

### 创建知识库 | Create Knowledge Base

```bash
mkdir -p knowledge
```

创建 `knowledge/business_rules.md`：

```markdown
# 业务规则

## VIP用户定义
- VIP用户：level字段为"C"的用户
- VIP用户享受20%折扣

## 用户状态
- status = 1：活跃用户
- status = 0：非活跃用户
- is_delete = 1：已删除用户
```

创建 `knowledge/field_explanations.md`：

```markdown
# 字段说明

## boom_user 表
- `level`: 用户等级（C=VIP, B=普通, A=新客户）
- `status`: 用户状态（1=活跃, 0=非活跃）
```

### 使用知识库查询 | Query with Knowledge Base

```bash
./bin/nlq query "查询VIP用户" --knowledge ./knowledge
```

---

## 🔄 反馈自学习机制 | Feedback Self-Learning Mechanism

### 核心流程 | Core Flow

```
用户查询 ──→ LLM生成SQL ──→ 执行 ──→ 返回结果
                │                    │
                │              ┌─────┴─────┐
                │              │           │
                │         ✅ 同意      ❌ 反对/失败
                │              │           │
                │              ▼           ▼
                │        positive_pool  negative_pool
                │              │           │
                │              └──► Merger ◄─┘
                │                   │
                │                   ▼
                │           *_examples.md
                │                   │
                └──────────────→ 下次查询
                                    ↑
                              LLM学习知识库
```

### 触发时机 | Triggers

| 触发条件 | 行为 | 合并时机 |
|----------|------|----------|
| 用户同意 | → `positive_pool.md` | 100ms后自动合并 |
| 用户反对 | → `negative_pool.md` | 100ms后自动合并 |
| SQL执行失败 | → `negative_pool.md` | 100ms后自动合并 |

### 去重机制 | Deduplication

基于问题内容去重，相同问题只保留首次反馈。

### 知识库结构 | Knowledge Structure

```
knowledge/
├── business_rules.md          # 业务规则（VIP定义、状态码等）
├── field_explanations.md     # 字段说明
├── positive/
│   ├── positive_examples.md  # ✅ 正确的查询示例
│   └── positive_pool.md      # ⏳ 待合并的正面反馈
└── negative/
    ├── negative_examples.md  # ❌ 错误模式及纠正
    └── negative_pool.md     # ⏳ 待合并的负面反馈
```

### 学习效果示例 | Example

```
# 1. 错误查询
Q: "查询100个最早的用户的name"
SQL: SELECT name FROM boom_customer...  # ❌ 错误表
Error: Unknown column 'name'

# 2. 自动记录 → 100ms后合并到 negative_examples.md

# 3. 下次查询，LLM学习后生成正确SQL
Q: "查询100个最早的用户的name"
SQL: SELECT username FROM boom_user...  # ✅ 正确表
```

---

## 📁 项目结构 | Project Structure

```
NLQ/
├── cmd/                        # 命令行入口 | CLI Entry
│   └── nlq/
│       └── main.go
├── internal/                   # 内部包 | Internal Packages
│   ├── config/               # 配置管理 | Config Management
│   ├── database/             # 数据库操作 | Database Operations
│   ├── feedback/             # 反馈系统 | Feedback System
│   │   ├── collector.go      # 反馈收集 | Feedback Collection
│   │   ├── merger.go         # 知识库合并 | Knowledge Merge
│   │   └── storage.go        # 反馈存储 | Feedback Storage
│   ├── handler/              # 查询处理器 | Query Handler
│   ├── llm/                 # LLM集成 | LLM Integration
│   ├── knowledge/           # 知识库加载 | Knowledge Loading
│   └── sql/                 # SQL处理 | SQL Processing
├── knowledge/                 # 知识库文件 | Knowledge Files
│   ├── positive/            # 正面示例池
│   ├── negative/            # 错误模式池
│   ├── business_rules.md    # 业务规则
│   └── field_explanations.md# 字段说明
├── pkg/
│   └── security/            # SQL安全防火墙 | SQL Firewall
├── config/                   # 配置文件 | Config Files
├── docs/                     # 文档 | Documentation
└── Makefile                 # 构建脚本 | Build Scripts
```

---

## 🛠️ 技术栈 | Tech Stack

- **语言 | Language**：Go 1.21+
- **数据库 | Database**：MySQL 8.0+
- **ORM**：GORM
- **LLM**：tmc/langchaingo + GLM4.7
- **CLI**：Cobra
- **配置 | Config**：Viper + YAML
- **测试 | Testing**：testing + testify

---

## 📖 文档 | Documentation

| 文档 | Description |
|------|-------------|
| [API_GUIDE.md](docs/API_GUIDE.md) | API接口文档 | API Documentation |
| [SECURITY.md](docs/SECURITY.md) | 安全性分析 | Security Analysis |
| [KNOWLEDGE_BASE_GUIDE.md](docs/KNOWLEDGE_BASE_GUIDE.md) | 知识库使用 | Knowledge Base Guide |
| [USAGE_GUIDE.md](docs/USAGE_GUIDE.md) | 使用指南 | Usage Guide |
| [ENHANCEMENT_SUMMARY.md](docs/ENHANCEMENT_SUMMARY.md) | 功能增强总结 | Enhancement Summary |

---

## 🧪 测试 | Testing

```bash
# 运行所有测试
make test

# 生成覆盖率报告
make coverage

# 运行特定模块
go test -v ./internal/feedback/
```

---

## 🤝 贡献 | Contributing

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

**注意**：所有PR必须通过测试，保持85%+测试覆盖率。

---

## 📝 许可证 | License

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢 | Acknowledgments

- [GORM](https://gorm.io/) - 优秀的Go ORM库
- [langchaingo](https://github.com/tmc/langchaingo) - Go版本的LangChain
- [智谱AI](https://open.bigmodel.cn/) - 提供GLM4.7模型支持

---

**🔒 安全提示 | Security Notice**：

本项目已通过安全审查，LLM不会直接访问数据库业务数据，且有严格的SQL防火墙防止不可逆操作。详见[安全性分析文档](docs/SECURITY.md)。

---

*Last Updated: 2026-03-27*
