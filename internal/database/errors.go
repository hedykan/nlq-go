package database

import "fmt"

// DatabaseError 数据库错误类型
type DatabaseError struct {
	Op      string // 操作类型
	SSH     bool   // 是否是SSH相关错误
	Err     error  // 原始错误
	Message string // 用户友好的错误消息
}

// Error 实现error接口
func (e *DatabaseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Op, e.Message)
}

// Unwrap 实现错误包装接口
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// NewSSHError 创建SSH相关错误
func NewSSHError(op string, err error, message string) *DatabaseError {
	return &DatabaseError{
		Op:      op,
		SSH:     true,
		Err:     err,
		Message: message,
	}
}

// NewConnectionError 创建连接错误
func NewConnectionError(op string, err error, message string) *DatabaseError {
	return &DatabaseError{
		Op:      op,
		SSH:     false,
		Err:     err,
		Message: message,
	}
}
