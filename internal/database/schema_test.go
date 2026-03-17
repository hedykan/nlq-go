package database

import (
	"strings"
	"testing"

	"github.com/channelwill/nlq/internal/config"
)

// TestSchemaParser_ParseSchema 测试解析数据库Schema
func TestSchemaParser_ParseSchema(t *testing.T) {
	// 创建数据库连接
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建Schema解析器
	parser := NewSchemaParser(db)

	// 解析Schema
	tables, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("解析Schema失败: %v", err)
	}

	// 验证至少有一些表
	if len(tables) == 0 {
		t.Error("期望至少有一个表")
	}

	// 验证表结构
	for _, table := range tables {
		if table.Name == "" {
			t.Error("表名不能为空")
		}
		if len(table.Columns) == 0 {
			t.Errorf("表 %s 应该至少有一列", table.Name)
		}
		for _, col := range table.Columns {
			if col.Name == "" {
				t.Errorf("表 %s 的列名不能为空", table.Name)
			}
			if col.Type == "" {
				t.Errorf("表 %s 的列 %s 类型不能为空", table.Name, col.Name)
			}
		}
	}
}

// TestSchemaParser_ParseSpecificTable 测试解析特定表
func TestSchemaParser_ParseSpecificTable(t *testing.T) {
	// 创建数据库连接
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建Schema解析器
	parser := NewSchemaParser(db)

	// 解析特定表
	table, err := parser.ParseTable("boom_customer")
	if err != nil {
		t.Fatalf("解析表 boom_customer 失败: %v", err)
	}

	// 验证表名
	if table.Name != "boom_customer" {
		t.Errorf("期望表名为 boom_customer, 实际为 %s", table.Name)
	}

	// 验证列
	if len(table.Columns) == 0 {
		t.Error("期望 boom_customer 表至少有一列")
	}

	// 验证一些已知的列
	columnMap := make(map[string]ColumnSchema)
	for _, col := range table.Columns {
		columnMap[col.Name] = col
	}

	// 检查一些已知的列名
	expectedColumns := []string{"id", "customer_rid", "first_name", "last_name", "email"}
	for _, expectedCol := range expectedColumns {
		if _, exists := columnMap[expectedCol]; !exists {
			t.Errorf("期望表中有列: %s", expectedCol)
		}
	}
}

// TestSchemaParser_ParseTable_NotFound 测试解析不存在的表
func TestSchemaParser_ParseTable_NotFound(t *testing.T) {
	// 创建数据库连接
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建Schema解析器
	parser := NewSchemaParser(db)

	// 尝试解析不存在的表
	_, err = parser.ParseTable("non_existent_table")
	if err == nil {
		t.Error("期望返回错误，但返回了 nil")
	}
}

// TestSchemaParser_FormatForPrompt 测试格式化Schema为Prompt
func TestSchemaParser_FormatForPrompt(t *testing.T) {
	// 创建数据库连接
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建Schema解析器
	parser := NewSchemaParser(db)

	// 格式化Schema
	prompt, err := parser.FormatForPrompt()
	if err != nil {
		t.Fatalf("格式化Schema失败: %v", err)
	}

	// 验证Prompt不为空
	if prompt == "" {
		t.Error("期望Prompt不为空")
	}

	// 验证Prompt包含一些关键信息
	if !strings.Contains(prompt, "boom_customer") {
		t.Error("期望Prompt包含表名 boom_customer")
	}
}

// TestSchemaParser_FilterTables 测试过滤表
func TestSchemaParser_FilterTables(t *testing.T) {
	// 创建数据库连接
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建Schema解析器
	parser := NewSchemaParser(db)

	// 解析Schema
	tables, err := parser.ParseSchema()
	if err != nil {
		t.Fatalf("解析Schema失败: %v", err)
	}

	// 过滤只包含customer的表
	filtered := parser.FilterTables(tables, func(table TableSchema) bool {
		return strings.Contains(table.Name, "customer")
	})

	// 验证过滤结果
	if len(filtered) == 0 {
		t.Error("期望过滤后有表")
	}

	for _, table := range filtered {
		if !strings.Contains(table.Name, "customer") {
			t.Errorf("期望过滤后的表名包含customer, 实际为 %s", table.Name)
		}
	}
}

// TestColumnSchema_String 测试列Schema的字符串表示
func TestColumnSchema_String(t *testing.T) {
	column := ColumnSchema{
		Name:     "test_column",
		Type:     "VARCHAR(255)",
		Nullable: true,
		Comment:  "测试列",
	}

	expected := "test_column VARCHAR(255) NULL COMMENT '测试列'"
	actual := column.String()

	if actual != expected {
		t.Errorf("列字符串表示不匹配\n期望: %s\n实际: %s", expected, actual)
	}
}

// TestTableSchema_String 测试表Schema的字符串表示
func TestTableSchema_String(t *testing.T) {
	table := TableSchema{
		Name: "test_table",
		Columns: []ColumnSchema{
			{
				Name:     "id",
				Type:     "INT",
				Nullable: false,
				Comment:  "主键",
			},
			{
				Name:     "name",
				Type:     "VARCHAR(255)",
				Nullable: true,
				Comment:  "名称",
			},
		},
	}

	expected := "test_table (\n  id INT NOT NULL COMMENT '主键',\n  name VARCHAR(255) NULL COMMENT '名称'\n)"
	actual := table.String()

	if actual != expected {
		t.Errorf("表字符串表示不匹配\n期望: %s\n实际: %s", expected, actual)
	}
}

// TestSchemaParser_GetPrimaryKey 测试获取主键
func TestSchemaParser_GetPrimaryKey(t *testing.T) {
	// 创建数据库连接
	cfg := &config.DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "loloyal",
		Username: "root",
		Password: "root",
		Readonly: true,
	}

	db, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建Schema解析器
	parser := NewSchemaParser(db)

	// 获取主键
	pk, err := parser.GetPrimaryKey("boom_customer")
	if err != nil {
		t.Fatalf("获取主键失败: %v", err)
	}

	// 验证主键
	if pk != "id" {
		t.Errorf("期望主键为 id, 实际为 %s", pk)
	}
}
