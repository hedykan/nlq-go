package database

import (
	"fmt"
	"strings"
	"text/template"

	"gorm.io/gorm"
)

// TableSchema 表结构
type TableSchema struct {
	Name    string
	Columns []ColumnSchema
}

// ColumnSchema 列结构
type ColumnSchema struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
	Comment  string
}

// String 返回列的字符串表示
func (c ColumnSchema) String() string {
	nullStr := "NOT NULL"
	if c.Nullable {
		nullStr = "NULL"
	}

	commentStr := ""
	if c.Comment != "" {
		commentStr = fmt.Sprintf(" COMMENT '%s'", c.Comment)
	}

	defaultStr := ""
	if c.Default != "" {
		defaultStr = fmt.Sprintf(" DEFAULT %s", c.Default)
	}

	return fmt.Sprintf("%s %s %s%s%s", c.Name, c.Type, nullStr, defaultStr, commentStr)
}

// String 返回表的字符串表示
func (t TableSchema) String() string {
	var columns []string
	for _, col := range t.Columns {
		columns = append(columns, "  "+col.String())
	}
	return fmt.Sprintf("%s (\n%s\n)", t.Name, strings.Join(columns, ",\n"))
}

// SchemaParser Schema解析器
type SchemaParser struct {
	db *gorm.DB
}

// NewSchemaParser 创建Schema解析器
func NewSchemaParser(db *gorm.DB) *SchemaParser {
	return &SchemaParser{db: db}
}

// ParseSchema 解析数据库中所有表的结构
func (p *SchemaParser) ParseSchema() ([]TableSchema, error) {
	var tables []TableSchema

	// 获取所有表名
	var tableNames []string
	if err := p.db.Table("information_schema.tables").
		Select("table_name").
		Where("table_schema = ?", p.db.Migrator().CurrentDatabase()).
		Pluck("table_name", &tableNames).Error; err != nil {
		return nil, fmt.Errorf("获取表名失败: %w", err)
	}

	// 解析每个表的结构
	for _, tableName := range tableNames {
		table, err := p.ParseTable(tableName)
		if err != nil {
			// 跳过无法解析的表
			continue
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// ParseTable 解析特定表的结构
func (p *SchemaParser) ParseTable(tableName string) (TableSchema, error) {
	type ColumnInfo struct {
		ColumnName    string
		DataType      string
		IsNullable    string
		ColumnDefault string
		ColumnComment string
	}

	var columns []ColumnInfo
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			column_comment
		FROM information_schema.columns
		WHERE table_schema = ?
			AND table_name = ?
		ORDER BY ordinal_position
	`

	if err := p.db.Raw(query, p.db.Migrator().CurrentDatabase(), tableName).Scan(&columns).Error; err != nil {
		return TableSchema{}, fmt.Errorf("获取表 %s 的列信息失败: %w", tableName, err)
	}

	if len(columns) == 0 {
		return TableSchema{}, fmt.Errorf("表 %s 不存在或没有列", tableName)
	}

	var columnSchemas []ColumnSchema
	for _, col := range columns {
		columnSchema := ColumnSchema{
			Name:     col.ColumnName,
			Type:     col.DataType,
			Nullable: col.IsNullable == "YES",
			Default:  col.ColumnDefault,
			Comment:  col.ColumnComment,
		}
		columnSchemas = append(columnSchemas, columnSchema)
	}

	return TableSchema{
		Name:    tableName,
		Columns: columnSchemas,
	}, nil
}

// GetPrimaryKey 获取表的主键
func (p *SchemaParser) GetPrimaryKey(tableName string) (string, error) {
	type KeyInfo struct {
		ColumnName string
	}

	var keyInfo KeyInfo
	query := `
		SELECT column_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ?
			AND table_name = ?
			AND constraint_name = 'PRIMARY'
		LIMIT 1
	`

	if err := p.db.Raw(query, p.db.Migrator().CurrentDatabase(), tableName).Scan(&keyInfo).Error; err != nil {
		return "", fmt.Errorf("获取表 %s 的主键失败: %w", tableName, err)
	}

	if keyInfo.ColumnName == "" {
		return "", fmt.Errorf("表 %s 没有主键", tableName)
	}

	return keyInfo.ColumnName, nil
}

// FormatForPrompt 格式化Schema为Prompt
func (p *SchemaParser) FormatForPrompt() (string, error) {
	tables, err := p.ParseSchema()
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("数据库Schema:\n\n")

	for _, table := range tables {
		builder.WriteString(fmt.Sprintf("表: %s\n", table.Name))
		for _, col := range table.Columns {
			builder.WriteString(fmt.Sprintf("  - %s: %s", col.Name, col.Type))
			if col.Comment != "" {
				builder.WriteString(fmt.Sprintf(" // %s", col.Comment))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// FormatForPromptWithTemplate 使用模板格式化Schema为Prompt
func (p *SchemaParser) FormatForPromptWithTemplate(tmpl string) (string, error) {
	tables, err := p.ParseSchema()
	if err != nil {
		return "", err
	}

	parsedTemplate, err := template.New("schema").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("解析模板失败: %w", err)
	}

	data := struct {
		Tables []TableSchema
	}{
		Tables: tables,
	}

	var builder strings.Builder
	if err := parsedTemplate.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("执行模板失败: %w", err)
	}

	return builder.String(), nil
}

// FilterTables 过滤表
func (p *SchemaParser) FilterTables(tables []TableSchema, filter func(TableSchema) bool) []TableSchema {
	var filtered []TableSchema
	for _, table := range tables {
		if filter(table) {
			filtered = append(filtered, table)
		}
	}
	return filtered
}

// FindTable 查找特定表
func (p *SchemaParser) FindTable(tables []TableSchema, tableName string) (TableSchema, bool) {
	for _, table := range tables {
		if table.Name == tableName {
			return table, true
		}
	}
	return TableSchema{}, false
}

// GetTableCount 获取表的数量
func (p *SchemaParser) GetTableCount() (int, error) {
	var count int64
	if err := p.db.Table("information_schema.tables").
		Where("table_schema = ?", p.db.Migrator().CurrentDatabase()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取表数量失败: %w", err)
	}
	return int(count), nil
}

// ========== 两阶段Schema选择支持 ==========

// TableSummary 表摘要（用于阶段1的轻量级选择）
type TableSummary struct {
	Name       string   `json:"name"`
	Comment    string   `json:"comment"`
	RowCount   int64    `json:"row_count"`
	KeyColumns []string `json:"key_columns"` // 关键字段列表（用于字段级别上下文）
}

// ForeignKey 外键关系
type ForeignKey struct {
	Column      string `json:"column"`
	ReferTable  string `json:"refer_table"`
	ReferColumn string `json:"refer_column"`
}

// TableDetail 表详情（用于阶段2的完整Schema）
type TableDetail struct {
	Name       string          `json:"name"`
	Comment    string          `json:"comment"`
	Columns    []ColumnSchema  `json:"columns"`
	ForeignKeys []ForeignKey   `json:"foreign_keys"`
	PrimaryKey string          `json:"primary_key"`
}

// GetTableSummaries 获取所有表的摘要信息（阶段1使用）
func (p *SchemaParser) GetTableSummaries() ([]TableSummary, error) {
	type TableInfo struct {
		TableName      string
		TableComment   string
		TableRows      int64
	}

	var tableInfos []TableInfo
	query := `
		SELECT
			t.table_name,
			t.table_comment,
			COALESCE(s.table_rows, 0) as table_rows
		FROM information_schema.tables t
		LEFT JOIN (
			SELECT table_name, SUM(table_rows) as table_rows
			FROM information_schema.tables
			WHERE table_schema = DATABASE()
			GROUP BY table_name
		) s ON s.table_name = t.table_name
		WHERE t.table_schema = ?
		ORDER BY t.table_name
	`

	if err := p.db.Raw(query, p.db.Migrator().CurrentDatabase()).Scan(&tableInfos).Error; err != nil {
		return nil, fmt.Errorf("获取表摘要失败: %w", err)
	}

	var summaries []TableSummary
	for _, info := range tableInfos {
		summaries = append(summaries, TableSummary{
			Name:     info.TableName,
			Comment:  info.TableComment,
			RowCount: info.TableRows,
		})
	}

	return summaries, nil
}

// GetTableSummariesEnhanced 获取增强的表摘要信息（包含关键字段）
func (p *SchemaParser) GetTableSummariesEnhanced() ([]TableSummary, error) {
	type TableInfo struct {
		TableName      string
		TableComment   string
		TableRows      int64
	}

	var tableInfos []TableInfo
	query := `
		SELECT
			t.table_name,
			t.table_comment,
			COALESCE(s.table_rows, 0) as table_rows
		FROM information_schema.tables t
		LEFT JOIN (
			SELECT table_name, SUM(table_rows) as table_rows
			FROM information_schema.tables
			WHERE table_schema = DATABASE()
			GROUP BY table_name
		) s ON s.table_name = t.table_name
		WHERE t.table_schema = ?
		ORDER BY t.table_name
	`

	if err := p.db.Raw(query, p.db.Migrator().CurrentDatabase()).Scan(&tableInfos).Error; err != nil {
		return nil, fmt.Errorf("获取表摘要失败: %w", err)
	}

	var summaries []TableSummary
	for _, info := range tableInfos {
		// 获取表的详细结构以提取关键字段
		detail, err := p.GetTableDetail(info.TableName)
		if err != nil {
			// 如果获取详情失败，使用基本摘要
			summaries = append(summaries, TableSummary{
				Name:       info.TableName,
				Comment:    info.TableComment,
				RowCount:   info.TableRows,
				KeyColumns: []string{},
			})
			continue
		}

		// 提取关键字段
		keyColumns := p.extractKeyColumns(detail)

		summaries = append(summaries, TableSummary{
			Name:       info.TableName,
			Comment:    info.TableComment,
			RowCount:   info.TableRows,
			KeyColumns: keyColumns,
		})
	}

	return summaries, nil
}

// extractKeyColumns 从表详情中提取关键字段
func (p *SchemaParser) extractKeyColumns(detail TableDetail) []string {
	var keyColumns []string
	keyColumnSet := make(map[string]bool)

	// 1. 主键
	if detail.PrimaryKey != "" {
		keyColumnSet[detail.PrimaryKey] = true
	}

	// 2. 外键
	for _, fk := range detail.ForeignKeys {
		keyColumnSet[fk.Column] = true
	}

	// 3. 常见业务字段（按优先级排序）
	commonFields := []string{
		"id", "name", "username", "shop_name", "customer_name",
		"email", "phone", "mobile", "status", "level",
		"created_at", "updated_at", "deleted_at",
		"user_id", "customer_id", "order_id", "product_id",
		"title", "description", "content", "type",
		"price", "amount", "quantity", "total",
		"first_name", "last_name", "full_name",
	}

	for _, field := range commonFields {
		// 检查表中是否存在该字段
		for _, col := range detail.Columns {
			if col.Name == field && !keyColumnSet[col.Name] {
				keyColumnSet[col.Name] = true
				break
			}
		}
	}

	// 转换为切片并保持顺序（主键、外键、常见字段）
	for _, col := range detail.Columns {
		if keyColumnSet[col.Name] {
			keyColumns = append(keyColumns, col.Name)
		}
	}

	// 限制关键字段数量（避免token过多）
	if len(keyColumns) > 10 {
		keyColumns = keyColumns[:10]
	}

	return keyColumns
}

// GetTableDetail 获取单个表的详细信息（阶段2使用）
func (p *SchemaParser) GetTableDetail(tableName string) (TableDetail, error) {
	// 获取基本表结构
	tableSchema, err := p.ParseTable(tableName)
	if err != nil {
		return TableDetail{}, err
	}

	// 获取主键
	pk, _ := p.GetPrimaryKey(tableName)

	// 获取外键关系
	fks, _ := p.GetForeignKeys(tableName)

	// 获取表注释
	var tableComment string
	p.db.Raw(`
		SELECT table_comment
		FROM information_schema.tables
		WHERE table_schema = ? AND table_name = ?
	`, p.db.Migrator().CurrentDatabase(), tableName).Scan(&tableComment)

	return TableDetail{
		Name:       tableSchema.Name,
		Comment:    tableComment,
		Columns:    tableSchema.Columns,
		ForeignKeys: fks,
		PrimaryKey: pk,
	}, nil
}

// GetForeignKeys 获取表的外键关系
func (p *SchemaParser) GetForeignKeys(tableName string) ([]ForeignKey, error) {
	type FkInfo struct {
		ColumnName       string
		ReferencedTable  string
		ReferencedColumn string
	}

	var fkInfos []FkInfo
	query := `
		SELECT
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ?
			AND tc.table_name = ?
	`

	if err := p.db.Raw(query, p.db.Migrator().CurrentDatabase(), tableName).Scan(&fkInfos).Error; err != nil {
		return nil, fmt.Errorf("获取外键关系失败: %w", err)
	}

	var fks []ForeignKey
	for _, fk := range fkInfos {
		fks = append(fks, ForeignKey{
			Column:      fk.ColumnName,
			ReferTable:  fk.ReferencedTable,
			ReferColumn: fk.ReferencedColumn,
		})
	}

	return fks, nil
}

// FormatTablesForPrompt 格式化指定表的Schema（用于阶段2）
func (p *SchemaParser) FormatTablesForPrompt(tableNames []string) (string, error) {
	var builder strings.Builder

	builder.WriteString("# 数据库Schema\n\n")

	for i, tableName := range tableNames {
		detail, err := p.GetTableDetail(tableName)
		if err != nil {
			continue // 跳过无法获取的表
		}

		builder.WriteString(fmt.Sprintf("## 表 %d: %s\n", i+1, detail.Name))
		if detail.Comment != "" {
			builder.WriteString(fmt.Sprintf("说明: %s\n", detail.Comment))
		}

		builder.WriteString("\n### 字段列表\n")
		for _, col := range detail.Columns {
			builder.WriteString(fmt.Sprintf("- %s %s", col.Name, col.Type))
			if !col.Nullable {
				builder.WriteString(" NOT NULL")
			}
			if col.Comment != "" {
				builder.WriteString(fmt.Sprintf(" -- %s", col.Comment))
			}
			builder.WriteString("\n")
		}

		if len(detail.ForeignKeys) > 0 {
			builder.WriteString("\n### 关联关系\n")
			for _, fk := range detail.ForeignKeys {
				builder.WriteString(fmt.Sprintf("- %s -> %s.%s\n", fk.Column, fk.ReferTable, fk.ReferColumn))
			}
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}
