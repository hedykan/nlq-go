# 智能查询处理器切换功能

## 概述

本小姐实现了智能查询处理器切换功能！根据数据库规模自动选择最优的查询处理器，平衡性能和精准度。 (￣▽￣)／

## 功能特性

### 三种处理模式

1. **auto（自动模式）** - 推荐！
   - 根据数据库表数量自动选择处理器
   - 小型数据库（≤20表）→ 单步法（快速）
   - 大型数据库（>20表）→ 两步法（精准）

2. **simple（单步法）**
   - 强制使用单步法处理器
   - 适合小型数据库
   - 性能：~25秒（1次LLM调用）

3. **two_phase（两步法）**
   - 强制使用两步法处理器
   - 适合大型数据库
   - 性能：~40秒（2次LLM调用）

## 配置说明

### config.yaml

```yaml
# 查询处理器配置
query:
  mode: auto  # 处理模式: "auto"(自动), "simple"(单步), "two_phase"(两步)
  table_count_threshold: 20  # 表数量阈值（自动模式下使用）
```

### 配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `mode` | string | `auto` | 处理模式：`auto`, `simple`, `two_phase` |
| `table_count_threshold` | int | `20` | 表数量阈值（仅在auto模式下有效） |

## 性能对比

| 数据库规模 | 处理器 | LLM调用次数 | 总耗时 | 适用场景 |
|-----------|--------|------------|--------|---------|
| 小型（≤20表） | 单步法 | 1次 | ~25秒 | 性能优先 |
| 大型（>20表） | 两步法 | 2次 | ~40秒 | 精准度优先 |

## 使用示例

### 示例1：自动模式（推荐）

```yaml
query:
  mode: auto
  table_count_threshold: 20
```

服务器启动时会自动检测数据库表数量：

```
📊 数据库表数量: 15
✅ 检测到小型数据库（15 ≤ 20），使用单步法以提高性能
🤖 查询处理器: 单步法（QueryHandler）
🤖 LLM模型: glm-4.7
```

### 示例2：强制单步法

```yaml
query:
  mode: simple
```

适合小型数据库或追求性能的场景：

```
📊 数据库表数量: 15
🤖 查询处理器: 单步法（QueryHandler）
🤖 LLM模型: glm-4.7
```

### 示例3：强制两步法

```yaml
query:
  mode: two_phase
```

适合大型数据库或追求精准度的场景：

```
📊 数据库表数量: 50
🤖 查询处理器: 两步法（TwoPhaseQueryHandler）
🤖 LLM模型: glm-4.7
```

## 性能优化建议

### 小型数据库优化

如果你的数据库表数量 ≤ 20，建议使用自动模式或单步法：

**性能提升：60%**
```
两步法: 40秒 (14.5s + 24.9s)
单步法: 25秒 (只有1次LLM调用)
```

### 大型数据库优化

如果你的数据库表数量 > 20，建议使用自动模式或两步法：

**精准度提升：避免token限制**
```
单步法: 可能超出LLM token限制
两步法: 精准选择相关表，避免token溢出
```

## 环境变量支持

除了配置文件，也可以通过环境变量设置：

```bash
# 设置处理模式
export NLQ_QUERY_MODE=auto

# 设置表数量阈值
export NLQ_QUERY_TABLE_COUNT_THRESHOLD=20
```

## 监控和调试

服务器启动时会显示选择的处理器类型：

```
📊 数据库表数量: 25
✅ 检测到大型数据库（25 > 20），使用两步法以保证精准度
🤖 查询处理器: 两步法（TwoPhaseQueryHandler）
🤖 LLM模型: glm-4.7
```

## 技术实现

### 智能选择逻辑

```go
func createQueryHandler(db *gorm.DB, cfg *config.Config) (handler.QueryHandlerInterface, error) {
    // 获取数据库表数量
    tables, err := getTableCount(db)

    // 根据配置模式和表数量选择处理器
    switch cfg.Query.Mode {
    case "auto":
        if tables <= cfg.Query.TableCountThreshold {
            return handler.NewQueryHandlerWithLLM(...)  // 单步法
        } else {
            return handler.NewTwoPhaseQueryHandlerWithLLM(...)  // 两步法
        }
    case "simple":
        return handler.NewQueryHandlerWithLLM(...)
    case "two_phase":
        return handler.NewTwoPhaseQueryHandlerWithLLM(...)
    }
}
```

### 表数量检测

```go
func getTableCount(db *gorm.DB) (int, error) {
    var count int64
    err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&count).Error
    return int(count), err
}
```

## 迁移指南

### 从旧版本升级

如果你之前使用的是固定两步法，现在可以：

1. **保持现状**（推荐）：
   ```yaml
   query:
     mode: two_phase
   ```

2. **启用智能切换**（推荐）：
   ```yaml
   query:
     mode: auto
     table_count_threshold: 20
   ```

3. **强制单步法**（如果数据库较小）：
   ```yaml
   query:
     mode: simple
   ```

## 故障排查

### 问题1：启动时显示"获取表数量失败"

**原因**：数据库连接失败或权限不足

**解决**：
- 检查数据库连接配置
- 确保数据库用户有`information_schema`表的查询权限

### 问题2：自动模式选择不正确

**原因**：表数量阈值设置不当

**解决**：
- 调整`table_count_threshold`值
- 或使用`simple`/`two_phase`强制指定模式

## 未来改进

- [ ] 支持动态调整阈值
- [ ] 添加性能监控和自动优化
- [ ] 支持更多处理器类型
- [ ] 添加缓存机制进一步提升性能

---

哼，这种完美的智能切换功能当然只有本小姐才能设计出来！笨蛋快去试试吧～ (￣▽￣)／

才、才不是特意帮你写的，只是看不惯那么慢的代码而已！笨蛋！(,,>﹏<,,)
