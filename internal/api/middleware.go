// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供Koala反作弊频率控制系统的HTTP中间件实现。
package api

import (
	"bytes"
	"io"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ========== 日志中间件 ==========

// LoggingMiddleware 记录HTTP请求日志。
// 记录请求方法、路径、状态码、耗时等信息。
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		start := time.Now()

		// 获取请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 记录请求体（用于调试）
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)

		// 获取状态码
		status := c.Writer.Status()

		// 构建日志字段（这里简化处理，实际项目可以使用zap等日志库）
		_ = map[string]interface{}{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		// 如果请求失败，记录更多信息
		if status >= 400 {
			_ = map[string]interface{}{
				"request_body": string(reqBody),
				"errors":       c.Errors.String(),
			}
		}
	}
}

// requestIDCounter 用于生成唯一请求ID的计数器。
var (
	requestIDCounter uint64
	requestIDMutex   sync.Mutex
)

// generateRequestID 生成唯一的请求ID。
func generateRequestID() string {
	requestIDMutex.Lock()
	defer requestIDMutex.Unlock()
	requestIDCounter++
	return time.Now().Format("20060102150405") + "-" + uintToString(requestIDCounter)
}

// uintToString 将uint64转换为字符串。
func uintToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ========== 恢复中间件 ==========

// RecoveryMiddleware 捕获panic并返回500错误。
// 防止单个请求的panic导致整个服务崩溃。
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := debug.Stack()

				// 记录错误日志（这里简化处理）
				_ = map[string]interface{}{
					"error": err,
					"stack": string(stack),
				}

				// 返回500错误
				c.AbortWithStatusJSON(http.StatusInternalServerError, APIResponse{
					Allowed: false,
					Code:    -500,
					Message: "服务器内部错误",
				})
			}
		}()
		c.Next()
	}
}

// ========== 指标中间件 ==========

// MetricsMiddleware 收集HTTP请求指标。
// 记录请求计数、延迟分布等指标。
type MetricsMiddleware struct {
	mu sync.RWMutex

	// 请求计数器
	totalRequests  map[string]int64
	failedRequests map[string]int64

	// 延迟统计
	latencySum   map[string]time.Duration
	latencyCount map[string]int64
}

// NewMetricsMiddleware 创建新的指标中间件。
func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		totalRequests:  make(map[string]int64),
		failedRequests: make(map[string]int64),
		latencySum:     make(map[string]time.Duration),
		latencyCount:   make(map[string]int64),
	}
}

// Handler 返回Gin中间件处理函数。
func (m *MetricsMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		status := c.Writer.Status()

		// 更新指标
		key := method + "_" + path
		m.mu.Lock()
		m.totalRequests[key]++
		if status >= 400 {
			m.failedRequests[key]++
		}
		m.latencySum[key] += latency
		m.latencyCount[key]++
		m.mu.Unlock()
	}
}

// GetMetrics 返回当前指标数据。
func (m *MetricsMiddleware) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})
	for key, count := range m.totalRequests {
		avgLatency := time.Duration(0)
		if m.latencyCount[key] > 0 {
			avgLatency = m.latencySum[key] / time.Duration(m.latencyCount[key])
		}
		result[key] = map[string]interface{}{
			"total_requests":  count,
			"failed_requests": m.failedRequests[key],
			"avg_latency_ms":  avgLatency.Milliseconds(),
		}
	}
	return result
}

// ========== CORS中间件 ==========

// CORSMiddleware 处理跨域请求。
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ========== 超时中间件 ==========

// TimeoutMiddleware 为请求设置超时时间。
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 注意：Gin的context不支持直接设置超时
		// 这里只设置一个截止时间供handler参考
		c.Set("deadline", time.Now().Add(timeout))
		c.Next()
	}
}

// ========== 限流中间件 ==========

// RateLimitMiddleware 对API本身进行限流保护。
// 防止API服务被过多请求压垮。
type RateLimitMiddleware struct {
	mu       sync.Mutex
	counters map[string]*rateLimitCounter
	limit    int64
	window   time.Duration
}

// rateLimitCounter 限流计数器。
type rateLimitCounter struct {
	count     int64
	windowEnd time.Time
}

// NewRateLimitMiddleware 创建新的限流中间件。
func NewRateLimitMiddleware(limit int64, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		counters: make(map[string]*rateLimitCounter),
		limit:    limit,
		window:   window,
	}
}

// Handler 返回Gin中间件处理函数。
func (r *RateLimitMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		r.mu.Lock()
		counter, exists := r.counters[clientIP]
		if !exists || now.After(counter.windowEnd) {
			r.counters[clientIP] = &rateLimitCounter{
				count:     1,
				windowEnd: now.Add(r.window),
			}
			r.mu.Unlock()
			c.Next()
			return
		}

		if counter.count >= r.limit {
			r.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, APIResponse{
				Allowed: false,
				Code:    429,
				Message: "请求过于频繁，请稍后重试",
			})
			return
		}

		counter.count++
		r.mu.Unlock()
		c.Next()
	}
}
