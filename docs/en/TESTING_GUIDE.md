# 两阶段动态Schema选择测试指南

## 📋 测试准备

### 1. 环境检查

```bash
# 1. 确保数据库正在运行
docker ps | grep mysql

# 2. 检查数据库连接
docker exec -it mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;"

# 3. 查看表数量
docker exec -it mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" | wc -l
```

### 2. 配置API Key

```bash
# 设置环境变量
export GLM_API_KEY="your-api-key-here"

# 或者在config.yaml中配置
vim config/config.yaml
```

## 🧪 单元测试

### 测试1：运行所有单元测试

```bash
# 运行handler包的所有测试
go test -v ./internal/handler/...

# 运行database包的测试
go test -v ./internal/database/...

# 运行所有测试
go test -v ./...
```

### 测试2：两阶段特定测试

```bash
# 只运行两阶段相关测试
go test -v ./internal/handler/... -run TwoPhase

# 测试JSON提取
go test -v ./internal/handler/... -run TestExtractJSON

# 测试表选择
go test -v ./internal/handler/... -run TestTableSelector

# 测试Schema构建
go test -v ./internal/handler/... -run TestSchemaBuilder
```

### 测试3：覆盖率测试

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/handler/...

# 查看覆盖率
go tool cover -func=coverage.out

# 生成HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

## 🔧 集成测试

### 测试4：创建集成测试文件

创建 `test/integration_test.go`:

```go
package test

import (
	"context"
	"fmt"
	"testing"
	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/llm"
)

// TestTwoPhaseIntegration 两阶段集成测试
func TestTwoPhaseIntegration(t *testing.T) {
	// 1. 连接数据库
	cfg := &database.Config{
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
	}

	db, err := database.NewConnection(cfg)
	if err != nil {
		t.Skipf("无法连接数据库: %v", err)
		return
	}
	defer db.Close()

	// 2. 创建Schema解析器
	parser := database.NewSchemaParser(db)

	// 3. 检查表数量
	tableCount, _ := parser.GetTableCount()
	t.Logf("数据库表数量: %d", tableCount)

	// 4. 创建LLM客户端
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("需要设置GLM_API_KEY环境变量")
		return
	}
	llmClient := llm.NewGLMClient(apiKey, "")

	// 5. 创建两阶段处理器
	twoPhaseHandler := handler.NewTwoPhaseQueryHandler(parser, llmClient)

	// 6. 测试查询
	testCases := []struct {
		question     string
		expectTables []string
	}{
		{
			question:     "查询VIP用户",
			expectTables: []string{"boom_user"},
		},
		{
			question:     "查询订单信息",
			expectTables: []string{"boom_order_paid_water"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.question, func(t *testing.T) {
			result, err := twoPhaseHandler.Handle(context.Background(), tc.question)
			if err != nil {
				t.Fatalf("查询失败: %v", err)
			}

			t.Logf("问题: %s", result.Question)
			t.Logf("生成的SQL: %s", result.SQL)
			t.Logf("主要表: %v", result.Metadata["primary_tables"])
			t.Logf("次要表: %v", result.Metadata["secondary_tables"])
			t.Logf("选择理由: %s", result.Metadata["reasoning"])

			// 验证生成了SQL
			if result.SQL == "" {
				t.Error("期望生成SQL")
			}
		})
	}
}

// TestCompareApproaches 对比全量Schema和两阶段方案
func TestCompareApproaches(t *testing.T) {
	// 连接数据库
	db, err := database.NewConnection(&database.Config{
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
	})
	if err != nil {
		t.Skipf("无法连接数据库: %v", err)
		return
	}
	defer db.Close()

	parser := database.NewSchemaParser(db)
	llmClient := llm.NewGLMClient(os.Getenv("GLM_API_KEY"), "")

	question := "查询VIP用户"

	// 方案1: 全量Schema
	t.Run("全量Schema方案", func(t *testing.T) {
		handler := handler.NewQueryHandlerWithLLM(db, llmClient)

		start := time.Now()
		result, err := handler.Handle(context.Background(), question)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}

		t.Logf("全量方案 SQL: %s", result.SQL)
		t.Logf("全量方案 耗时: %v", duration)
	})

	// 方案2: 两阶段选择
	t.Run("两阶段选择方案", func(t *testing.T) {
		twoPhaseHandler := handler.NewTwoPhaseQueryHandler(parser, llmClient)

		start := time.Now()
		result, err := twoPhaseHandler.Handle(context.Background(), question)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("查询失败: %v", err)
		}

		t.Logf("两阶段方案 SQL: %s", result.SQL)
		t.Logf("两阶段方案 耗时: %v", duration)
		t.Logf("两阶段方案 主要表: %v", result.Metadata["primary_tables"])
	})
}
```

## 🚀 手动测试

### 测试5：命令行测试

```bash
# 1. 编译程序
go build -o bin/nlq cmd/nlq/main.go

# 2. 测试简单查询
./bin/nlq query "查询VIP用户" --verbose

# 3. 测试复杂查询
./bin/nlq query "查询最近一周下单金额最高的用户" --verbose

# 4. 测试JOIN查询
./bin/nlq query "查询用户的订单信息" --verbose

# 5. 测试聚合查询
./bin/nlq query "统计每个城市的用户数量" --verbose
```

### 测试6：Token使用测试

创建一个测试脚本 `test/token_usage.sh`:

```bash
#!/bin/bash

echo "=== Token使用测试 ==="

# 测试问题
questions=(
    "查询VIP用户"
    "查询订单信息"
    "查询用户的订单总额"
    "统计每个商品的销售数量"
    "查询最近注册的用户"
)

for question in "${questions[@]}"; do
    echo ""
    echo "问题: $question"
    ./bin/nlq query "$question" --verbose 2>&1 | grep -E "(Token|使用|耗时)"
done
```

运行测试：
```bash
chmod +x test/token_usage.sh
./test/token_usage.sh
```

## 📊 性能对比测试

### 测试7：创建性能对比脚本

创建 `test/performance_comparison.sh`:

```bash
#!/bin/bash

echo "=== 性能对比测试 ==="
echo "数据库: loloyal"
echo ""

# 获取表数量
TABLE_COUNT=$(docker exec -it mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" | wc -l)
echo "表数量: $TABLE_COUNT"
echo ""

# 测试问题
QUESTION="查询VIP用户的订单信息"

echo "=== 方案1: 全量Schema ==="
TIME1=$(./bin/nlq query "$QUESTION" 2>&1 | grep "耗时" | awk '{print $2}')
echo "耗时: $TIME1"

echo ""
echo "=== 方案2: 两阶段选择 ==="
TIME2=$(./bin/nlq query "$QUESTION" --mode=two-phase 2>&1 | grep "耗时" | awk '{print $2}')
echo "耗时: $TIME2"

echo ""
echo "=== 性能提升 ==="
echo "方案2比方案1快: $(echo "$TIME1 - $TIME2" | bc)s"
```

## 🔍 调试测试

### 测试8：启用详细日志

```bash
# 设置调试环境变量
export DEBUG=true
export VERBOSE=true

# 运行测试
go test -v ./internal/handler/... -run TestTwoPhase
```

### 测试9：单独测试组件

```bash
# 测试表摘要获取
go test -v ./internal/database/... -run TestGetTableSummaries

# 测试表详情获取
go test -v ./internal/database/... -run TestGetTableDetail

# 测试Schema格式化
go test -v ./internal/database/... -run TestFormatTablesForPrompt
```

## 📈 真实场景测试

### 测试10：端到端测试

创建 `test/e2e_test.sh`:

```bash
#!/bin/bash

echo "=== 端到端测试 ==="

# 1. 启动服务
echo "1. 启动HTTP服务"
./bin/nlq-server &
SERVER_PID=$!
sleep 3

# 2. 测试API
echo "2. 测试查询API"
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"question": "查询VIP用户"}' \
  | jq .

# 3. 测试Schema API
echo "3. 测试Schema API"
curl http://localhost:8080/api/v1/schema | jq .

# 4. 清理
echo "4. 清理"
kill $SERVER_PID
```

## 🎯 快速测试命令

```bash
# 快速验证所有测试通过
make test

# 快速验证编译成功
make build

# 快速测试两阶段方案
go test -v ./internal/handler/... -run TwoPhase

# 快速测试数据库连接
docker exec -it mysql mysql -uroot -proot -e "USE loloyal; SELECT COUNT(*) FROM boom_user;"
```

## 📝 测试检查清单

- [ ] 所有单元测试通过
- [ ] 数据库连接正常
- [ ] API Key配置正确
- [ ] 表摘要获取成功
- [ ] 表详情获取成功
- [ ] 两阶段选择正常工作
- [ ] SQL生成正确
- [ ] Token消耗在预期范围
- [ ] 响应时间可接受
- [ ] 错误处理正确

## 🐛 常见问题

### 问题1: 测试失败 "需要真实数据库连接"
**解决**: 确保MySQL容器正在运行
```bash
docker ps | grep mysql
```

### 问题2: API Key错误
**解决**: 设置环境变量
```bash
export GLM_API_KEY="your-api-key-here"
```

### 问题3: 表选择不准确
**解决**: 检查表注释和命名规范
```sql
-- 查看表注释
SELECT table_name, table_comment
FROM information_schema.tables
WHERE table_schema = 'loloyal';
```

---

哼，按照这个测试指南，笨蛋也能把两阶段方案测试得明明白白呢！(￣▽￣)ゞ
