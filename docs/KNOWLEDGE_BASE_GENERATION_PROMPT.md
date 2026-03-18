# NLQ 知识库自动生成提示词

> 专门为 Claude Code 和其他 AI Agent 设计的提示词文档
> 用于分析 NLQ 项目代码库并生成高质量的知识库文档

---

## 📋 任务目标

你的任务是通过深入分析 NLQ 项目的以下内容，生成高质量的知识库文档：

1. **数据库模型结构**：表定义、字段说明、关系映射
2. **代码内联逻辑**：业务规则、查询模式、数据流转
3. **模型关联关系**：外键、索引、JOIN 模式
4. **业务领域知识**：行业术语、业务概念、数据含义

生成的知识库文档将被 NLQ 系统的 LLM 使用，以提高自然语言到 SQL 转换的准确性。

---

## 🎯 分析步骤

### 第一步：数据库模型分析

**目标**: 理解数据结构和表关系

**分析内容**:
```go
// 检查以下文件和目录：
internal/database/schema.go      // Schema 定义
internal/database/migrations/    // 数据库迁移文件
internal/database/models/        // 数据模型定义
```

**需要提取的信息**:
- 表名和字段名
- 字段类型和约束
- 主键和外键关系
- 索引定义
- 字段默认值
- 枚举值含义

**输出示例**:
```markdown
## boom_user 表结构

| 字段名 | 类型 | 说明 | 约束 |
|--------|------|------|------|
| id | bigint | 用户唯一标识 | PRIMARY KEY |
| shop_name | varchar(255) | 商户名称 | NOT NULL |
| level | varchar(10) | 用户等级 | DEFAULT 'A' |
| status | tinyint | 用户状态 | DEFAULT 1 |
| is_delete | tinyint | 删除标记 | DEFAULT 0 |
| created_at | datetime | 创建时间 | NOT NULL |

### 字段值说明
- level: "A"=新客户, "B"=普通客户, "C"=VIP客户
- status: 1=活跃, 0=非活跃
- is_delete: 1=已删除, 0=正常

### 关系
- 一对多关联 boom_order_paid_water (id -> user_id)
```

### 第二步：代码逻辑分析

**目标**: 理解业务规则和查询模式

**分析内容**:
```go
// 检查以下文件：
internal/handler/*.go           // 查询处理逻辑
internal/server/handlers.go     // HTTP 处理器
pkg/security/firewall.go        // SQL 防火墙规则
internal/llm/prompts.go         // 现有 Prompt 模板
```

**需要提取的信息**:
- 常见查询模式
- 安全规则（SQL 防火墙）
- 权限控制逻辑
- 数据过滤条件
- 统计和聚合规则

**输出示例**:
```markdown
## 查询模式

### 用户查询
- 基础查询: `SELECT * FROM boom_user WHERE is_delete = 0`
- 活跃用户: `WHERE status = 1 AND is_delete = 0`
- VIP 用户: `WHERE level = 'C' AND status = 1 AND is_delete = 0`

### 订单查询
- 订单统计: `COUNT(*) as total, SUM(amount) as amount`
- 最近订单: `ORDER BY created_at DESC LIMIT 10`
- 订单筛选: `WHERE status = 'completed'`

### JOIN 模式
- 用户-订单: `boom_user JOIN boom_order_paid_water ON boom_user.id = boom_order_paid_water.user_id`
- 客户-用户: `boom_customer JOIN boom_user ON boom_customer.id = boom_user.customer_id`
```

### 第三步：业务规则提取

**目标**: 理解业务逻辑和领域知识

**分析内容**:
```go
// 检查以下位置：
- SQL 查询中的 WHERE 条件
- 数据验证逻辑
- 业务计算公式
- 状态转换规则
- 配置文件中的业务参数
```

**需要提取的信息**:
- 业务实体定义（如 VIP 用户）
- 状态值含义
- 计算公式
- 时间范围规则
- 数据权限规则

**输出示例**:
```markdown
## 业务规则

### VIP 用户定义
- **条件**: `level = 'C' AND status = 1 AND is_delete = 0`
- **权益**: 享受 20% 折扣
- **识别**: 通过 level 字段区分

### 用户状态规则
- **活跃用户**: status = 1 AND is_delete = 0
- **非活跃用户**: status = 0 AND is_delete = 0
- **已删除用户**: is_delete = 1

### 订单状态
- **已完成**: status = 'completed'
- **处理中**: status = 'pending'
- **已取消**: status = 'cancelled'

### 时间相关查询
- **最新数据**: ORDER BY created_at DESC
- **时间范围**: created_at BETWEEN '开始时间' AND '结束时间'
- **最近 N 天**: DATE_SUB(NOW(), INTERVAL N DAY)
```

---

## 📝 知识库文档格式

### 标准文档结构

```markdown
# [文档标题]

> 文档描述和用途说明

---

## [业务领域/表名]

### 定义说明
[业务概念或表的详细说明]

### 字段说明
| 字段 | 类型 | 说明 | 取值 |
|------|------|------|------|
| ... | ... | ... | ... |

### 业务规则
- 规则1: [详细说明]
- 规则2: [详细说明]

### 查询示例
\`\`\`sql
[标准查询语句]
\`\`\`

### 注意事项
- ⚠️ [重要提示]
- ℹ️ [补充说明]
```

### 推荐的文档分类

```
knowledge/
├── 01_table_schemas.md        # 表结构说明
├── 02_business_rules.md       # 业务规则
├── 03_field_definitions.md    # 字段定义
├── 04_query_patterns.md       # 查询模式
├── 05_relationships.md        # 表关系
└── 06_security_rules.md       # 安全规则
```

---

## 🎨 生成示例

### 示例 1: 表结构文档

**输入**: 分析 `internal/database/schema.go`

**输出** (`knowledge/01_table_schemas.md`):
```markdown
# 数据库表结构说明

本文档包含 NLQ 系统中所有数据库表的结构说明。

---

## boom_user (用户表)

### 表说明
存储系统中的所有用户信息，包括商户、客户等。

### 字段结构
| 字段名 | 类型 | 说明 | 约束 | 默认值 |
|--------|------|------|------|--------|
| id | bigint | 用户唯一标识 | PRIMARY KEY | AUTO_INCREMENT |
| shop_name | varchar(255) | 商户名称 | NOT NULL | - |
| level | varchar(10) | 用户等级 | - | 'A' |
| status | tinyint | 用户状态 | - | 1 |
| is_delete | tinyint | 删除标记 | - | 0 |
| created_at | datetime | 创建时间 | NOT NULL | CURRENT_TIMESTAMP |
| updated_at | datetime | 更新时间 | - | NULL |

### 字段值说明
- **level (用户等级)**:
  - `A`: 新客户
  - `B`: 普通客户
  - `C`: VIP 客户

- **status (状态)**:
  - `1`: 活跃
  - `0`: 非活跃

- **is_delete (删除标记)**:
  - `0`: 正常
  - `1`: 已删除

### 关系
- **一对多**: `id` → `boom_order_paid_water.user_id`

### 索引
- PRIMARY KEY (`id`)
- INDEX `idx_shop_name` (`shop_name`)
- INDEX `idx_level` (`level`)
- INDEX `idx_status` (`status`)

---

## boom_customer (客户表)

### 表说明
存储客户基本信息。

### 字段结构
| 字段名 | 类型 | 说明 | 约束 | 默认值 |
|--------|------|------|------|--------|
| id | bigint | 客户ID | PRIMARY KEY | AUTO_INCREMENT |
| member_state | tinyint | 会员状态 | - | 0 |
| ... | ... | ... | ... | ... |

[继续其他表...]
```

### 示例 2: 业务规则文档

**输入**: 分析代码中的业务逻辑

**输出** (`knowledge/02_business_rules.md`):
```markdown
# 业务规则说明

本文档包含 NLQ 系统的核心业务规则和查询约定。

---

## 用户相关规则

### VIP 用户定义
**定义条件**:
```sql
WHERE level = 'C' AND status = 1 AND is_delete = 0
```

**业务说明**:
- `level = 'C'` 表示 VIP 客户等级
- 必须是活跃状态 (`status = 1`)
- 未被删除 (`is_delete = 0`)

**常见查询**:
```sql
-- 查询所有 VIP 用户
SELECT * FROM boom_user
WHERE level = 'C' AND status = 1 AND is_delete = 0;

-- 统计 VIP 用户数量
SELECT COUNT(*) as vip_count
FROM boom_user
WHERE level = 'C' AND status = 1 AND is_delete = 0;
```

---

## 订单相关规则

### 订单状态说明
| 状态值 | 说明 | 查询条件 |
|--------|------|----------|
| completed | 已完成 | status = 'completed' |
| pending | 处理中 | status = 'pending' |
| cancelled | 已取消 | status = 'cancelled' |

### �金额统计
```sql
-- 订单总金额
SELECT SUM(amount) as total_amount
FROM boom_order_paid_water
WHERE status = 'completed';

-- 按用户统计订单金额
SELECT user_id, SUM(amount) as total_amount
FROM boom_order_paid_water
GROUP BY user_id;
```

---

## 查询约定

### 时间相关
- **最新数据**: `ORDER BY created_at DESC`
- **最早数据**: `ORDER BY created_at ASC`
- **最近 N 天**: `WHERE created_at >= DATE_SUB(NOW(), INTERVAL N DAY)`

### 数据过滤
- **未删除**: `is_delete = 0`
- **活跃用户**: `status = 1 AND is_delete = 0`
- **分页查询**: `LIMIT N`

### JOIN 模式
```sql
-- 用户关联订单
SELECT u.*, o.*
FROM boom_user u
LEFT JOIN boom_order_paid_water o ON u.id = o.user_id
WHERE u.is_delete = 0;

-- 客户关联用户
SELECT c.*, u.*
FROM boom_customer c
LEFT JOIN boom_user u ON c.id = u.customer_id
WHERE c.is_delete = 0;
```

---

## 数据安全规则

### 只读限制
所有查询都是只读的，不允许以下操作：
- ❌ INSERT / UPDATE / DELETE
- ❌ DROP / TRUNCATE
- ❌ CREATE / ALTER

### 允许的操作
- ✅ SELECT (仅查询)
- ✅ JOIN (表关联)
- ✅ GROUP BY (分组)
- ✅ ORDER BY (排序)
- ✅ LIMIT (限制行数)
```

### 示例 3: 查询模式文档

**输入**: 分析常见查询模式

**输出** (`knowledge/04_query_patterns.md`):
```markdown
# 常用查询模式

本文档包含 NLQ 系统中最常用的查询模式和示例。

---

## 计数查询

### 用户计数
```sql
-- 总用户数
SELECT COUNT(*) as total FROM boom_user WHERE is_delete = 0;

-- VIP 用户数
SELECT COUNT(*) as vip_count
FROM boom_user
WHERE level = 'C' AND status = 1 AND is_delete = 0;

-- 按等级统计用户
SELECT level, COUNT(*) as count
FROM boom_user
WHERE is_delete = 0
GROUP BY level;
```

### 订单计数
```sql
-- 总订单数
SELECT COUNT(*) as total_orders FROM boom_order_paid_water;

-- 按状态统计订单
SELECT status, COUNT(*) as count
FROM boom_order_paid_water
GROUP BY status;
```

---

## 排序查询

### 最新记录
```sql
-- 最新用户
SELECT * FROM boom_user
WHERE is_delete = 0
ORDER BY created_at DESC
LIMIT 10;

-- 最新订单
SELECT * FROM boom_order_paid_water
ORDER BY created_at DESC
LIMIT 10;
```

### 最早记录
```sql
-- 最早注册的用户
SELECT * FROM boom_user
WHERE is_delete = 0
ORDER BY created_at ASC
LIMIT 100;
```

---

## 聚合查询

### 金额统计
```sql
-- 总金额
SELECT SUM(amount) as total_amount
FROM boom_order_paid_water
WHERE status = 'completed';

-- 平均订单金额
SELECT AVG(amount) as avg_amount
FROM boom_order_paid_water
WHERE status = 'completed';

-- 按用户统计金额
SELECT user_id, SUM(amount) as total, COUNT(*) as order_count
FROM boom_order_paid_water
WHERE status = 'completed'
GROUP BY user_id
ORDER BY total DESC;
```

---

## 多表关联

### 用户-订单关联
```sql
-- 查询用户及其订单
SELECT
    u.id as user_id,
    u.shop_name,
    o.id as order_id,
    o.amount
FROM boom_user u
LEFT JOIN boom_order_paid_water o ON u.id = o.user_id
WHERE u.is_delete = 0
ORDER BY u.created_at DESC;
```

### 客户-用户-订单关联
```sql
-- 查询客户的用户订单信息
SELECT
    c.id as customer_id,
    u.shop_name,
    COUNT(o.id) as order_count
FROM boom_customer c
LEFT JOIN boom_user u ON c.id = u.customer_id
LEFT JOIN boom_order_paid_water o ON u.id = o.user_id
WHERE c.is_delete = 0
GROUP BY c.id, u.id;
```

---

## 条件查询模式

### 多条件组合
```sql
-- VIP 且活跃用户
SELECT * FROM boom_user
WHERE level = 'C' AND status = 1 AND is_delete = 0;

-- 时间范围查询
SELECT * FROM boom_order_paid_water
WHERE created_at >= '2024-01-01'
  AND created_at <= '2024-12-31'
  AND status = 'completed';

-- IN 查询
SELECT * FROM boom_user
WHERE level IN ('A', 'B', 'C')
AND is_delete = 0;
```

---

## 性能优化建议

### 使用索引字段
- 优先使用 `id`, `created_at`, `level`, `status` 等索引字段作为查询条件

### 避免 SELECT *
- 明确指定需要的字段，减少数据传输

### 合理使用 LIMIT
- 对于可能返回大量数据的查询，使用 LIMIT 限制结果集

### 分页查询
```sql
-- 第一页
SELECT * FROM boom_user
WHERE is_delete = 0
ORDER BY created_at DESC
LIMIT 20 OFFSET 0;

-- 第二页
SELECT * FROM boom_user
WHERE is_delete = 0
ORDER BY created_at DESC
LIMIT 20 OFFSET 20;
```
```

---

## ✅ 质量检查清单

生成知识库文档后，请检查以下质量标准：

### 内容完整性
- [ ] 所有表都有详细的结构说明
- [ ] 所有枚举字段都有值说明
- [ ] 所有关系都有明确的 JOIN 示例
- [ ] 所有关键业务规则都有文档说明

### 格式规范性
- [ ] 使用标准的 Markdown 格式
- [ ] 表格对齐整齐
- [ ] 代码块使用正确的语言标识
- [ ] 标题层级清晰合理

### 示例准确性
- [ ] SQL 示例可以执行（语法正确）
- [ ] 字段名与实际表结构一致
- [ ] 业务规则描述准确无误
- [ ] 查询结果符合业务逻辑

### 可读性
- [ ] 文档结构清晰，易于导航
- [ ] 描述简洁明了，避免冗余
- [ ] 使用列表和表格提高可读性
- [ ] 重要信息有适当的标注

---

## 🚀 执行流程

1. **代码分析阶段**
   ```
   - 读取数据库模型定义
   - 分析业务逻辑代码
   - 提取查询模式
   - 识别业务规则
   ```

2. **文档生成阶段**
   ```
   - 生成表结构文档
   - 生成业务规则文档
   - 生成查询模式文档
   - 生成字段说明文档
   ```

3. **质量验证阶段**
   ```
   - 检查完整性
   - 验证准确性
   - 优化可读性
   - 补充缺失内容
   ```

4. **输出整合阶段**
   ```
   - 组织文档结构
   - 添加交叉引用
   - 生成索引文档
   - 输出到 knowledge/ 目录
   ```

---

## 📦 输出位置

将生成的知识库文档保存到以下位置：

```
NLQ/
└── knowledge/
    ├── 01_table_schemas.md       # 表结构说明
    ├── 02_business_rules.md      # 业务规则
    ├── 03_field_definitions.md   # 字段定义
    ├── 04_query_patterns.md      # 查询模式
    ├── 05_relationships.md       # 表关系
    ├── 06_security_rules.md      # 安全规则
    └── README.md                 # 知识库索引
```

---

## 💡 注意事项

### 数据敏感性
- ⚠️ 不要在知识库中包含真实的敏感数据
- ⚠️ 使用占位符或示例数据
- ⚠️ 避免暴露用户隐私信息

### 版本控制
- 📝 知识库文档应该纳入版本控制
- 📝 随着代码更新同步更新知识库
- 📝 记录文档的变更历史

### 维护性
- 🔧 定期审查和更新知识库内容
- 🔧 根据实际查询效果优化文档
- 🔧 收集用户反馈持续改进

---

## 🎓 总结

通过遵循本提示词文档，AI Agent 可以生成高质量、结构化的 NLQ 知识库文档，显著提高自然语言到 SQL 转换的准确性和可靠性。

**关键成功因素**:
1. ✅ 深入分析代码库，理解业务逻辑
2. ✅ 遵循标准格式，保持文档一致性
3. ✅ 提供准确示例，确保可执行性
4. ✅ 持续优化改进，跟上业务变化

---

*本提示词文档由 NLQ 项目团队维护*
*最后更新: 2026-03-18*
