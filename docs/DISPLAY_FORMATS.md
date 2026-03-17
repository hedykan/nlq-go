# NLQ结果显示美化指南

## 📋 显示模式

NLQ提供了多种结果显示模式，根据不同场景选择最适合的显示方式。

---

## 🎯 默认模式（智能列选择）

默认情况下，NLQ会智能选择重要的列显示，最多显示8列：

```bash
./bin/nlq query "查询最近创建的5个用户"
```

**显示效果**：
```
💡 显示 8 列（共 58 列，使用 --wide 查看所有列，或 --columns 指定列）
┌──────────┬────────────────────┬────────────┬────────────────────────────┬──────────┬──────────┬─────────────┬────────────┐
│ id       │ name               │ username   │ email                      │ phone    │ status   │ created_at  │ updated_at │
├──────────┼────────────────────┼────────────┼────────────────────────────┼──────────┼──────────┼─────────────┼────────────┤
│ 102019   │ ctest-2525         │ 彭 红洁       │ penghongjie@channelwill.cn │          │ 1        │ 73620160534 │ 1770096449 │
│ 102093   │                    │            │                            │          │ 0        │ 1771920461  │ 1771920461 │
└──────────┴────────────────────┴────────────┴────────────────────────────┴──────────┴──────────┴─────────────┴────────────┘
```

**智能列优先级**（按优先级排序）：
- ID字段：`id`, `user_id`, `customer_id`, `order_id`
- 基本信息：`name`, `username`, `email`, `phone`
- 状态字段：`status`, `state`, `type`, `category`
- 金额数量：`amount`, `price`, `total`, `count`, `quantity`
- 时间字段：`created_at`, `updated_at`, `date`, `time`
- 地理位置：`country`, `city`, `address`

---

## 📊 Wide模式（显示所有列）

使用 `--wide` 或 `-w` 参数显示所有列：

```bash
./bin/nlq query "查询用户信息" --wide
```

**适用场景**：
- 需要查看完整数据结构
- 调试和数据分析
- 导出完整数据

---

## 🎯 自定义列（--columns）

使用 `--columns` 参数指定要显示的列：

```bash
# 基本用法
./bin/nlq query "查询用户" --columns "id,name,email"

# 多个列
./bin/nlq query "查询订单" --columns "id,customer_id,amount,status,created_at"

# 列名不区分大小写
./bin/nlq query "查询用户" --columns "ID,NAME,Email"
```

**适用场景**：
- 只关注特定字段
- 简化输出用于报告
- 提高可读性

---

## 📝 紧凑模式（--compact）

使用 `--compact` 参数使用简洁的表格格式：

```bash
./bin/nlq query "查询用户" --compact
```

**显示效果**：
```
id | name | username | email | phone | status | created_at | updated_at
---+------+----------+-------+-------+--------+------------+-----------
102019 | ctest-2525 | 彭 红洁 | penghongjie@channelwill.cn |  | 1 | 73620160534 | 1770096449
102093 |  |  |  |  | 0 | 1771920461 | 1771920461
```

**适用场景**：
- 复制粘贴到其他工具
- 日志记录
- 脚本处理

---

## 📄 JSON格式输出

使用 `--json` 或 `-j` 参数输出JSON格式：

```bash
./bin/nlq query "查询用户数量" --json
```

**输出格式**：
```json
{"question":"查询用户数量","sql":"SELECT COUNT(*) FROM boom_user","count":102093,"duration_ms":523}
```

**适用场景**：
- API集成
- 自动化脚本
- 数据处理pipeline

---

## 🔄 组合使用

可以组合多个参数：

```bash
# 指定列 + 紧凑模式
./bin/nlq query "查询用户" --columns "id,name,email" --compact

# Wide模式 + JSON
./bin/nlq query "查询用户" --wide --json

# 详细输出 + 自定义列
./bin/nlq -v query "查询用户" --columns "id,name" --compact
```

---

## 💡 使用建议

### 日常查询
```bash
# 默认模式，智能列选择
./bin/nlq query "查询最近创建的5个用户"
```

### 数据分析
```bash
# 显示所有列，完整数据
./bin/nlq query "查询订单详情" --wide
```

### 报告生成
```bash
# 指定重要列，紧凑格式
./bin/nlq query "查询销售数据" --columns "id,amount,date" --compact
```

### API集成
```bash
# JSON格式输出
./bin/nlq query "查询用户数量" --json
```

### 脚本处理
```bash
# 紧凑模式，便于解析
./bin/nlq query "查询用户" --compact | grep "active"
```

---

## 🎨 显示特性

### 自动列宽调整
表格列宽会根据内容自动调整，确保数据显示完整且美观。

### 智能值格式化
- **NULL值**：显示为 `NULL`
- **长字符串**：自动截断并显示 `...`
- **时间戳**：保持原始格式

### 行数限制
- 默认最多显示10行数据
- 超过限制会显示 `... 还有 N 行`

### 表格美化
- 使用Unicode绘制框线
- 列名和数据对齐
- 清晰的视觉分隔

---

## 🚀 高级技巧

### 创建常用查询别名
```bash
# 在 ~/.bashrc 或 ~/.zshrc 中添加
alias nlq-users='nlq query "查询用户" --columns "id,name,email"'
alias nlq-recent='nlq query "查询最近创建的记录" --compact'
alias nlq-wide='nlq query "查询详情" --wide'
```

### 结合其他工具
```bash
# 结合jq处理JSON
nlq query "查询用户" --json | jq '.count'

# 结合grep过滤
nlq query "查询用户" --compact | grep "active"

# 结合less分页
nlq query "查询所有用户" --wide | less

# 导出到文件
nlq query "查询用户" --compact > users.txt
```

### 创建查询脚本
```bash
#!/bin/bash
# daily_report.sh - 日报查询脚本

echo "=== 今日数据报告 ==="
echo "用户总数: $(nlq query "boom_user有多少条数据？" --json | jq -r '.count')"
echo "订单总数: $(nlq query "boom_order_paid_water有多少条数据？" --json | jq -r '.count')"
echo ""
echo "=== 最新用户 ==="
nlq query "查询最近创建的3个用户" --columns "id,name,email,created_at" --compact
```

---

**享受更美观、更高效的查询体验！** ✨
