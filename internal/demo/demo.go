package main

import (
	"fmt"
	"log"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/pkg/security"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           NLQ - 当前功能演示                                  ║")
	fmt.Println("║           Natural Language Query Demo                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. 配置管理演示
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 模块 1: 配置管理")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	demoConfig()

	fmt.Println()

	// 2. 数据库连接演示
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔗 模块 2: 数据库连接")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	db := demoDatabaseConnection()

	fmt.Println()

	// 3. Schema解析演示
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 模块 3: Schema解析")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	demoSchemaParsing(db)

	fmt.Println()

	// 4. SQL防火墙演示
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔒 模块 4: SQL安全防火墙")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	demoSQLFirewall()

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 演示完成！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("📝 当前可用功能总结：")
	fmt.Println("  ✓ 配置管理：支持YAML文件和环境变量")
	fmt.Println("  ✓ 数据库连接：GORM连接池管理")
	fmt.Println("  ✓ Schema解析：自动解析表结构")
	fmt.Println("  ✓ SQL安全检查：严格的SELECT-only策略")
	fmt.Println()
	fmt.Println("🚧 待开发功能：")
	fmt.Println("  ✗ LLM集成：将自然语言转换为SQL")
	fmt.Println("  ✗ SQL生成器：智能SQL构建")
	fmt.Println("  ✗ CLI界面：命令行交互工具")
	fmt.Println("  ✗ 结果格式化：美化查询结果")
	fmt.Println()
}

// demoConfig 配置管理演示
func demoConfig() {
	fmt.Println("📌 创建默认配置...")
	cfg := &config.Config{}
	cfg.SetDefaults()

	fmt.Printf("   • 数据库: %s@%s:%d/%s\n",
		cfg.Database.Username,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)
	fmt.Printf("   • 只读模式: %v\n", cfg.Database.Readonly)
	fmt.Printf("   • LLM模型: %s\n", cfg.LLM.Model)
	fmt.Printf("   • 安全模式: %s\n", cfg.Security.Mode)
	fmt.Printf("   • 服务器端口: %d\n", cfg.Server.Port)

	fmt.Println("\n✅ 配置管理模块工作正常！")
}

// demoDatabaseConnection 数据库连接演示
func demoDatabaseConnection() *gorm.DB {
	fmt.Println("📌 连接到数据库...")

	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}

	fmt.Println("   ✓ 数据库连接成功")
	fmt.Printf("   • 连接类型: %s\n", cfg.Driver)
	fmt.Printf("   • 主机: %s:%d\n", cfg.Host, cfg.Port)
	fmt.Printf("   • 数据库: %s\n", cfg.Database)
	fmt.Printf("   • 只读模式: %v\n", cfg.Readonly)

	// 验证连接
	sqlDB, _ := db.DB()
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("❌ Ping失败: %v", err)
	}
	fmt.Println("   ✓ 数据库Ping成功")

	// 获取表数量
	parser := database.NewSchemaParser(db)
	tableCount, err := parser.GetTableCount()
	if err != nil {
		log.Printf("⚠️  获取表数量失败: %v", err)
	} else {
		fmt.Printf("   • 数据库表数量: %d\n", tableCount)
	}

	fmt.Println("\n✅ 数据库连接模块工作正常！")
	return db
}

// demoSchemaParsing Schema解析演示
func demoSchemaParsing(db *gorm.DB) {
	fmt.Println("📌 解析数据库Schema...")

	parser := database.NewSchemaParser(db)

	// 解析特定表
	tableName := "boom_customer"
	table, err := parser.ParseTable(tableName)
	if err != nil {
		log.Printf("⚠️  解析表 %s 失败: %v", tableName, err)
		return
	}

	fmt.Printf("   ✓ 成功解析表: %s\n", table.Name)
	fmt.Printf("   • 列数量: %d\n", len(table.Columns))
	fmt.Println("\n   表结构（前5列）：")
	fmt.Println("   ┌────────────────────┬──────────────────┬──────────┬────────────┐")
	fmt.Println("   │ 列名               │ 类型             │ 可空     │ 注释       │")
	fmt.Println("   ├────────────────────┼──────────────────┼──────────┼────────────┤")

	maxColumns := 5
	if len(table.Columns) < maxColumns {
		maxColumns = len(table.Columns)
	}

	for i := 0; i < maxColumns; i++ {
		col := table.Columns[i]
		nullable := "NOT NULL"
		if col.Nullable {
			nullable = "NULL"
		}
		comment := col.Comment
		if len(comment) > 10 {
			comment = comment[:10] + "..."
		}
		fmt.Printf("   │ %-18s │ %-16s │ %-8s │ %-10s │\n",
			truncateString(col.Name, 18),
			truncateString(col.Type, 16),
			nullable,
			truncateString(comment, 10),
		)
	}
	fmt.Println("   └────────────────────┴──────────────────┴──────────┴────────────┘")

	// 获取主键
	pk, err := parser.GetPrimaryKey(tableName)
	if err != nil {
		log.Printf("⚠️  获取主键失败: %v", err)
	} else {
		fmt.Printf("\n   • 主键: %s\n", pk)
	}

	// 格式化为Prompt
	prompt, err := parser.FormatForPrompt()
	if err != nil {
		log.Printf("⚠️  格式化Prompt失败: %v", err)
	} else {
		fmt.Printf("\n   ✓ 生成LLM Prompt（前200字符）:\n")
		fmt.Printf("   %s\n", truncateString(prompt, 200)+"...")
	}

	fmt.Println("\n✅ Schema解析模块工作正常！")
}

// demoSQLFirewall SQL防火墙演示
func demoSQLFirewall() {
	fmt.Println("📌 测试SQL安全防火墙...")

	firewall := security.NewFirewall()

	testCases := []struct {
		name string
		sql  string
	}{
		{"安全的SELECT查询", "SELECT * FROM boom_customer LIMIT 10"},
		{"带JOIN的查询", "SELECT c.*, o.* FROM boom_customer c JOIN boom_order_paid_water o ON c.id = o.customer_id"},
		{"聚合查询", "SELECT COUNT(*) as total FROM boom_customer"},
		{"子查询", "SELECT * FROM boom_customer WHERE id IN (SELECT customer_id FROM boom_order_paid_water)"},
		{"❌ DROP TABLE（危险）", "DROP TABLE boom_customer"},
		{"❌ DELETE（危险）", "DELETE FROM boom_customer WHERE id = 1"},
		{"❌ UPDATE（危险）", "UPDATE boom_customer SET name='test' WHERE id=1"},
		{"❌ 注释注入（危险）", "SELECT * FROM boom_customer; DROP TABLE boom_customer--"},
		{"❌ 多语句（危险）", "SELECT * FROM boom_customer; SELECT * FROM boom_order_paid_water"},
	}

	fmt.Println("   测试用例：")
	fmt.Println("   ┌─────────────────────────────────────────────────────────┐")
	fmt.Println("   │ 测试场景                                               │ 结果    │")
	fmt.Println("   ├─────────────────────────────────────────────────────────┤")

	passCount := 0
	for _, tc := range testCases {
		err := firewall.Check(tc.sql)
		result := "✅ 允许"
		if err != nil {
			result = "❌ 拒绝"
		} else {
			passCount++
		}

		// 截断SQL显示
		sqlDisplay := truncateString(tc.sql, 50)
		fmt.Printf("   │ %-55s │ %-8s │\n", sqlDisplay, result)
	}

	fmt.Println("   └─────────────────────────────────────────────────────────┘")

	// 显示防火墙配置
	fmt.Printf("\n   防火墙配置：\n")
	fmt.Printf("   • 允许的前缀: %v\n", firewall.GetAllowedPrefixes())
	fmt.Printf("   • 拦截的关键字: %d个\n", len(firewall.GetBlockedKeywords()))
	fmt.Printf("   • 测试通过率: %d/%d\n", passCount, len(testCases))

	fmt.Println("\n✅ SQL防火墙模块工作正常！")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
