package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/channelwill/nlq/internal/handler"
)

// SuggestionsHandler 示例问题处理器
type SuggestionsHandler struct {
	queryHandler handler.QueryHandlerInterface
	cache        *suggestionsCache
}

// suggestionsCache 示例问题缓存
type suggestionsCache struct {
	mu        sync.RWMutex
	data      []string
	timestamp time.Time
	ttl       time.Duration
}

// newSuggestionsCache 创建缓存
func newSuggestionsCache(ttl time.Duration) *suggestionsCache {
	return &suggestionsCache{
		ttl: ttl,
	}
}

// get 获取缓存的示例问题
func (c *suggestionsCache) get() ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data == nil || time.Since(c.timestamp) > c.ttl {
		return nil, false
	}
	return c.data, true
}

// set 设置缓存的示例问题
func (c *suggestionsCache) set(data []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = data
	c.timestamp = time.Now()
}

// clear 清空缓存
func (c *suggestionsCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = nil
	c.timestamp = time.Time{}
}

// NewSuggestionsHandler 创建示例问题处理器
func NewSuggestionsHandler(queryHandler handler.QueryHandlerInterface) *SuggestionsHandler {
	return &SuggestionsHandler{
		queryHandler: queryHandler,
		cache:        newSuggestionsCache(30 * time.Minute), // 缓存30分钟
	}
}

// SuggestionsResponse 示例问题响应
type SuggestionsResponse struct {
	Success      bool     `json:"success"`
	Suggestions  []string `json:"suggestions"`
	Cached       bool     `json:"cached"`
	GeneratedAt  string   `json:"generated_at,omitempty"`
}

// HandleSuggestions 处理示例问题请求
func (h *SuggestionsHandler) HandleSuggestions(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	h.setCORSHeaders(w)

	// 处理OPTIONS请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许GET请求
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "方法不允许", "method_not_allowed", http.StatusMethodNotAllowed)
		return
	}

	// 尝试从缓存获取
	if suggestions, cached := h.cache.get(); cached {
		response := SuggestionsResponse{
			Success:     true,
			Suggestions: suggestions,
			Cached:      true,
		}
		h.sendJSONResponse(w, response, http.StatusOK)
		return
	}

	// 缓存未命中，生成新的示例问题
	suggestions := h.generateSmartSuggestions()

	// 保存到缓存
	h.cache.set(suggestions)

	// 返回响应
	response := SuggestionsResponse{
		Success:     true,
		Suggestions: suggestions,
		Cached:      false,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	h.sendJSONResponse(w, response, http.StatusOK)
}

// generateSmartSuggestions 智能生成示例问题
// 基于常见数据库表名和查询模式生成实用的示例问题
func (h *SuggestionsHandler) generateSmartSuggestions() []string {
	// 返回精心设计的示例问题，涵盖不同查询场景
	return []string{
		"boom_user表有多少条数据？",
		"查询所有VIP用户",
		"统计不同等级的用户数量",
		"查询最近创建的10个用户",
	}
}

// ClearCache 清空缓存（用于测试或手动刷新）
func (h *SuggestionsHandler) ClearCache() {
	h.cache.clear()
}

// setCORSHeaders 设置CORS头
func (h *SuggestionsHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")
}

// sendJSONResponse 发送JSON响应
func (h *SuggestionsHandler) sendJSONResponse(w http.ResponseWriter, data any, statusCode int) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
	}
}

// sendErrorResponse 发送错误响应
func (h *SuggestionsHandler) sendErrorResponse(w http.ResponseWriter, message, code string, statusCode int) {
	response := struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Code    string `json:"code,omitempty"`
	}{
		Success: false,
		Error:   message,
		Code:    code,
	}

	h.sendJSONResponse(w, response, statusCode)
}
