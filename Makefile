.PHONY: help test test-unit coverage build run clean deps fmt lint vet

# 默认目标
.DEFAULT_GOAL := help

# 项目配置
BINARY_NAME=nlq
BUILD_DIR=bin
CMD_DIR=cmd/nlq
MAIN_FILE=$(CMD_DIR)/main.go

# Go配置
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# 颜色输出
COLOR_RESET=\033[0m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m
COLOR_BLUE=\033[34m

## help: 显示帮助信息
help:
	@echo "$(COLOR_BLUE)NLQ - Natural Language Query 工具$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_GREEN)可用命令:$(COLOR_RESET)"
	@grep -E '^## ' Makefile | sed 's/## /  /' | sed 's/: /: /' | column -t -s ':'
	@echo ""

## deps: 下载依赖
deps:
	@echo "$(COLOR_YELLOW)下载依赖...$(COLOR_RESET)"
	$(GOMOD) download
	$(GOMOD) tidy

## deps-llm: 安装LLM依赖（langchaingo）
deps-llm:
	@echo "$(COLOR_YELLOW)安装LLM依赖...$(COLOR_RESET)"
	$(GOGET) github.com/tmc/langchaingo
	$(GOMOD) tidy

## deps-cli: 安装CLI依赖（cobra等）
deps-cli:
	@echo "$(COLOR_YELLOW)安装CLI依赖...$(COLOR_RESET)"
	$(GOGET) github.com/spf13/cobra@latest
	$(GOGET) github.com/spf13/viper@latest
	$(GOMOD) tidy

## test: 运行所有测试
test:
	@echo "$(COLOR_YELLOW)运行所有测试...$(COLOR_RESET)"
	$(GOTEST) -v -cover ./...

## test-unit: 运行单元测试（跳过集成测试）
test-unit:
	@echo "$(COLOR_YELLOW)运行单元测试...$(COLOR_RESET)"
	$(GOTEST) -v -short ./internal/... ./pkg/...

## test-coverage: 生成测试覆盖率报告
test-coverage:
	@echo "$(COLOR_YELLOW)生成测试覆盖率报告...$(COLOR_RESET)"
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)覆盖率报告已生成: coverage.html$(COLOR_RESET)"

## test-coverage-term: 在终端显示测试覆盖率
test-coverage-term:
	@echo "$(COLOR_YELLOW)测试覆盖率（终端模式）...$(COLOR_RESET)"
	$(GOTEST) -cover ./...

## fmt: 格式化代码
fmt:
	@echo "$(COLOR_YELLOW)格式化代码...$(COLOR_RESET)"
	$(GOFMT) -s -w .

## lint: 运行代码检查
lint:
	@echo "$(COLOR_YELLOW)运行代码检查...$(COLOR_RESET)"
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "$(COLOR_YELLOW)golangci-lint 未安装，跳过代码检查$(COLOR_RESET)"; \
		echo "安装: brew install golangci-lint"; \
	fi

## vet: 运行go vet检查
vet:
	@echo "$(COLOR_YELLOW)运行go vet检查...$(COLOR_RESET)"
	$(GOCMD) vet ./...

## check: 运行所有检查（fmt, vet, test）
check: fmt vet test
	@echo "$(COLOR_GREEN)所有检查通过！$(COLOR_RESET)"

## build: 编译项目
build:
	@echo "$(COLOR_YELLOW)编译项目...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "$(COLOR_GREEN)编译完成: $(BUILD_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

## build-server: 编译HTTP服务器
build-server:
	@echo "$(COLOR_YELLOW)编译HTTP服务器...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/nlq-server cmd/nlq-server/main.go
	@echo "$(COLOR_GREEN)HTTP服务器编译完成: $(BUILD_DIR)/nlq-server$(COLOR_RESET)"

## build-all: 编译所有二进制文件
build-all: build build-server
	@echo "$(COLOR_GREEN)所有二进制文件编译完成！$(COLOR_RESET)"

## build-debug: 编译项目（调试模式）
build-debug:
	@echo "$(COLOR_YELLOW)编译项目（调试模式）...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_FILE)
	@echo "$(COLOR_GREEN)调试版本编译完成: $(BUILD_DIR)/$(BINARY_NAME)-debug$(COLOR_RESET)"

## run: 运行项目
run:
	@echo "$(COLOR_YELLOW)运行项目...$(COLOR_RESET)"
	$(GOCMD) run $(MAIN_FILE)

## clean: 清理编译产物
clean:
	@echo "$(COLOR_YELLOW)清理编译产物...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "$(COLOR_GREEN)清理完成$(COLOR_RESET)"

## clean-mod: 清理依赖并重新下载
clean-mod:
	@echo "$(COLOR_YELLOW)清理依赖...$(COLOR_RESET)"
	@rm -rf go.sum
	$(GOMOD) tidy
	@echo "$(COLOR_GREEN)依赖清理完成$(COLOR_RESET)"

## install: 安装到GOPATH/bin
install:
	@echo "$(COLOR_YELLOW)安装到GOPATH/bin...$(COLOR_RESET)"
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(MAIN_FILE)
	@echo "$(COLOR_GREEN)安装完成: $(GOPATH)/bin/$(BINARY_NAME)$(COLOR_RESET)"

## test-db: 测试数据库连接
test-db:
	@echo "$(COLOR_YELLOW)测试数据库连接...$(COLOR_RESET)"
	docker exec mysql mysql -uroot -proot -e "SELECT 1"

## show-tables: 显示数据库中的所有表
show-tables:
	@echo "$(COLOR_YELLOW)数据库表列表：$(COLOR_RESET)"
	docker exec mysql mysql -uroot -proot -e "USE loloyal; SHOW TABLES;"

## describe-table: 描述表结构（使用 TABLE=table_name）
describe-table:
	@if [ -z "$(TABLE)" ]; then \
		echo "$(COLOR_YELLOW)用法: make describe-table TABLE=table_name$(COLOR_RESET)"; \
		exit 1; \
	fi
	@echo "$(COLOR_YELLOW)表 $(TABLE) 的结构：$(COLOR_RESET)"
	docker exec mysql mysql -uroot -proot -e "USE loloyal; DESCRIBE $(TABLE);"

## setup: 初始化项目设置
setup: deps
	@echo "$(COLOR_YELLOW)初始化项目设置...$(COLOR_RESET)"
	@mkdir -p config
	@echo "$(COLOR_GREEN)项目设置完成$(COLOR_RESET)"

## all: 完整构建流程（检查+编译）
all: check build
	@echo "$(COLOR_GREEN)完整构建成功！$(COLOR_RESET)"

## dev: 开发模式（运行所有检查和测试）
dev: fmt vet test-unit
	@echo "$(COLOR_GREEN)开发检查完成！$(COLOR_RESET)"
