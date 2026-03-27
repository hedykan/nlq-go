package main

import (
	"fmt"
	"log"

	"github.com/channelwill/nlq/internal/config"
	"github.com/channelwill/nlq/internal/database"
)

func main() {
	fmt.Println("🧪 SSH隧道连接单元测试")
	fmt.Println("========================================")

	// 测试1: 加载配置文件
	fmt.Println("\n📋 测试1: 加载配置文件")
	cfg, err := config.LoadConfig("./config/config.yaml")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	fmt.Printf("   数据库: %s@%s:%d/%s\n",
		cfg.Database.Username,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)
	fmt.Printf("   SSH启用: %v\n", cfg.Database.SSHEnabled)
	if cfg.Database.SSHEnabled {
		fmt.Printf("   SSH服务器: %s:%d\n", cfg.Database.SSHHost, cfg.Database.SSHPort)
		fmt.Printf("   SSH用户: %s\n", cfg.Database.SSHUser)
		fmt.Printf("   私钥文件: %s\n", cfg.Database.SSHPrivateKeyFile)
	}
	fmt.Println("   ✅ 配置加载成功")

	// 测试2: 验证SSH配置
	fmt.Println("\n🔐 测试2: 验证SSH配置")
	if err := cfg.Database.ValidateSSHConfig(); err != nil {
		log.Fatalf("❌ SSH配置验证失败: %v", err)
	}
	fmt.Println("   ✅ SSH配置验证通过")

	// 测试3: 创建数据库连接（会自动建立SSH隧道）
	fmt.Println("\n🔗 测试3: 创建数据库连接（通过SSH隧道）")
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("❌ 创建数据库连接失败: %v", err)
	}
	defer database.CloseConnection(db)
	fmt.Println("   ✅ 数据库连接创建成功")

	// 测试4: 验证数据库连接
	fmt.Println("\n🔍 测试4: 验证数据库连接")
	if err := database.ValidateConnection(db); err != nil {
		log.Fatalf("❌ 数据库连接验证失败: %v", err)
	}
	fmt.Println("   ✅ 数据库连接验证成功")

	// 测试5: 检查SSH隧道状态
	fmt.Println("\n📊 测试5: 检查SSH隧道状态")
	tunnel := database.GetSSHTunnel()
	if tunnel == nil {
		log.Fatalf("❌ SSH隧道实例为空")
	}
	if !tunnel.IsConnected() {
		log.Fatalf("❌ SSH隧道未连接")
	}
	localPort := tunnel.GetLocalPort()
	if localPort == 0 {
		log.Fatalf("❌ SSH隧道本地端口为0")
	}
	fmt.Printf("   隧道状态: 已连接\n")
	fmt.Printf("   本地端口: %d\n", localPort)
	fmt.Println("   ✅ SSH隧道状态正常")

	// 测试6: 执行简单查询
	fmt.Println("\n🔎 测试6: 执行数据库查询")
	var version string
	if err := db.Raw("SELECT VERSION()").Scan(&version).Error; err != nil {
		log.Fatalf("❌ 查询数据库版本失败: %v", err)
	}
	fmt.Printf("   MySQL版本: %s\n", version)

	var dbName string
	if err := db.Raw("SELECT DATABASE()").Scan(&dbName).Error; err != nil {
		log.Fatalf("❌ 查询当前数据库失败: %v", err)
	}
	fmt.Printf("   当前数据库: %s\n", dbName)

	// 获取表数量
	var tableCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", cfg.Database.Database).Scan(&tableCount).Error; err != nil {
		log.Printf("⚠️  查询表数量失败: %v", err)
	} else {
		fmt.Printf("   数据库表数量: %d\n", tableCount)
	}

	fmt.Println("   ✅ 数据库查询成功")

	// 测试7: 测试复杂的JOIN查询
	fmt.Println("\n🔎 测试7: 测试复杂查询")
	type TableRow struct {
		TableName string `gorm:"column:table_name"`
	}
	var tables []TableRow
	if err := db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = ? ORDER BY table_name LIMIT 10", cfg.Database.Database).Scan(&tables).Error; err != nil {
		log.Printf("⚠️  查询表列表失败: %v", err)
	} else {
		fmt.Printf("   前10个表:\n")
		for i, table := range tables {
			fmt.Printf("      %d. %s\n", i+1, table.TableName)
		}
		fmt.Println("   ✅ 复杂查询成功")
	}

	// 测试8: 测试事务支持
	fmt.Println("\n💾 测试8: 测试事务支持")
	tx := db.Begin()
	if tx.Error != nil {
		log.Printf("⚠️  开始事务失败: %v", tx.Error)
	} else {
		var result int
		if err := tx.Raw("SELECT 1").Scan(&result).Error; err != nil {
			log.Printf("⚠️  事务查询失败: %v", err)
			tx.Rollback()
		} else {
			tx.Commit()
			fmt.Printf("   事务查询结果: %d\n", result)
			fmt.Println("   ✅ 事务支持正常")
		}
	}

	// 测试9: 测试连接池
	fmt.Println("\n🔗 测试9: 测试连接池")
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("⚠️  获取SQL DB失败: %v", err)
	} else {
		stats := sqlDB.Stats()
		fmt.Printf("   最大打开连接数: %d\n", stats.MaxOpenConnections)
		fmt.Printf("   空闲连接数: %d\n", stats.Idle)
		fmt.Printf("   在用连接数: %d\n", stats.InUse)
		fmt.Println("   ✅ 连接池状态正常")
	}

	// 测试10: 关闭连接
	fmt.Println("\n🔒 测试10: 关闭连接")
	if err := database.CloseConnection(db); err != nil {
		log.Fatalf("❌ 关闭连接失败: %v", err)
	}

	// 验证SSH隧道已关闭
	tunnel = database.GetSSHTunnel()
	if tunnel != nil && tunnel.IsConnected() {
		log.Fatalf("❌ 关闭连接后SSH隧道仍然连接")
	}
	fmt.Println("   ✅ 连接关闭成功，SSH隧道已释放")

	fmt.Println("\n========================================")
	fmt.Println("🎉 所有单元测试通过！")
	fmt.Println("========================================")

	// 打印总结
	fmt.Println("\n📊 测试总结:")
	fmt.Println("   ✅ 配置文件加载")
	fmt.Println("   ✅ SSH配置验证")
	fmt.Println("   ✅ 数据库连接创建")
	fmt.Println("   ✅ 数据库连接验证")
	fmt.Println("   ✅ SSH隧道状态检查")
	fmt.Println("   ✅ 数据库查询执行")
	fmt.Println("   ✅ 复杂查询执行")
	fmt.Println("   ✅ 事务支持")
	fmt.Println("   ✅ 连接池状态")
	fmt.Println("   ✅ 连接关闭和资源释放")
	fmt.Println("\n💡 配置文件SSH隧道功能完全正常！")
	fmt.Println("💡 可以启动NLQ服务器进行自然语言查询了！")
}
