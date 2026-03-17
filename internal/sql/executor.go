package sql

import (
	"context"
	"fmt"
	"time"

	"github.com/channelwill/nlq/pkg/security"
	"gorm.io/gorm"
)

// Executor SQL执行器
type Executor struct {
	db       *gorm.DB
	firewall *security.Firewall
}

// NewExecutor 创建SQL执行器
func NewExecutor(db *gorm.DB) *Executor {
	return &Executor{
		db:       db,
		firewall: security.NewFirewall(),
	}
}

// ExecuteResult 执行SQL并返回结果
type ExecuteResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int              `json:"count"`
	Duration time.Duration   `json:"duration"`
}

// Execute 执行SQL查询
func (e *Executor) Execute(ctx context.Context, sqlQuery string) (*ExecuteResult, error) {
	start := time.Now()

	// 1. 安全检查
	if err := e.firewall.Check(sqlQuery); err != nil {
		return nil, fmt.Errorf("SQL安全检查失败: %w", err)
	}

	// 2. 执行查询
	rows, err := e.db.WithContext(ctx).Raw(sqlQuery).Rows()
	if err != nil {
		return nil, fmt.Errorf("执行SQL失败: %w", err)
	}
	defer rows.Close()

	// 3. 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}

	// 4. 读取结果
	var resultRows []map[string]interface{}
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// 扫描行
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}

		// 构建结果映射
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 处理NULL值
			if val == nil {
				row[col] = nil
			} else {
				// 转换字节数组为字符串
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	// 5. 构建结果
	result := &ExecuteResult{
		Columns: columns,
		Rows:    resultRows,
		Count:   len(resultRows),
		Duration: time.Since(start),
	}

	return result, nil
}

// ExecuteSimple 简单执行（返回第一行第一列）
func (e *Executor) ExecuteSimple(ctx context.Context, sqlQuery string) (interface{}, error) {
	// 1. 安全检查
	if err := e.firewall.Check(sqlQuery); err != nil {
		return nil, fmt.Errorf("SQL安全检查失败: %w", err)
	}

	// 2. 执行查询
	var result interface{}
	if err := e.db.WithContext(ctx).Raw(sqlQuery).Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("执行SQL失败: %w", err)
	}

	return result, nil
}

// ExecuteCount 执行COUNT查询
func (e *Executor) ExecuteCount(ctx context.Context, sqlQuery string) (int64, error) {
	// 1. 安全检查
	if err := e.firewall.Check(sqlQuery); err != nil {
		return 0, fmt.Errorf("SQL安全检查失败: %w", err)
	}

	// 2. 执行查询
	var count int64
	if err := e.db.WithContext(ctx).Raw(sqlQuery).Scan(&count).Error; err != nil {
		return 0, fmt.Errorf("执行COUNT查询失败: %w", err)
	}

	return count, nil
}

// ValidateOnly 仅验证SQL安全性，不执行
func (e *Executor) ValidateOnly(sqlQuery string) error {
	return e.firewall.Check(sqlQuery)
}

// GetFirewall 获取防火墙实例
func (e *Executor) GetFirewall() *security.Firewall {
	return e.firewall
}
