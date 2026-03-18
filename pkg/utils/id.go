package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateQueryID 生成查询唯一标识
// 格式: qry_{YYYYMMDD}_{随机8位字符}
func GenerateQueryID() string {
	date := time.Now().Format("20060102")
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		// 如果随机生成失败，使用时间戳
		return fmt.Sprintf("qry_%s_%d", date, time.Now().UnixNano()%100000000)
	}
	randomStr := hex.EncodeToString(randomBytes)[:8]
	return fmt.Sprintf("qry_%s_%s", date, randomStr)
}

// GenerateFeedbackID 生成反馈唯一标识
// 格式: fb_{16位十六进制字符}
func GenerateFeedbackID() string {
	// 生成8字节的随机数
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，回退到时间戳+计数器的方式
		return fmt.Sprintf("fb_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
	}
	// 转换为16字符的十六进制字符串
	return fmt.Sprintf("fb_%s", hex.EncodeToString(b))
}
