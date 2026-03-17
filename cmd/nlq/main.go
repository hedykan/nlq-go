package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/internal/handler"
	"github.com/channelwill/nlq/internal/knowledge"
	"github.com/channelwill/nlq/internal/sql"
	"github.com/spf13/cobra"
)

var (
	configFile    string
	verbose       bool
	jsonOutput    bool
	wideOutput    bool
	compactOutput bool
	columnsFilter string
	knowledgePath string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "nlq",
		Short: "NLQ - 基于自然语言的数据库查询工具",
		Long: `NLQ (Natural Language Query) 是一个创新的数据库查询工具，
允许用户使用自然语言提问，自动转换为SQL查询并返回结果。`,
	}

	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "显示详细输出")

	// queryCmd 查询命令
	var queryCmd = &cobra.Command{
		Use:   "query [问题/SQL]",
		Short: "执行自然语言查询或SQL查询",
		Long:  "执行自然语言查询或直接使用SQL查询数据库",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runQuery,
	}

	queryCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "以JSON格式输出")
	queryCmd.Flags().BoolVarP(&wideOutput, "wide", "w", false, "显示所有列（不受列数限制）")
	queryCmd.Flags().BoolVarP(&compactOutput, "compact", "", false, "紧凑模式（更简洁的显示）")
	queryCmd.Flags().StringVarP(&columnsFilter, "columns", "", "", "指定要显示的列（逗号分隔，如：id,name,created_at）")
	queryCmd.Flags().StringVarP(&knowledgePath, "knowledge", "k", "", "知识库文件夹路径（包含MD文档）")

	// sqlCmd SQL命令
	var sqlCmd = &cobra.Command{
		Use:   "sql [SQL语句]",
		Short: "直接执行SQL查询",
		Long:  "直接执行SQL查询语句（仅限SELECT）",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runSQL,
	}

	sqlCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "以JSON格式输出")
	sqlCmd.Flags().BoolVarP(&wideOutput, "wide", "w", false, "显示所有列（不受列数限制）")
	sqlCmd.Flags().BoolVarP(&compactOutput, "compact", "", false, "紧凑模式（更简洁的显示）")
	sqlCmd.Flags().StringVarP(&columnsFilter, "columns", "", "", "指定要显示的列（逗号分隔，如：id,name,created_at）")
	sqlCmd.Flags().StringVarP(&knowledgePath, "knowledge", "k", "", "知识库文件夹路径（包含MD文档）")

	// schemaCmd schema命令
	var schemaCmd = &cobra.Command{
		Use:   "schema [表名]",
		Short: "显示数据库Schema",
		Long:  "显示数据库Schema信息，可指定表名",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runSchema,
	}

	// 添加子命令
	rootCmd.AddCommand(queryCmd, sqlCmd, schemaCmd)

	// 执行
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// runQuery 执行查询
func runQuery(cmd *cobra.Command, args []string) error {
	question := args[0]

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 连接数据库
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建查询处理器（强制要求LLM模式）
	if cfg.LLM.APIKey == "" || cfg.LLM.APIKey == "${GLM_API_KEY}" || cfg.LLM.APIKey == "your-api-key-here" {
		fmt.Fprintln(os.Stderr, "❌ 错误: NLQ服务需要配置GLM API Key才能使用")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "📝 配置步骤:")
		fmt.Fprintln(os.Stderr, "1. 访问智谱AI开放平台: https://open.bigmodel.cn/")
		fmt.Fprintln(os.Stderr, "2. 注册/登录账号")
		fmt.Fprintln(os.Stderr, "3. 创建API Key")
		fmt.Fprintln(os.Stderr, "4. 设置环境变量或配置文件:")
		fmt.Fprintln(os.Stderr, "   export GLM_API_KEY=\"your-api-key-here\"")
		fmt.Fprintln(os.Stderr, "   或编辑 config/config.yaml 中的 api_key 字段")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "💡 简化模式已移除，NLQ现在强制使用GLM4.7以提供更准确的查询结果！")
		os.Exit(1)
	}

	queryHandler := handler.NewQueryHandlerWithLLM(db, cfg.LLM.APIKey, cfg.LLM.BaseURL)
	if verbose {
		fmt.Printf("🤖 使用GLM4.7 LLM: %s\n", cfg.LLM.Model)
	}

	// 加载知识库（如果指定）
	if knowledgePath != "" {
		if err := loadKnowledgeBase(queryHandler, knowledgePath, verbose); err != nil {
			fmt.Printf("⚠️  警告: 加载知识库失败: %v\n", err)
		}
	}

	// 判断是SQL还是自然语言
	if isSQLQuery(question) {
		// 直接执行SQL
		if verbose {
			fmt.Printf("📝 SQL查询: %s\n", question)
		}

		result, err := queryHandler.HandleWithSQL(context.Background(), question)
		if err != nil {
			return err
		}

		displayResult(result, jsonOutput)
	} else {
		// 自然语言查询
		if verbose {
			fmt.Printf("❓ 问题: %s\n", question)
		}

		result, err := queryHandler.Handle(context.Background(), question)
		if err != nil {
			return err
		}

		displayResult(result, jsonOutput)
	}

	return nil
}

// runSQL 执行SQL
func runSQL(cmd *cobra.Command, args []string) error {
	sqlQuery := args[0]

	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 连接数据库
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建查询处理器
	queryHandler := handler.NewQueryHandler(db)

	// 执行SQL
	if verbose {
		fmt.Printf("📝 SQL查询: %s\n", sqlQuery)
	}

	result, err := queryHandler.HandleWithSQL(context.Background(), sqlQuery)
	if err != nil {
		return err
	}

	displayResult(result, jsonOutput)
	return nil
}

// runSchema 显示Schema
func runSchema(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 连接数据库
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建查询处理器
	queryHandler := handler.NewQueryHandler(db)

	if len(args) == 1 {
		// 显示特定表的信息
		tableName := args[0]
		if verbose {
			fmt.Printf("📊 表: %s\n", tableName)
		}

		table, err := queryHandler.GetTableInfo(tableName)
		if err != nil {
			return fmt.Errorf("获取表信息失败: %w", err)
		}

		displayTableSchema(table)
	} else {
		// 显示所有表
		if verbose {
			fmt.Println("📊 数据库Schema")
		}

		schema, err := queryHandler.GetSchema()
		if err != nil {
			return fmt.Errorf("获取Schema失败: %w", err)
		}

		fmt.Println(schema)
	}

	return nil
}

// loadConfig 加载配置
func loadConfig() (*config.Config, error) {
	cfg := &config.Config{}
	cfg.SetDefaults()

	// 如果指定了配置文件，从文件加载
	if configFile != "" {
		loadedCfg, err := config.LoadFromFile(configFile)
		if err != nil {
			return nil, err
		}
		cfg = loadedCfg
	}

	// 尝试从配置文件加载（默认路径）
	if configFile == "" {
		defaultConfigFile := "config/config.yaml"
		if _, err := os.Stat(defaultConfigFile); err == nil {
			// 配置文件存在，从文件加载
			loadedCfg, err := config.LoadFromFile(defaultConfigFile)
			if err == nil {
				cfg = loadedCfg
			}
		}
	}

	// 如果没有配置文件，使用默认配置（连接到本地Docker MySQL）
	if configFile == "" {
		cfg.Database.Host = "localhost"
		cfg.Database.Port = 3306
		cfg.Database.Database = "loloyal"
		cfg.Database.Username = "root"
		cfg.Database.Password = "root"
		cfg.Database.Readonly = true
	}

	// 尝试从环境变量覆盖（优先级最高）
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

// isSQLQuery 判断是否是SQL查询
func isSQLQuery(query string) bool {
	query = trimToLower(query)
	return startsWith(query, "select") || startsWith(query, "with")
}

// displayResult 显示结果
func displayResult(result *handler.QueryResult, jsonFormat bool) {
	if jsonFormat {
		// JSON格式输出
		fmt.Printf("{\"question\":\"%s\",\"sql\":\"%s\",\"count\":%d,\"duration_ms\":%d}\n",
			result.Question,
			result.SQL,
			result.Result.Count,
			result.Duration.Milliseconds(),
		)
		return
	}

	// 人类可读格式
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 查询结果")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if result.Question != "" {
		fmt.Printf("❓ 问题: %s\n", result.Question)
	}

	fmt.Printf("📝 SQL: %s\n", result.SQL)
	fmt.Printf("⏱️  耗时: %v\n", result.Duration)
	fmt.Printf("📊 结果数量: %d\n", result.Result.Count)

	if result.Result.Count > 0 {
		fmt.Println("\n结果：")
		displayResultTable(result.Result)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// displayResultTable 显示结果表格（智能美化版本）
func displayResultTable(result *sql.ExecuteResult) {
	if len(result.Rows) == 0 {
		fmt.Println("（无结果）")
		return
	}

	// 处理列过滤和选择
	columnsToShow := selectColumnsToShow(result.Columns)

	if len(columnsToShow) == 0 {
		fmt.Println("（没有可显示的列）")
		return
	}

	// 检查是否有列被过滤
	if len(columnsToShow) < len(result.Columns) {
		fmt.Printf("💡 显示 %d 列（共 %d 列，使用 --wide 查看所有列，或 --columns 指定列）\n",
			len(columnsToShow), len(result.Columns))
	}

	// 限制显示的行数
	maxRows := 10
	if len(result.Rows) < maxRows {
		maxRows = len(result.Rows)
	}

	// 根据输出模式选择样式
	if compactOutput {
		displayCompactTable(result, columnsToShow, maxRows)
	} else {
		displayPrettyTable(result, columnsToShow, maxRows)
	}
}

// selectColumnsToShow 选择要显示的列
func selectColumnsToShow(allColumns []string) []string {
	// 如果用户指定了列，使用用户指定的列
	if columnsFilter != "" {
		userColumns := strings.Split(columnsFilter, ",")
		var selectedColumns []string
		for _, col := range userColumns {
			col = strings.TrimSpace(col)
			// 检查列是否存在
			for _, existingCol := range allColumns {
				if strings.EqualFold(existingCol, col) {
					selectedColumns = append(selectedColumns, existingCol)
					break
				}
			}
		}
		return selectedColumns
	}

	// 如果是wide模式，显示所有列
	if wideOutput {
		return allColumns
	}

	// 默认：优先显示重要列，限制列数
	maxColumns := 8 // 默认最多显示8列
	if len(allColumns) <= maxColumns {
		return allColumns
	}

	// 定义重要列的优先级
	priorityColumns := []string{
		"id", "ID", "Id", "user_id", "customer_id", "order_id",
		"name", "username", "email", "phone", "title",
		"status", "state", "type", "category",
		"amount", "price", "total", "count", "quantity",
		"created_at", "updated_at", "date", "time",
		"country", "city", "address",
	}

	// 先收集优先级高的列
	var selectedColumns []string
	remainingColumns := make([]string, 0, len(allColumns))

	// 记录已选择的列，避免重复
	selectedMap := make(map[string]bool)

	for _, priorityCol := range priorityColumns {
		for _, existingCol := range allColumns {
			if strings.EqualFold(existingCol, priorityCol) && !selectedMap[existingCol] {
				selectedColumns = append(selectedColumns, existingCol)
				selectedMap[existingCol] = true
				break
			}
		}
		if len(selectedColumns) >= maxColumns {
			return selectedColumns
		}
	}

	// 添加剩余的列（按优先级）
	for _, col := range allColumns {
		if !selectedMap[col] {
			remainingColumns = append(remainingColumns, col)
		}
	}

	// 填充到最大列数
	remainingSlots := maxColumns - len(selectedColumns)
	if remainingSlots > 0 && len(remainingColumns) > 0 {
		if len(remainingColumns) > remainingSlots {
			remainingColumns = remainingColumns[:remainingSlots]
		}
		selectedColumns = append(selectedColumns, remainingColumns...)
	}

	return selectedColumns
}

// displayPrettyTable 显示美观的表格
func displayPrettyTable(result *sql.ExecuteResult, columns []string, maxRows int) {
	// 计算每列的最佳宽度
	columnWidths := calculateColumnWidths(result, columns, maxRows)

	// 显示表头
	printTableLine(columns, columnWidths, "┌", "┬", "┐")

	// 显示列名
	fmt.Print("│")
	for i, col := range columns {
		width := columnWidths[i]
		fmt.Printf(" %-*s │", width, truncateString(col, width))
	}
	fmt.Println()

	// 显示分隔线
	printTableLine(columns, columnWidths, "├", "┼", "┤")

	// 显示数据行
	for i := 0; i < maxRows; i++ {
		fmt.Print("│")
		for j, col := range columns {
			width := columnWidths[j]
			val := formatValue(result.Rows[i][col])
			fmt.Printf(" %-*s │", width, truncateString(val, width))
		}
		fmt.Println()
	}

	if len(result.Rows) > maxRows {
		fmt.Printf("│ %s │\n", truncateString(
			fmt.Sprintf("... 还有 %d 行", len(result.Rows)-maxRows),
			sumWidths(columnWidths)+2*len(columns)-1,
		))
	}

	// 显示底边
	printTableLine(columns, columnWidths, "└", "┴", "┘")
}

// displayCompactTable 显示紧凑的表格
func displayCompactTable(result *sql.ExecuteResult, columns []string, maxRows int) {
	// 显示列名（用 | 分隔）
	fmt.Println(strings.Join(columns, " | "))

	// 显示分隔线
	separators := make([]string, len(columns))
	for i := range separators {
		separators[i] = strings.Repeat("-", len(columns[i]))
	}
	fmt.Println(strings.Join(separators, "-+-"))

	// 显示数据行
	for i := 0; i < maxRows; i++ {
		values := make([]string, len(columns))
		for j, col := range columns {
			values[j] = formatValue(result.Rows[i][col])
		}
		fmt.Println(strings.Join(values, " | "))
	}

	if len(result.Rows) > maxRows {
		fmt.Printf("... 还有 %d 行\n", len(result.Rows)-maxRows)
	}
}

// calculateColumnWidths 计算每列的最佳宽度
func calculateColumnWidths(result *sql.ExecuteResult, columns []string, maxRows int) []int {
	widths := make([]int, len(columns))

	// 初始化为列名的宽度
	for i, col := range columns {
		widths[i] = len(col)
	}

	// 考虑数据的宽度
	for i := 0; i < maxRows; i++ {
		for j, col := range columns {
			val := formatValue(result.Rows[i][col])
			valLen := len(val)
			if valLen > widths[j] {
				// 限制最大宽度为30字符
				if valLen > 30 {
					widths[j] = 30
				} else {
					widths[j] = valLen
				}
			}
		}
	}

	// 确保最小宽度为8，最大宽度为30
	for i := range widths {
		if widths[i] < 8 {
			widths[i] = 8
		}
		if widths[i] > 30 {
			widths[i] = 30
		}
	}

	return widths
}

// printTableLine 打印表格线条
func printTableLine(columns []string, widths []int, start, middle, end string) {
	fmt.Print(start)
	for i, width := range widths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(columns)-1 {
			fmt.Print(middle)
		} else {
			fmt.Print(end)
		}
	}
	fmt.Println()
}

// sumWidths 计算宽度总和
func sumWidths(widths []int) int {
	sum := 0
	for _, w := range widths {
		sum += w
	}
	return sum
}

// formatValue 格式化值显示
func formatValue(value interface{}) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case []byte:
		return string(v)
	case string:
		// 如果是长字符串，显示为...
		if len(v) > 50 {
			return v[:47] + "..."
		}
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// displayTableSchema 显示表Schema
func displayTableSchema(table database.TableSchema) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 表: %s (%d 列)\n", table.Name, len(table.Columns))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("┌────────────────────┬──────────────────┬──────────┬────────────┐")
	fmt.Println("│ 列名               │ 类型             │ 可空     │ 注释       │")
	fmt.Println("├────────────────────┼──────────────────┼──────────┼────────────┤")

	for _, col := range table.Columns {
		nullable := "NOT NULL"
		if col.Nullable {
			nullable = "NULL"
		}
		comment := col.Comment
		if len(comment) > 10 {
			comment = comment[:10] + "..."
		}

		fmt.Printf("│ %-18s │ %-16s │ %-8s │ %-10s │\n",
			truncateString(col.Name, 18),
			truncateString(col.Type, 16),
			nullable,
			truncateString(comment, 10),
		)
	}

	fmt.Println("└────────────────────┴──────────────────┴──────────┴────────────┘")
	fmt.Println()
}

// 辅助函数
func trimToLower(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func startsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// loadKnowledgeBase 加载知识库并设置到查询处理器
func loadKnowledgeBase(queryHandler *handler.QueryHandler, knowledgePath string, verbose bool) error {
	// 创建知识库加载器
	loader := knowledge.NewLoader()

	// 加载知识库文档
	docs, err := loader.LoadFromDirectory(knowledgePath)
	if err != nil {
		return fmt.Errorf("加载知识库文档失败: %w", err)
	}

	if len(docs) == 0 {
		if verbose {
			fmt.Println("📚 知识库为空，未找到MD文档")
		}
		return nil
	}

	// 设置知识库到查询处理器
	if err := queryHandler.SetKnowledge(docs); err != nil {
		return fmt.Errorf("设置知识库失败: %w", err)
	}

	if verbose {
		fmt.Printf("📚 已加载 %d 个知识库文档:\n", len(docs))
		for _, doc := range docs {
			fmt.Printf("   - %s\n", doc.Title)
		}
	}

	return nil
}
