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
	"github.com/channelwill/nlq/internal/server"
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

	// 使用两阶段处理器（推荐用于大型数据库）
	queryHandler := handler.NewTwoPhaseQueryHandlerWithLLM(db, cfg.LLM.APIKey, cfg.LLM.BaseURL)
	fmt.Printf("🤖 使用两阶段查询处理器 + GLM4.7 LLM: %s\n", cfg.LLM.Model)

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
	cfg := &config.Config{}
	cfg.SetDefaults()

	// 设置默认数据库配置（与CLI工具一致）
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 3306
	cfg.Database.Database = "loloyal"
	cfg.Database.Username = "root"
	cfg.Database.Password = "root"
	cfg.Database.Readonly = true

	// 尝试从配置文件加载
	configFile := "config/config.yaml"
	if _, err := os.Stat(configFile); err == nil {
		// 配置文件存在，从文件加载
		loadedCfg, err := config.LoadFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("加载配置文件失败: %w", err)
		}
		cfg = loadedCfg
	} else {
		fmt.Println("⚠️  配置文件不存在，使用默认配置")
	}

	// 尝试从环境变量覆盖
	if apiKey := os.Getenv("GLM_API_KEY"); apiKey != "" {
		cfg.LLM.APIKey = apiKey
	}
	if dbHost := os.Getenv("DATABASE_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DATABASE_PORT"); dbPort != "" {
		fmt.Sscanf(dbPort, "%d", &cfg.Database.Port)
	}
	if dbName := os.Getenv("DATABASE_NAME"); dbName != "" {
		cfg.Database.Database = dbName
	}

	return cfg, nil
}
