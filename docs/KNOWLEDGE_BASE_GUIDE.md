# NLQ知识库功能使用指南

## 📚 什么是知识库功能？

知识库功能允许您在查询时提供业务规则和字段说明文档，LLM会自动学习这些知识，生成更准确的SQL查询。

### 使用场景

- **业务规则复杂**：VIP用户定义、折扣规则、状态字段含义
- **字段命名不直观**：status=1/0的具体含义
- **表关系复杂**：多表关联、外键关系
- **行业特定术语**：业务领域的专业词汇

---

## 🎯 使用方法

### 1. 创建知识库文件夹

```bash
mkdir knowledge
cd knowledge
```

### 2. 编写MD文档

创建业务规则文档 `business_rules.md`：
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

创建字段说明文档 `field_explanations.md`：
```markdown
# 字段说明

## boom_user 表
- id: 用户唯一标识
- level: 用户等级（C=VIP, B=普通, A=新客户）
- status: 用户状态（1=活跃, 0=非活跃）
```

### 3. 使用知识库查询

```bash
# 指定知识库文件夹
./bin/nlq query "查询VIP用户" --knowledge ./knowledge

# 简写形式
./bin/nlq query "查询活跃用户" -k ./knowledge

# 结合详细输出
./bin/nlq query "查询VIP用户" --knowledge ./knowledge -v
```

---

## 📊 效果对比

### 无知识库

```bash
./bin/nlq query "查询VIP用户"
```

**生成的SQL**：
```sql
SELECT * FROM boom_customer WHERE member_state = 1
```

**问题**：
- ❌ 猜测错误：member_state = 1
- ❌ 表名错误：boom_customer（应该是boom_user）
- ❌ 缺少条件：没有考虑status和is_delete字段

### 有知识库

```bash
./bin/nlq query "查询VIP用户" --knowledge ./knowledge
```

**生成的SQL**：
```sql
SELECT * FROM boom_user
WHERE level = 'C' AND status = 1 AND is_delete = 0
```

**优势**：
- ✅ 精确定义：level = 'C'
- ✅ 正确表名：boom_user
- ✅ 完整条件：考虑了status和is_delete字段
- ✅ 符合业务规则：完全按照知识库中的定义

---

## 🗂️ 知识库最佳实践

### 文档组织

```
knowledge/
├── business_rules.md      # 业务规则
├── field_explanations.md  # 字段说明
├── table_relations.md     # 表关系说明
└── examples/              # 示例查询
    ├── common_queries.md
    └── complex_joins.md
```

### 文档编写技巧

1. **使用清晰的标题结构**
   ```markdown
   # VIP用户规则
   ## 定义
   ## 查询条件
   ```

2. **提供具体示例**
   ```markdown
   ## VIP用户查询示例
   ```sql
   SELECT * FROM boom_user WHERE level = 'C'
   ```
   ```

3. **字段值映射**
   ```markdown
   - level = "C": VIP客户
   - level = "B": 普通客户
   - level = "A": 新客户
   ```

4. **常见查询模式**
   ```markdown
   ## 常用查询
   - 查询活跃用户: status = 1 AND is_delete = 0
   - 查询最近创建: ORDER BY created_at DESC
   ```

---

## 🔧 高级功能

### 1. 多个知识库文档

系统会自动加载文件夹中的所有MD文档：

```bash
knowledge/
├── business_rules.md      # 业务规则
├── field_explanations.md  # 字段说明
└── vip_rules.md          # VIP特定规则
```

### 2. 递归加载子文件夹

系统会递归加载所有子文件夹中的MD文档：

```bash
knowledge/
├── business/
│   ├── vip.md
│   └── discount.md
└── technical/
    ├── fields.md
    └── indexes.md
```

### 3. 结合配置文件

在 `config/config.yaml` 中配置默认知识库：

```yaml
knowledge:
  base_path: ./knowledge
  auto_load: true
```

### 4. 详细输出模式

查看加载的知识库文档：

```bash
./bin/nlq query "查询VIP用户" --knowledge ./knowledge -v
```

**输出**：
```
🤖 使用GLM4.7 LLM: glm-4-plus
📚 已加载 2 个知识库文档:
   - 业务规则
   - 字段说明
```

---

## 🎨 实际应用场景

### 场景1：电商查询

**业务规则**：
```markdown
# 电商业务规则

## 订单状态
- status = "completed": 已完成
- status = "pending": 处理中
- status = "cancelled": 已取消

## VIP折扣
- VIP客户：20%折扣
- 普通客户：无折扣
```

**查询**：
```bash
./bin/nlq query "查询已完成订单的总金额" --knowledge ./knowledge
```

### 场景2：用户管理

**字段说明**：
```markdown
# 用户字段说明

## 用户类型
- user_type = "premium": 付费用户
- user_type = "free": 免费用户

## 账户状态
- account_status = 1: 正常
- account_status = 0: 冻结
```

**查询**：
```bash
./bin/nlq query "查询付费用户中账户正常的用户" --knowledge ./knowledge
```

### 场景3：数据分析

**分析规则**：
```markdown
# 数据分析规则

## 活跃用户定义
- 最近30天有登录记录
- status = 1
- is_delete = 0

## 高价值客户
- 累计消费 > 10000元
- 购买次数 > 10次
```

**查询**：
```bash
./bin/nlq query "查询高价值客户" --knowledge ./knowledge
```

---

## 💡 提示和技巧

### 1. 知识库内容质量

- ✅ **明确具体**：避免模糊描述
- ✅ **提供示例**：包含SQL示例
- ✅ **保持更新**：及时更新业务规则变更
- ❌ **避免冗余**：不要重复相同信息

### 2. 文档命名

- ✅ 使用描述性名称：`vip_rules.md`
- ✅ 使用下划线分隔：`field_explanations.md`
- ❌ 避免特殊字符：`业务规则.md`（可能有问题）

### 3. 内容结构

```markdown
# 主标题（文档标题）

## 子标题1
内容说明

## 子标题2
- 要点1
- 要点2

### 三级标题
详细说明
```

---

## 🚀 故障排查

### 问题1：知识库未加载

**症状**：
```bash
./bin/nlq query "查询" --knowledge ./knowledge
# 没有显示"已加载X个知识库文档"
```

**解决方案**：
1. 检查文件夹路径是否正确
2. 确认文件夹中有.md文件
3. 检查文件权限

### 问题2：SQL生成不准确

**症状**：即使有知识库，SQL仍然不准确

**解决方案**：
1. 检查知识库内容是否清晰
2. 添加更多示例和说明
3. 使用详细输出查看加载的文档

### 问题3：加载文档过多

**症状**：
```bash
📚 已加载 50 个知识库文档
```

**解决方案**：
1. 精简知识库内容
2. 合并相关文档
3. 使用子文件夹组织

---

## 📈 性能考虑

### Token使用

知识库内容会消耗LLM的Token配额：

- **小文档**（<1000字）：几乎无影响
- **中等文档**（1000-5000字）：轻微影响
- **大文档**（>5000字）：建议拆分

### 优化建议

1. **精简内容**：只包含必要信息
2. **分文件管理**：按主题拆分
3. **使用索引**：在主文档中引用其他文档

---

## 🎓 总结

知识库功能是NLQ的强大特性，它能够：

- ✅ **提高准确性**：准确理解业务规则
- ✅ **减少错误**：避免字段理解错误
- ✅ **增强可维护性**：业务规则集中管理
- ✅ **支持复杂查询**：处理复杂业务逻辑

**最佳实践**：
1. 从简单业务规则开始
2. 逐步完善知识库内容
3. 定期更新维护
4. 结合实际查询效果优化

---

**享受更智能的NLQ查询体验！** ✨
