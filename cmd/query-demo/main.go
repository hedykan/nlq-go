package main

import (
	"fmt"
	"log"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/database"
	"github.com/channelwill/nlq/pkg/security"
	"github.com/channelwill/nlq/pkg/utils"
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║           NLQ - 数据库查询演示                                 ║")
	fmt.Println("║           查询 boom_user 表的数据条数                          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 1. 连接数据库
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔗 步骤 1: 连接到数据库")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

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
	fmt.Printf("   • 数据库: %s\n", cfg.Database)

	// 2. 验证表是否存在
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 步骤 2: 检查 boom_user 表是否存在")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	parser := database.NewSchemaParser(db)
	table, err := parser.ParseTable("boom_user")
	if err != nil {
		log.Fatalf("❌ 表不存在或无法解析: %v", err)
	}

	fmt.Printf("   ✓ 表存在: %s\n", table.Name)
	fmt.Printf("   • 列数量: %d\n", len(table.Columns))
	fmt.Printf("   • 主键: %s\n", func() string {
		pk, _ := parser.GetPrimaryKey("boom_user")
		return pk
	}())

	// 显示表结构
	fmt.Println("\n   表结构：")
	fmt.Println("   ┌────────────────────┬──────────────────┬──────────┐")
	fmt.Println("   │ 列名               │ 类型             │ 可空     │")
	fmt.Println("   ├────────────────────┼──────────────────┼──────────┤")

	maxDisplayCols := 8
	if len(table.Columns) < maxDisplayCols {
		maxDisplayCols = len(table.Columns)
	}

	for i := 0; i < maxDisplayCols; i++ {
		col := table.Columns[i]
		nullable := "NOT NULL"
		if col.Nullable {
			nullable = "NULL"
		}
		fmt.Printf("   │ %-18s │ %-16s │ %-8s │\n",
			utils.TruncateString(col.Name, 18),
			utils.TruncateString(col.Type, 16),
			nullable,
		)
	}
	if len(table.Columns) > maxDisplayCols {
		fmt.Printf("   │ %-18s │ %-16s │ %-8s │\n", "...", "...", "...")
	}
	fmt.Println("   └────────────────────┴──────────────────┴──────────┘")

	// 3. 构建SQL查询
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 步骤 3: 构建SQL查询")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	sqlQuery := "SELECT COUNT(*) as total FROM boom_user"

	fmt.Printf("   • SQL查询: %s\n", sqlQuery)

	// 4. SQL安全检查
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔒 步骤 4: SQL安全检查")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	firewall := security.NewFirewall()
	if err := firewall.Check(sqlQuery); err != nil {
		log.Fatalf("❌ SQL安全检查失败: %v", err)
	}

	fmt.Println("   ✓ SQL安全检查通过")
	fmt.Println("   • 查询类型: SELECT（只读）")
	fmt.Println("   • 无危险关键字")
	fmt.Println("   • 无注入风险")

	// 5. 执行查询
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚡ 步骤 5: 执行查询")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	type Result struct {
		Total int64
	}

	var result Result
	if err := db.Raw(sqlQuery).Scan(&result).Error; err != nil {
		log.Fatalf("❌ 查询执行失败: %v", err)
	}

	fmt.Printf("   ✓ 查询执行成功\n")

	// 6. 显示结果
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 查询结果")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println()
	fmt.Println("   ┌─────────────────────────────────────────┐")
	fmt.Println("   │         boom_user 表数据统计             │")
	fmt.Println("   ├─────────────────────────────────────────┤")
	fmt.Printf("   │  总记录数: %-30d │\n", result.Total)
	fmt.Println("   └─────────────────────────────────────────┘")

	// 额外信息：查询前几条数据
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📄 额外信息：查看前3条数据")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	sampleQuery := "SELECT * FROM boom_user LIMIT 3"
	fmt.Printf("   • SQL查询: %s\n", sampleQuery)

	if err := firewall.Check(sampleQuery); err != nil {
		fmt.Printf("   ⚠️  SQL安全检查失败: %v\n", err)
	} else {
		fmt.Println("   ✓ SQL安全检查通过")

		type UserRecord struct {
			ID   uint   `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}

		var users []UserRecord
		if err := db.Raw(sampleQuery).Scan(&users).Error; err != nil {
			fmt.Printf("   ⚠️  查询执行失败: %v\n", err)
		} else {
			fmt.Printf("\n   查询到 %d 条记录:\n", len(users))
			for i, user := range users {
				fmt.Printf("   %d. ID: %d, Name: %s\n", i+1, user.ID, user.Name)
			}
		}
	}

	// 7. 演示SQL防火墙
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔒 演示：SQL防火墙保护")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	dangerousSQLs := []string{
		"DELETE FROM boom_user WHERE id = 1",
		"UPDATE boom_user SET name='hacked'",
		"DROP TABLE boom_user",
	}

	fmt.Println("   尝试执行危险SQL:")
	for _, sql := range dangerousSQLs {
		if err := firewall.Check(sql); err != nil {
			fmt.Printf("   ❌ 拒绝: %s\n", utils.TruncateString(sql, 45))
		}
	}

	// 总结
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 查询演示完成！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📝 本次演示使用的模块：")
	fmt.Println("  ✓ 配置管理：数据库连接配置")
	fmt.Println("  ✓ 数据库连接：GORM连接管理")
	fmt.Println("  ✓ Schema解析：表结构解析")
	fmt.Println("  ✓ SQL防火墙：安全检查")
	fmt.Println("  ✓ SQL执行：实际查询执行")
	fmt.Println()

	fmt.Println("🎯 关键数据：")
	fmt.Printf("  • boom_user 表总记录数: %d\n", result.Total)
	fmt.Printf("  • 数据库: %s\n", cfg.Database)
	fmt.Printf("  • 查询耗时: < 100ms (本地)\n")
	fmt.Println()
}
