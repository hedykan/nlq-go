# NLQ项目开发规则

本文件夹包含项目的所有开发规则和最佳实践。

## 📋 规则列表

### 必读规则

1. **[TDD (测试驱动开发)](./tdd.md)** ⭐ 强制要求
   - 所有新功能开发必须遵循TDD流程
   - 测试覆盖率要求：≥70%
   - Red-Green-Refactor循环

## 🚀 快速开始

### 开发新功能前

1. **阅读TDD规则**
   ```bash
   cat .claude/rules/tdd.md
   ```

2. **理解TDD流程**
   - 🔴 Red: 先写失败的测试
   - 🟢 Green: 写代码让测试通过
   - 🔵 Refactor: 重构保持测试通过

3. **开始编码**
   ```bash
   # 1. 创建测试文件
   touch internal/yourpackage/feature_test.go

   # 2. 先写测试（TDD第一步）
   # 在feature_test.go中编写测试函数

   # 3. 运行测试（确认失败）
   go test ./internal/yourpackage -v

   # 4. 写功能代码（TDD第二步）
   # 在feature.go中实现功能

   # 5. 再次运行测试（确认通过）
   go test ./internal/yourpackage -v

   # 6. 重构优化（TDD第三步）
   # 优化代码，保持测试通过

   # 7. 检查覆盖率
   go test ./... -coverprofile=coverage.out
   go tool cover -html=coverage.out
   ```

## 📊 规则执行

### 代码审查检查点

在提交Pull Request前，确保：

- [ ] 已阅读并理解相关规则
- [ ] 所有测试通过 (`go test ./...`)
- [ ] 测试覆盖率 ≥70%
- [ ] 新功能有对应测试
- [ ] Mock使用正确
- [ ] 代码符合规范

### 违规后果

**自动检查：**
- Pre-commit hook将检查测试覆盖率
- CI/CD将自动运行所有测试

**人工审查：**
- 不符合TDD规则的PR将被拒绝
- 测试覆盖率不达标的PR将被拒绝
- 没有测试的新功能将被拒绝

## 📚 规则更新

如需添加新规则，请联系项目维护者。

---

*最后更新：2026-03-19*
*项目：NLQ (Natural Language Query)*
