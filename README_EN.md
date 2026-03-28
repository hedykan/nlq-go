# NLQ - Natural Language Query

> A powerful natural language database query tool that uses LLM to convert natural language into SQL queries

[![Go Report Card](https://goreportcard.com/badge/github.com/hedykan/nlq-go)](https://goreportcard.com/report/github.com/hedykan/nlq-go)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## Project Introduction

NLQ is an innovative natural language query tool that allows users to ask questions in natural language, automatically converts them to SQL queries, and returns human-readable results.

**Key Problem Solved**:
- ❌ Before: Writing SQL requires mastering database structure and SQL syntax
- ✅ Now: Simply ask in Chinese or English, e.g., "Who are the top 3 customers by sales last year?"

---

## Core Features

### 1. Natural Language Interface
- Supports **Chinese** and **English** queries
- Automatically converts to precise SQL queries
- Examples:
  - `"How many users?" → SELECT COUNT(*) FROM boom_user`
  - `"Query VIP users"` → Automatically identifies level='C' conditions

### 2. Intelligent Knowledge Base System
- Provides business rules and field documentation
- LLM automatically learns business context
- Supports Markdown format
- Directory Structure:
```
knowledge/
├── business_rules.md       # Business Rules
├── field_explanations.md  # Field Explanations
├── positive/              # Positive Examples
│   ├── positive_examples.md
│   └── positive_pool.md
└── negative/             # Error Patterns
    ├── negative_examples.md
    └── negative_pool.md
```

### 3. Feedback Mechanism & Self-Learning

**Unique Feedback Loop System**:

```
┌─────────────┐      ✅ Agree       ┌─────────────┐
│  User Query │ ────────────────→  │Positive Pool│
└─────────────┘                    └──────┬──────┘
       │                                   │
       │  ❌ Disagree                      │ Auto Merge
       ↓                                   ↓
┌─────────────┐                  ┌─────────────┐
│Negative Pool│ ←────────────── │   Merger    │
└─────────────┘    Correct SQL  └─────────────┘
```

**Feedback Types**:
- ✅ **Positive**: Query results meet expectations → Auto-added to positive knowledge base
- ❌ **Negative**: Query results don't meet expectations → Auto-added to error pattern base
- 🔧 **Correction**: User provides correct SQL → Merged into knowledge base

**Feedback API**:
```bash
# Agree
curl "http://localhost:8080/feedback/positive/{query_id}"

# Disagree
curl "http://localhost:8080/feedback/negative/{query_id}"

# Submit correction
curl -X POST "http://localhost:8080/feedback/submit" \
  -d '{"query_id":"xxx","correct_sql":"SELECT ..."}'
```

### 4. Strict Security
- **SELECT-only policy**: Only query operations allowed
- **Multi-layer SQL injection protection**:
  - Dangerous keyword blocking (DROP, DELETE, UPDATE, etc.)
  - SQL comment detection (--, #, /* */)
  - Semicolon multi-statement detection
  - Parentheses balance checking
- **LLM data isolation**: Only receives schema metadata, no access to business data
- **Read-only database connection**

### 5. Test-Driven Development
- 85%+ test coverage
- 100+ security module test cases
- Strict TDD workflow

### 6. Intelligent Schema Parsing
- Automatically identifies 126+ database tables
- Smart field mapping (name→username/shop_name, etc.)
- Two-phase query optimization (for large databases)
- Supports SSH tunnel for remote database connections

### 7. Multi-LLM Provider Support
- Supports multiple LLM providers via unified interface
- **Supported Providers**:
  | Provider | Models | Config Value |
  |----------|--------|--------------|
  | ZhipuAI | glm-4, glm-4-plus, glm-4-flash | `zhipuai` |
  | MiniMax | M2-her, M2.5, M2.7 | `minimax` |
  | OpenAI | gpt-4, gpt-3.5-turbo | `openai` |
  | Azure OpenAI | gpt-4, gpt-35-turbo | `azure` |
  | Ollama | llama2, mistral | `ollama` |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Request                              │
│                   (Natural Language Query)                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     QueryHandler                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Schema     │  │  Knowledge  │  │      Feedback          │  │
│  │  Parser     │◄─│  Loader     │  │      Collector         │  │
│  └─────────────┘  └─────────────┘  └───────────┬─────────────┘  │
└────────────────────────────┬───────────────────┼────────────────┘
                             │                   │
           ┌──────────────────┼───────────────────┘
           │                  │        ▲
           ▼                  ▼        │
┌─────────────┐      ┌─────────────┐   │
│   LLM       │      │    SQL     │   │ Auto Merge
│  (SQL Gen)  │      │  Firewall   │   │ Feedback
└──────┬──────┘      └──────┬──────┘   │
       │                    │          │
       │  SQL               │ Valid    │
       ▼                    ▼          │
┌─────────────┐      ┌─────────────┐   │
│   MySQL     │      │   Result    │   │
│  Database   │      │  Formatter  │   │
└─────────────┘      └─────────────┘   │
                             │          │
           └────────────────────────────┘
                             │ Auto Record Failed SQL
                             ▼
┌─────────────────┐
│     Merger      │ ────► Knowledge (Self-Learning Loop)
└─────────────────┘
```

---

## Quick Start

### Requirements

- Go 1.21+
- MySQL 8.0+
- LLM API Key (ZhipuAI, MiniMax, OpenAI, etc.)

### Installation

```bash
# Clone the project
git clone https://github.com/hedykan/nlq-go.git
cd nlq

# Install dependencies
go mod download

# Build
make build
```

### Configuration

Create `config/config.yaml`:

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
  provider: zhipuai    # or minimax, openai, azure, ollama
  model: glm-4.7
  api_key: ${LLM_API_KEY}
  base_url: https://open.bigmodel.cn/api/paas/v4/
  temperature: 0.0     # Lower = more deterministic
  max_tokens: 2048

security:
  mode: strict
```

### Usage

```bash
# CLI query
./bin/nlq query "How many customers?"
./bin/nlq query "Query VIP users" --json

# Start HTTP server
./bin/nlq-server

# API query
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "How many users?"}'
```

---

## Knowledge Base Usage

### Create Knowledge Base

```bash
mkdir -p knowledge
```

Create `knowledge/business_rules.md`:

```markdown
# Business Rules

## VIP User Definition
- VIP users: users with level field = 'C'
- VIP users enjoy 20% discount

## User Status
- status = 1: Active user
- status = 0: Inactive user
- is_delete = 1: Deleted user
```

Create `knowledge/field_explanations.md`:

```markdown
# Field Explanations

## boom_user table
- `level`: User level (C=VIP, B=Normal, A=New)
- `status`: User status (1=Active, 0=Inactive)
```

### Query with Knowledge Base

```bash
./bin/nlq query "Query VIP users" --knowledge ./knowledge
```

---

## Feedback Self-Learning Mechanism

### Core Flow

```
User Query ──→ LLM Generate SQL ──→ Execute ──→ Return Result
                  │                    │
                  │              ┌─────┴─────┐
                  │              │           │
                  │         ✅ Agree     ❌ Disagree/Fail
                  │              │           │
                  │              ▼           ▼
                  │        positive_pool  negative_pool
                  │              │           │
                  │              └──► Merger ◄─┘
                  │                   │
                  │                   ▼
                  │           *_examples.md
                  │                   │
                  └──────────────→ Next Query
                                     ↑
                               LLM Learns from KB
```

### Triggers

| Trigger | Action | Merge Timing |
|---------|--------|--------------|
| User agrees | → `positive_pool.md` | Auto-merge after 100ms |
| User disagrees | → `negative_pool.md` | Auto-merge after 100ms |
| SQL execution fails | → `negative_pool.md` | Auto-merge after 100ms |

### Deduplication

Based on query content deduplication, same query only keeps the first feedback.

### Learning Example

```
# 1. Wrong query
Q: "Query names of 100 earliest users"
SQL: SELECT name FROM boom_customer...  # ❌ Wrong table
Error: Unknown column 'name'

# 2. Auto-record → 100ms later merged to negative_examples.md

# 3. Next query, LLM learns and generates correct SQL
Q: "Query names of 100 earliest users"
SQL: SELECT username FROM boom_user...  # ✅ Correct table
```

---

## Project Structure

```
NLQ/
├── cmd/                        # CLI Entry
│   └── nlq/
│       └── main.go
├── internal/                   # Internal Packages
│   ├── config/               # Config Management
│   ├── database/             # Database Operations
│   ├── feedback/             # Feedback System
│   │   ├── collector.go      # Feedback Collection
│   │   ├── merger.go         # Knowledge Merge
│   │   └── storage.go        # Feedback Storage
│   ├── handler/              # Query Handler
│   ├── llm/                 # LLM Integration (Multi-Provider)
│   ├── knowledge/            # Knowledge Loading
│   └── sql/                 # SQL Processing
├── knowledge/                 # Knowledge Files
│   ├── positive/            # Positive Examples Pool
│   ├── negative/            # Error Patterns Pool
│   ├── business_rules.md    # Business Rules
│   └── field_explanations.md# Field Explanations
├── pkg/
│   └── security/            # SQL Firewall
├── config/                   # Config Files
├── docs/                     # Documentation
│   ├── en/                  # English Docs
│   └── zh/                  # Chinese Docs
└── Makefile                 # Build Scripts
```

---

## Tech Stack

- **Language**: Go 1.21+
- **Database**: MySQL 8.0+
- **ORM**: GORM
- **LLM**: tmc/langchaingo + OpenAI-compatible APIs
- **CLI**: Cobra
- **Config**: Viper + YAML
- **Testing**: testing + testify

---

## Documentation

### English Documentation (docs/en/)

| Document | Description |
|----------|-------------|
| [API_GUIDE.md](docs/en/API_GUIDE.md) | API Documentation |
| [SECURITY.md](docs/en/SECURITY.md) | Security Analysis |
| [KNOWLEDGE_BASE_GUIDE.md](docs/en/KNOWLEDGE_BASE_GUIDE.md) | Knowledge Base Guide |
| [USAGE_GUIDE.md](docs/en/USAGE_GUIDE.md) | Usage Guide |
| [MULTI_LLM_PROVIDER.md](docs/en/MULTI_LLM_PROVIDER.md) | Multi-LLM Provider Guide |
| [CONFIGURATION_GUIDE.md](docs/en/CONFIGURATION_GUIDE.md) | Configuration Guide |
| [LLM_OPTIMIZATION.md](docs/en/LLM_OPTIMIZATION.md) | LLM Optimization |
| [HTTP_LOGGING.md](docs/en/HTTP_LOGGING.md) | HTTP Logging |
| [TESTING_GUIDE.md](docs/en/TESTING_GUIDE.md) | Testing Guide |
| [SSH_TUNNEL_USAGE.md](docs/en/SSH_TUNNEL_USAGE.md) | SSH Tunnel Usage |

### 中文文档 (docs/zh/)

| 文档 | 描述 |
|------|------|
| [PLAN.md](docs/zh/PLAN.md) | 项目计划 |
| [PROJECT_SUMMARY.md](docs/zh/PROJECT_SUMMARY.md) | 项目总结 |

---

## Testing

```bash
# Run all tests
make test

# Generate coverage report
make coverage

# Run specific module
go test -v ./internal/feedback/
```

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

**Note**: All PRs must pass tests and maintain 85%+ test coverage.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- [GORM](https://gorm.io/) - Excellent Go ORM library
- [langchaingo](https://github.com/tmc/langchaingo) - LangChain for Go
- [ZhipuAI](https://open.bigmodel.cn/) - GLM model support
- [MiniMax](https://www.minimaxi.com/) - MiniMax model support

---

**Security Notice**:

This project has passed security review. The LLM does not directly access database business data, and there is a strict SQL firewall to prevent irreversible operations. See [Security Analysis](docs/en/SECURITY.md) for details.

---

*Last Updated: 2026-03-28*
