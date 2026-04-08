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
	"github.com/channelwill/nlq/internal/llm"
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
	if cfg.LLM.APIKey == "" {
		fmt.Fprintln(os.Stderr, "❌ 错误: NLQ服务需要配置LLM API Key才能启动")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "📝 配置步骤:")
		fmt.Fprintln(os.Stderr, "1. 访问 LLM 提供商平台获取 API Key")
		fmt.Fprintln(os.Stderr, "2. 设置环境变量:")
		fmt.Fprintln(os.Stderr, "   export LLM_API_KEY=\"your-api-key-here\"")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "然后重新启动服务器")
		os.Exit(1)
	}

	// 1. 构建知识路由器（只加载索引，不加载全文）
	knowledgeRouter := buildKnowledgeRouter()

	// 2. 创建 LLM 客户端
	llmClient, err := createLLMClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建LLM客户端失败: %v\n", err)
		os.Exit(1)
	}

	// 3. 创建 Agent 查询处理器
	queryHandler := createAgentHandler(db, llmClient, knowledgeRouter, cfg)

	// 4. 创建HTTP服务器
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

	// 打印查询页面链接
	queryURL := fmt.Sprintf("http://%s:%d/query", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("🔗 查询页面: %s\n", queryURL)

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
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// buildKnowledgeRouter 构建知识路由器（启动时只加载索引）
func buildKnowledgeRouter() *knowledge.Router {
	router := knowledge.NewRouter()

	// 定义要加载的知识库目录（不重复，不递归嵌套）
	knowledgeDirs := []string{
		"knowledge/model",       // 知识库文档（核心）
		"knowledge/positive",   // 正面反馈知识库（可选）
		"knowledge/negative",   // 负面反馈知识库（可选）
	}

	totalDocs := 0
	for _, dir := range knowledgeDirs {
		docsBefore := router.GetIndexCount()
		if err := router.BuildIndex(dir); err != nil {
			fmt.Printf("  ⚠️  跳过知识库目录 %s: %v\n", dir, err)
			continue
		}
		docsAfter := router.GetIndexCount()
		added := docsAfter - docsBefore
		if added > 0 {
			fmt.Printf("  📚 从 %s 索引了 %d 个文档\n", dir, added)
			totalDocs += added
		}
	}

	if totalDocs > 0 {
		fmt.Printf("  ✅ 知识路由器就绪，共 %d 个文档索引\n", totalDocs)
	} else {
		fmt.Println("  📭 未找到知识库文档")
	}

	return router
}

// createLLMClient 创建 LLM 客户端
func createLLMClient(cfg *config.Config) (llm.LLMClient, error) {
	opts := &llm.LLMOptions{
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	}

	client, err := llm.NewLLMClient(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model, opts)
	if err != nil {
		return nil, fmt.Errorf("创建LLM客户端失败: %w", err)
	}

	fmt.Printf("🤖 LLM客户端就绪 | Provider: %s | Model: %s\n", cfg.LLM.Provider, cfg.LLM.Model)
	return client, nil
}

// createAgentHandler 创建 Agent 查询处理器
func createAgentHandler(db *gorm.DB, llmClient llm.LLMClient, knowledgeRouter *knowledge.Router, cfg *config.Config) handler.QueryHandlerInterface {
	parser := database.NewSchemaParser(db)

	agentConfig := handler.AgentConfig{
		MaxSelfCorrect: cfg.Query.Agent.MaxSelfCorrect,
		MaxTurns:       cfg.Query.Agent.MaxTurns,
		Verbose:        cfg.Query.Agent.Verbose,
	}

	agentHandler := handler.NewAgentQueryHandler(parser, db, llmClient, knowledgeRouter, agentConfig)

	fmt.Printf("🤖 Agent处理器就绪 | maxSelfCorrect: %d | maxTurns: %d | verbose: %v\n",
		agentConfig.MaxSelfCorrect, agentConfig.MaxTurns, agentConfig.Verbose)

	return agentHandler
}
