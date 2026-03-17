# NLQ - Natural Language Query

> 基于自然语言的数据库查询工具，使用LLM将自然语言转换为SQL查询

[![Go Report Card](https://goreportcard.com/badge/github.com/channelwill/nlq)](https://goreportcard.com/report/github.com/channelwill/nlq)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## 🌟 项目简介

NLQ是一个创新的自然语言查询工具，允许用户使用自然语言提问（如"去年销售额前三的员工是谁？"），自动转换为SQL查询并返回人类可读的结果。

**核心特性**：
- 🗣️ **自然语言接口**：用中文提问，自动转换为SQL
- 🔒 **严格安全防护**：只允许SELECT查询，多层SQL注入防护
- 🧪 **TDD开发**：85%+测试覆盖率，严格遵循测试驱动开发
- ⚡ **智能Schema解析**：自动识别数据库表结构
- 🤖 **GLM4.7驱动**：使用智谱AI最新大语言模型

## 🚀 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Docker（用于本地数据库）
- 智谱AI API Key

### 安装

```bash
# 克隆项目
git clone https://github.com/channelwill/nlq.git
cd nlq

# 安装依赖
go mod download

# 编译
make build
```

### 配置

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
  model: glm-4-plus
  api_key: ${GLM_API_KEY}
  base_url: https://open.bigmodel.cn/api/paas/v4/
  max_retries: 3
  timeout: 30s

security:
  mode: strict
  check_comments: true
  check_semicolon: true
```

### 使用

```bash
# 查询客户数量
./bin/nlq query "有多少个客户？"

# 查询销售额最高的用户
./bin/nlq query "谁是去年下单金额最高的用户？"

# JSON格式输出
./bin/nlq query "显示所有用户的数量" --json

# 详细模式
./bin/nlq query "查询年龄大于25岁的用户" -v
```

## 🏗️ 项目架构

```
┌─────────────┐
│   用户输入   │ (自然语言问题)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  QueryHandler│ (查询处理器)
└──────┬──────┘
       │
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌─────────────┐   ┌──────────────┐
│SchemaParser │   │  GLMClient   │
└──────┬──────┘   └──────┬───────┘
       │                 │
       ▼                 ▼
┌─────────────┐   ┌──────────────┐
│   Database  │   │  SQLGenerator│
└─────────────┘   └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │   Firewall   │ (SQL安全检查)
                  └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │   Executor   │ (SQL执行)
                  └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │  Formatter   │ (结果格式化)
                  └──────────────┘
```

## 🔒 安全特性

本项目采用多层安全防护机制，确保系统安全可靠。

> 📖 **详细安全性分析请查看**：[**docs/SECURITY.md**](docs/SECURITY.md)

### ✅ 已实现的安全特性

1. **严格的SELECT-only策略**
   - 只允许SELECT查询语句
   - 拦截所有DDL/DML操作（DROP, DELETE, UPDATE, INSERT等）

2. **SQL注入防护**
   - 危险关键字拦截
   - SQL注释注入检测（--, #, /* */）
   - 分号注入检测
   - 多语句执行防护
   - 括号平衡检查
   - 字符串字面量智能处理

3. **LLM数据隔离**
   - LLM只接收Schema元数据，**不接触实际业务数据**
   - 通过Prompt约束禁止生成修改性SQL

4. **连接安全**
   - 只读数据库连接
   - 连接池管理
   - 查询超时控制

### 测试覆盖率

- **安全模块测试**：10个测试函数，**100+测试用例**
- **总体测试覆盖率**：85%+

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 生成测试覆盖率报告
make coverage

# 运行特定模块测试
go test -v ./internal/database/
go test -v ./pkg/security/
```

## 📁 项目结构

```
NLQ/
├── cmd/                    # 命令行入口
│   └── nlq/
│       └── main.go
├── internal/               # 内部包
│   ├── config/            # 配置管理 ✅
│   ├── database/          # 数据库操作 ✅
│   ├── llm/               # LLM集成 ⬜
│   ├── sql/               # SQL处理 ⬜
│   ├── response/          # 响应格式化 ⬜
│   ├── handler/           # 查询处理器 ⬜
│   └── cmd/               # CLI命令 ⬜
├── pkg/                    # 公共包
│   └── security/          # SQL防火墙 ✅
├── config/                 # 配置文件
├── test/                   # 测试资源
└── Makefile               # 构建脚本
```

## 🔧 开发指南

本项目严格遵循TDD（测试驱动开发）方法论：

1. **红**：先编写失败的测试
2. **绿**：实现最小功能让测试通过
3. **重构**：优化代码质量

### 添加新功能

```bash
# 1. 编写测试
vim internal/yourmodule/feature_test.go

# 2. 运行测试（应该失败）
go test -v ./internal/yourmodule/

# 3. 实现功能
vim internal/yourmodule/feature.go

# 4. 再次运行测试（应该通过）
go test -v ./internal/yourmodule/

# 5. 重构优化
# 6. 提交代码
```

## 📊 当前状态

### ✅ 已完成模块

- [x] 项目初始化和目录结构
- [x] 配置管理模块
- [x] 数据库连接管理
- [x] Schema解析器
- [x] SQL安全防火墙
- [x] LLM客户端集成（GLM-4-Plus）
- [x] SQL生成器（两阶段查询）
- [x] 查询执行器
- [x] CLI接口实现
- [x] WebSocket服务器
- [x] 知识库注入

### 📖 文档

- [API使用指南](docs/API_GUIDE.md) - API接口文档
- [安全性分析](docs/SECURITY.md) - 详细安全机制说明
- [使用指南](docs/USAGE_GUIDE.md) - 使用说明
- [测试指南](docs/TESTING_GUIDE.md) - 测试相关
- [GLM API配置](docs/GLM_API_KEY_SETUP.md) - API密钥配置
- [知识库使用](docs/KNOWLEDGE_BASE_GUIDE.md) - 知识库功能
- [两阶段查询](docs/Two_PHASE_SCHEMA.md) - 大型数据库优化
- [开发计划](docs/PLAN.md) - 历史开发计划
- [项目总结](docs/PROJECT_SUMMARY.md) - 开发总结

## 🛠️ 技术栈

- **语言**：Go 1.21+
- **数据库**：MySQL 8.0
- **ORM**：GORM
- **LLM**：tmc/langchaingo + GLM4.7
- **CLI**：Cobra
- **测试**：testing + testify
- **配置**：Viper + YAML

## 🤝 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

**注意**：所有PR必须通过测试，并且保持85%+的测试覆盖率。

## 📝 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 👥 作者

- Channelwill - 初始工作

## 🙏 致谢

- [GORM](https://gorm.io/) - 优秀的Go ORM库
- [langchaingo](https://github.com/tmc/langchaingo) - Go版本的LangChain
- [智谱AI](https://open.bigmodel.cn/) - 提供GLM4.7模型支持

---

**🔒 安全提示**：本项目已通过安全审查，LLM不会直接访问数据库业务数据，且有严格的SQL防火墙防止不可逆操作。详见[安全性分析文档](docs/SECURITY.md)。
