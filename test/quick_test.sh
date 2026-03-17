#!/bin/bash

echo "🧪 NLQ两阶段方案快速测试"
echo "================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查函数
check_pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
}

check_fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
}

check_warn() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
}

# 1. 检查环境
echo "1️⃣  检查测试环境"
echo "--------------------------------"

# 检查Docker MySQL
if docker ps | grep -q mysql; then
    check_pass "MySQL容器正在运行"
else
    check_fail "MySQL容器未运行"
    echo "   请先启动MySQL: docker-compose up -d mysql"
    exit 1
fi

# 检查API Key
if [ -n "$GLM_API_KEY" ]; then
    check_pass "GLM_API_KEY已设置"
else
    check_warn "GLM_API_KEY未设置"
    echo "   设置方式: export GLM_API_KEY=\"your-api-key-here\""
fi

# 检查编译
echo ""
echo "2️⃣  检查编译状态"
echo "--------------------------------"

if go build -o bin/nlq cmd/nlq/main.go 2>/dev/null; then
    check_pass "CLI工具编译成功"
else
    check_fail "CLI工具编译失败"
    exit 1
fi

if go build -o bin/nlq-server cmd/nlq-server/main.go 2>/dev/null; then
    check_pass "服务器工具编译成功"
else
    check_fail "服务器工具编译失败"
    exit 1
fi

# 3. 运行单元测试
echo ""
echo "3️⃣  运行单元测试"
echo "--------------------------------"

echo "运行handler测试..."
if go test -v ./internal/handler/... -run TestExtractJSON 2>&1 | grep -q "PASS"; then
    check_pass "JSON提取测试通过"
else
    check_fail "JSON提取测试失败"
fi

echo "运行database测试..."
if go test -v ./internal/database/... 2>&1 | grep -q "PASS"; then
    check_pass "数据库模块测试通过"
else
    check_fail "数据库模块测试失败"
fi

# 4. 检查数据库连接
echo ""
echo "4️⃣  检查数据库连接"
echo "--------------------------------"

# 尝试不同的MySQL连接方式
TABLE_COUNT=""
if docker exec mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" >/dev/null 2>&1; then
    TABLE_COUNT=$(docker exec mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" 2>/dev/null | wc -l | tr -d ' ')
elif docker exec -it mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" >/dev/null 2>&1; then
    TABLE_COUNT=$(docker exec -it mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;" 2>/dev/null | wc -l | tr -d ' ')
fi

if [ -n "$TABLE_COUNT" ] && [ "$TABLE_COUNT" -gt 0 ]; then
    check_pass "数据库连接正常，共 $TABLE_COUNT 个表"
else
    check_warn "无法直接连接到数据库容器"
    echo "   你可以直接测试: ./bin/nlq schema"
fi

# 5. 显示测试命令
echo ""
echo "5️⃣  可用的测试命令"
echo "--------------------------------"

echo "📝 单元测试:"
echo "   go test -v ./internal/handler/..."
echo "   go test -v ./internal/database/..."
echo ""

echo "🔧 集成测试:"
echo "   ./bin/nlq query \"查询VIP用户\" --verbose"
echo "   ./bin/nlq schema"
echo ""

echo "📊 覆盖率测试:"
echo "   go test -coverprofile=coverage.out ./..."
echo "   go tool cover -html=coverage.out"
echo ""

echo "🚀 性能测试:"
echo "   ./test/performance_test.sh"
echo ""

# 6. 总结
echo ""
echo "================================"
echo "✨ 测试环境检查完成！"
echo ""
echo "下一步:"
echo "1. 设置API Key: export GLM_API_KEY=\"your-api-key-here\""
echo "2. 运行查询: ./bin/nlq query \"查询VIP用户\""
echo "3. 查看文档: cat docs/TESTING_GUIDE.md"
echo ""
