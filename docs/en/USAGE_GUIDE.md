# NLQ使用指南

## 🎉 快速开始

### 无需API Key（简化模式）

```bash
# 编译项目
make build

# 直接使用（自动使用简化SQL生成）
./bin/nlq query "boom_user有多少条数据？"
./bin/nlq query "显示所有客户"
./bin/nlq sql "SELECT * FROM boom_user LIMIT 5"
```

### 使用GLM4.7 API Key（完整功能）

```bash
# 设置API Key环境变量
export GLM_API_KEY="your-api-key-here"

# 编辑配置文件
vim config/config.yaml
# 将 api_key: ${GLM_API_KEY} 改为你的真实API Key

# 使用配置文件运行
./bin/nlq -c config/config.yaml query "去年销售额最高的员工是谁？"
```

---

## 📋 可用命令

### 1. query命令 - 自然语言查询

```bash
# 简化模式（无需API Key）
./bin/nlq query "boom_user有多少条数据？"
./bin/nlq query "显示所有客户"

# 使用GLM4.7（需要API Key）
export GLM_API_KEY="your-api-key-here"
./bin/nlq query "查询下单金额最高的前10个用户"
./bin/nlq query "每个城市的订单数量统计"
./bin/nlq query "最近7天的销售趋势"
```

### 2. sql命令 - 直接SQL查询

```bash
# 简单查询
./bin/nlq sql "SELECT COUNT(*) FROM boom_user"

# 复杂查询
./bin/nlq sql "SELECT u.name, COUNT(*) as order_count FROM boom_user u JOIN boom_order_paid_water o ON u.id = o.customer_id GROUP BY u.id ORDER BY order_count DESC LIMIT 10"

# JSON输出
./bin/nlq sql "SELECT * FROM boom_user LIMIT 5" --json
```

### 3. schema命令 - 查看数据库结构

```bash
# 查看所有表
./bin/nlq schema

# 查看特定表
./bin/nlq schema boom_user
./bin/nlq schema boom_customer
./bin/nlq schema boom_order_paid_water
```

---

## 🔧 配置说明

### 配置文件结构

```yaml
database:          # 数据库配置
  driver: mysql
  host: localhost
  port: 3306
  database: loloyal
  username: root
  password: root
  readonly: true

llm:               # LLM配置（GLM4.7）
  provider: zhipuai
  model: glm-4-plus
  api_key: ${GLM_API_KEY}  # 从环境变量读取
  base_url: https://open.bigmodel.cn/api/paas/v4/
  timeout: 30s
  temperature: 0.1

security:          # 安全配置
  mode: strict
  check_comments: true
  check_semicolon: true
```

### 环境变量

```bash
# GLM API Key
export GLM_API_KEY="your-api-key-here"

# 数据库配置（可选，会覆盖配置文件）
export DATABASE_HOST="localhost"
export DATABASE_PORT="3306"
export DATABASE_NAME="loloyal"
```

---

## 🎯 功能对比

### 简化模式 vs GLM4.7模式

| 功能 | 简化模式 | GLM4.7模式 |
|------|---------|-----------|
| **支持表** | 预定义表名 | 任意表 |
| **问题类型** | 简单关键词 | 复杂自然语言 |
| **SQL复杂度** | 简单查询 | 复杂查询（JOIN、子查询） |
| **准确性** | 模式匹配 | 语义理解 |
| **API Key** | 不需要 | 需要 |
| **成本** | 免费 | 按API调用收费 |

### 使用建议

**日常开发测试** → 简化模式
```bash
./bin/nlq query "boom_user有多少条数据？"
```

**生产环境** → GLM4.7模式
```bash
export GLM_API_KEY="your-api-key-here"
./bin/nlq query "去年销售额最高的员工是谁？"
```

---

## 💡 常见使用场景

### 1. 数据探索

```bash
# 查看所有表
./bin/nlq schema

# 查看表结构
./bin/nlq schema boom_user

# 统计记录数
./bin/nlq query "boom_user有多少条数据？"
./bin/nlq query "boom_customer有多少条数据？"
```

### 2. 数据查询

```bash
# 简单查询
./bin/nlq query "显示前5个用户"
./bin/nlq query "查询所有订单"

# 复杂查询（使用GLM4.7）
./bin/nlq query "查询下单金额最高的前10个用户"
./bin/nlq query "每个城市的用户数量统计"
```

### 3. 数据分析

```bash
# 统计分析
./bin/nlq query "订单金额的平均值"
./bin/nlq query "用户的积分分布"

# 趋势分析（使用GLM4.7）
./bin/nlq query "最近30天的订单趋势"
./bin/nlq query "用户的活跃度变化"
```

---

## 🚨 常见问题

### Q1: 如何切换使用模式？

**A:** 程序会自动检测：

1. 如果配置了有效的API Key → 使用GLM4.7
2. 如果没有配置API Key → 使用简化模式

**手动控制**：
```bash
# 强制使用简化模式
./bin/nlq query "问题"  # 不设置API Key

# 使用GLM4.7模式
export GLM_API_KEY="your-api-key"
./bin/nlq query "问题"  # 会自动使用GLM4.7
```

### Q2: API调用失败怎么办？

**A:** 程序会自动降级到简化模式：

```bash
# 即使配置了API Key，如果调用失败也会降级
./bin/nlq query "boom_user有多少条数据？"
# 如果GLM4.7调用失败，会自动使用简化模式
```

### Q3: 支持哪些表？

**简化模式**：
- boom_user
- boom_customer
- boom_order_paid_water
- boom_product
- boom_member
- 其他带boom_前缀的表

**GLM4.7模式**：
- 数据库中的所有表
- 支持任意表名

---

## 📊 性能优化建议

### 1. 批量查询

```bash
# 避免频繁的小查询，使用一次查询获取所有需要的数据
./bin/nlq sql "SELECT u.*, COUNT(o.id) as order_count FROM boom_user u LEFT JOIN boom_order_paid_water o ON u.id = o.customer_id GROUP BY u.id"
```

### 2. 使用LIMIT

```bash
# 限制结果数量，提高响应速度
./bin/nlq query "显示前10个用户"
./bin/nlq sql "SELECT * FROM boom_user LIMIT 10"
```

### 3. 缓存常用查询

```bash
# 将常用查询保存为脚本
echo "SELECT COUNT(*) FROM boom_user" > user_count.sql
./bin/nlq sql "$(< user_count.sql)"
```

---

## 🎓 进阶使用

### 1. 编写Shell脚本

```bash
#!/bin/bash
# daily_report.sh - 日报查询脚本

echo "=== 每日数据报告 ==="
echo "用户总数: $(./bin/nlq query "boom_user有多少条数据？" --json | jq -r '.count')"
echo "客户总数: $(./bin/nlq query "boom_customer有多少条数据？" --json | jq -r '.count')"
echo "订单总数: $(./bin/nlq query "boom_order_paid_water有多少条数据？" --json | jq -r '.count')"
```

### 2. 集成到其他工具

```bash
# 结合jq工具处理JSON结果
./bin/nlq query "boom_user有多少条数据？" --json | jq '.duration_ms'

# 结合grep过滤结果
./bin/nlq schema | grep "boom_customer"

# 结合wc统计
./bin/nlq schema | wc -l  # 统计表数量
```

### 3. 自动化工作流

```bash
# 定时任务查询
# 添加到crontab: 0 9 * * * /path/to/nlq query "今日订单统计" > daily_report.txt
```

---

**现在你可以选择：**

1. **直接使用简化模式**（无需API Key，立即可用）
2. **配置GLM4.7 API Key**（完整功能，需要注册智谱AI）

**无论哪种模式，都可以安全地查询数据库！** 🔒✨
