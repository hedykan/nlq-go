package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/server"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 连接数据库
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建查询处理器（强制要求LLM模式）
	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "${GLM_API_KEY}" || cfg.LLM.APIKey == "your-api-key-here" {
		fmt.Fprintln(os.Stderr, "❌ 错误: NLQ服务需要配置GLM API Key才能启动")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "📝 配置步骤:")
		fmt.Fprintln(os.Stderr, "1. 访问智谱AI开放平台: https://open.bigmodel.cn/")
		fmt.Fprintln(os.Stderr, "2. 注册/登录账号")
		fmt.Fprintln(os.Stderr, "3. 创建API Key")
		fmt.Fprintln(os.Stderr, "4. 设置环境变量:")
		fmt.Fprintln(os.Stderr, "   export GLM_API_KEY=\"your-api-key-here\"")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "然后重新启动服务器")
		os.Exit(1)
	}

	// 智能选择查询处理器
	queryHandler, err := createQueryHandler(db, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建查询处理器失败: %v\n", err)
		os.Exit(1)
	}

	// 自动加载知识库
	if err := loadKnowledgeBases(queryHandler); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  加载知识库失败: %v\n", err)
		// 继续启动，知识库加载失败不应阻止服务器启动
	} else {
		fmt.Println("✅ 知识库加载成功")
	}

	// 创建HTTP服务器
	srv, err := server.NewServer(cfg, queryHandler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建服务器失败: %v\n", err)
		os.Exit(1)
	}

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "服务器启动失败: %v\n", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n正在关闭服务器...")

	// 优雅关闭
	ctx := context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "服务器关闭失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("服务器已关闭")
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	// 使用viper加载配置（支持配置文件、环境变量、默认值）
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// loadKnowledgeBases 自动加载所有知识库
func loadKnowledgeBases(queryHandler handler.QueryHandlerInterface) error {
	loader := knowledge.NewLoader()

	// 定义要加载的知识库目录
	knowledgeDirs := []string{
		"knowledge",           // 主知识库目录
		"knowledge/positive",  // 正面反馈知识库
		"knowledge/negative",  // 负面反馈知识库
	}

	var allDocs []knowledge.Document

	// 加载所有知识库目录
	for _, dir := range knowledgeDirs {
		docs, err := loader.LoadFromDirectory(dir)
		if err != nil {
			// 目录不存在或无法读取，继续尝试其他目录
			fmt.Printf("  ⚠️  跳过 %s: %v\n", dir, err)
			continue
		}

		if len(docs) > 0 {
			fmt.Printf("  📚 从 %s 加载了 %d 个文档\n", dir, len(docs))
			allDocs = append(allDocs, docs...)
		}
	}

	// 如果有文档，设置到查询处理器
	if len(allDocs) > 0 {
		if err := queryHandler.SetKnowledge(allDocs); err != nil {
			return fmt.Errorf("设置知识库到查询处理器失败: %w", err)
		}
		fmt.Printf("  ✅ 总共加载 %d 个知识库文档\n", len(allDocs))
	} else {
		fmt.Println("  📭 未找到知识库文档")
	}

	return nil
}

// createQueryHandler 智能创建查询处理器
// 根据配置模式和数据库表数量自动选择最优处理器
func createQueryHandler(db *gorm.DB, cfg *config.Config) (handler.QueryHandlerInterface, error) {
	// 获取数据库表数量
	tables, err := getTableCount(db)
	if err != nil {
		return nil, fmt.Errorf("获取表数量失败: %w", err)
	}

	fmt.Printf("📊 数据库表数量: %d\n", tables)

	// 根据配置模式选择处理器
	mode := cfg.Query.Mode
	threshold := cfg.Query.TableCountThreshold

	var queryHandler handler.QueryHandlerInterface
	var handlerType string

	switch mode {
	case "simple":
		// 强制使用单步法
		queryHandler = handler.NewQueryHandlerWithLLM(db, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
		handlerType = "单步法（QueryHandler）"

	case "two_phase":
		// 强制使用两步法
		queryHandler = handler.NewTwoPhaseQueryHandlerWithLLM(db, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
		handlerType = "两步法（TwoPhaseQueryHandler）"

	case "auto":
		// 自动模式：根据表数量智能选择
		if tables <= threshold {
			// 小型数据库：使用单步法（速度快）
			queryHandler = handler.NewQueryHandlerWithLLM(db, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
			handlerType = "单步法（QueryHandler）"
			fmt.Printf("✅ 检测到小型数据库（%d ≤ %d），使用单步法以提高性能\n", tables, threshold)
		} else {
			// 大型数据库：使用两步法（精准度高）
			queryHandler = handler.NewTwoPhaseQueryHandlerWithLLM(db, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
			handlerType = "两步法（TwoPhaseQueryHandler）"
			fmt.Printf("✅ 检测到大型数据库（%d > %d），使用两步法以保证精准度\n", tables, threshold)
		}

	default:
		return nil, fmt.Errorf("未知的查询模式: %s（支持: auto, simple, two_phase）", mode)
	}

	fmt.Printf("🤖 查询处理器: %s\n", handlerType)
	fmt.Printf("🤖 LLM模型: %s\n", cfg.LLM.Model)

	return queryHandler, nil
}

// getTableCount 获取数据库表数量
func getTableCount(db *gorm.DB) (int, error) {
	// 使用GORM获取表数量
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("查询表数量失败: %w", err)
	}

	return int(count), nil
}
