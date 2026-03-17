# 🧪 NLQ两阶段方案 - 立即可用的测试命令

## 🚀 快速开始（3步测试）

### 1️⃣ 运行快速测试脚本
```bash
./test/quick_test.sh
```

### 2️⃣ 测试单元测试
```bash
# 测试JSON提取功能
go test -v ./internal/handler/... -run TestExtractJSON

# 测试表选择功能
go test -v ./internal/handler/... -run TestTableSelector

# 测试Schema构建
go test -v ./internal/handler/... -run TestSchemaBuilder

# 运行所有handler测试
go test -v ./internal/handler/...
```

### 3️⃣ 测试数据库连接
```bash
# 查看数据库schema
./bin/nlq schema

# 查看特定表
./bin/nlq schema boom_user
```

## 📊 功能测试（设置API Key后）

```bash
# 1. 设置API Key
export GLM_API_KEY="your-api-key-here"

# 2. 测试简单查询
./bin/nlq query "查询VIP用户" --verbose

# 3. 测试复杂查询
./bin/nlq query "查询最近一周下单金额最高的用户" --verbose

# 4. 测试JOIN查询
./bin/nlq query "查询用户的订单信息" --verbose
```

## 🔧 高级测试

### 单元测试覆盖率
```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/handler/...

# 查看函数覆盖率
go tool cover -func=coverage.out

# 生成HTML报告
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### 性能测试
```bash
# 测试响应时间
time ./bin/nlq query "查询VIP用户"

# 对比不同查询的性能
for q in "查询用户" "查询订单" "查询产品"; do
    echo "测试: $q"
    time ./bin/nlq query "$q"
done
```

### 集成测试
```bash
# 测试数据库模块
go test -v ./internal/database/...

# 测试LLM模块
go test -v ./internal/llm/...

# 测试安全模块
go test -v ./pkg/security/...
```

## 🐛 调试测试

### 启用详细日志
```bash
# 设置调试环境变量
export DEBUG=true
export VERBOSE=true

# 运行测试
go test -v ./internal/handler/... -run TestTwoPhase
```

### 查看数据库表信息
```bash
# 查看所有表
./bin/nlq schema

# 查看表数量
docker exec mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" | wc -l

# 查看表结构
docker exec mysql mysql -uroot -proot -e "USE loloyal; DESCRIBE boom_user;"
```

## 📈 Token使用测试

```bash
# 创建测试脚本
cat > test_token_usage.sh << 'EOF'
#!/bin/bash
export GLM_API_KEY="your-api-key-here"

questions=(
    "查询VIP用户"
    "查询订单信息"
    "查询用户订单"
)

for q in "${questions[@]}"; do
    echo "问题: $q"
    ./bin/nlq query "$q" --verbose
    echo "---"
done
EOF

chmod +x test_token_usage.sh
./test_token_usage.sh
```

## 🎯 特定功能测试

### 测试表摘要获取
```bash
# 运行数据库测试
go test -v ./internal/database/... -run TestGetTableSummaries
```

### 测试两阶段选择
```bash
# 需要设置API Key
export GLM_API_KEY="your-api-key-here"

# 这个功能已经集成到主handler中
# 当表数量>20时自动启用
./bin/nlq query "查询VIP用户" --verbose
```

### 测试错误处理
```bash
# 测试无效SQL
./bin/nlq query "DELETE FROM users"

# 测试空问题
./bin/nlq query ""

# 测试无API Key
unset GLM_API_KEY
./bin/nlq query "查询用户"
```

## 📝 手动验证步骤

### 1. 验证数据库结构
```bash
# 确认数据库存在
docker exec mysql mysql -uroot -proot -e "SHOW DATABASES LIKE 'loloyal';"

# 确认表存在
docker exec mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;"

# 确认有数据
docker exec mysql mysql -uroot -proot -e "USE loloyal; SELECT COUNT(*) FROM boom_user;"
```

### 2. 验证编译成功
```bash
# 编译CLI工具
go build -o bin/nlq cmd/nlq/main.go

# 编译服务器工具
go build -o bin/nlq-server cmd/nlq-server/main.go

# 验证文件存在
ls -lh bin/
```

### 3. 验证测试通过
```bash
# 运行所有测试
go test ./... 2>&1 | grep -E "(PASS|FAIL)"

# 查看测试总结
go test ./... 2>&1 | tail -5
```

## 🚨 故障排查

### 问题: 数据库连接失败
```bash
# 检查MySQL容器
docker ps | grep mysql

# 检查MySQL日志
docker logs mysql

# 尝试直接连接
docker exec -it mysql mysql -uroot -proot
```

### 问题: API Key错误
```bash
# 检查环境变量
echo $GLM_API_KEY

# 设置正确的API Key
export GLM_API_KEY="your-actual-api-key"
```

### 问题: 测试失败
```bash
# 清理缓存
go clean -testcache

# 重新运行测试
go test -v ./internal/handler/...

# 查看详细错误
go test -v ./internal/handler/... 2>&1 | grep -A 10 "FAIL"
```

## ✨ 推荐测试顺序

1. **环境检查**: `./test/quick_test.sh`
2. **单元测试**: `go test -v ./internal/handler/...`
3. **数据库测试**: `./bin/nlq schema`
4. **功能测试**: `./bin/nlq query "查询VIP用户"`
5. **性能测试**: `time ./bin/nlq query "查询VIP用户"`

---

哼，按照这些命令一步步测试，就能把两阶段方案的功能都验证清楚了呢！(￣▽￣)ゞ
