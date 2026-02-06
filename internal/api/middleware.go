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
	"context"
	"io"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"koala/pkg/logger"
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

		// 记录请求日志
		logger.Info("HTTP请求完成",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"latency", latency.String(),
			"client_ip", c.ClientIP(),
		)

		// 如果请求失败，记录更多信息
		if status >= 400 {
			logger.Warn("HTTP请求失败",
				"request_id", requestID,
				"status", status,
				"request_body", string(reqBody),
				"errors", c.Errors.String(),
			)
		}
	}
}

// requestIDCounter 用于生成唯一请求ID的原子计数器，无需互斥锁即可并发安全递增。
var requestIDCounter atomic.Uint64

// generateRequestID 生成唯一的请求ID。
func generateRequestID() string {
	id := requestIDCounter.Add(1)
	return time.Now().Format("20060102150405") + "-" + uintToString(id)
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

				// 记录错误日志
				logger.Error("panic恢复",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"stack", string(stack),
				)

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

		// 使用路由模式作为key，避免动态路径参数导致map无限增长
		routePath := c.FullPath()
		if routePath == "" {
			routePath = path // 如果没有匹配到路由，回退使用原始路径
		}
		key := method + "_" + routePath

		// 更新指标
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

// CORSConfig CORS中间件配置。
type CORSConfig struct {
	AllowOrigins []string // 允许的来源列表，为空或包含"*"则允许所有
}

// CORSMiddlewareWithConfig 处理跨域请求。
// 支持可配置的允许来源列表。
func CORSMiddlewareWithConfig(config CORSConfig) gin.HandlerFunc {
	// 预计算是否允许所有来源
	allowAll := len(config.AllowOrigins) == 0
	originMap := make(map[string]bool)
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		originMap[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if originMap[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		} else {
			// 不允许的来源，不设置CORS头
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

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

// CORSMiddleware 处理跨域请求（向后兼容，允许所有来源）。
func CORSMiddleware() gin.HandlerFunc {
	return CORSMiddlewareWithConfig(CORSConfig{})
}

// ========== 超时中间件 ==========

// TimeoutMiddleware 为请求设置超时时间。
// 使用context.WithTimeout为下游handler提供deadline，
// 确保下游可以通过ctx.Deadline()获取截止时间并进行超时控制。
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用context.WithTimeout包装请求上下文，设置真正的超时截止时间
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 将带有deadline的context注入到请求中
		c.Request = c.Request.WithContext(ctx)

		// 同时设置deadline到gin context中，保持向后兼容
		c.Set("deadline", time.Now().Add(timeout))
		c.Next()
	}
}

// ========== 限流中间件 ==========

// defaultCleanupInterval 默认清理间隔（1分钟）。
const defaultCleanupInterval = 1 * time.Minute

// RateLimitMiddleware 对API本身进行限流保护。
// 防止API服务被过多请求压垮。
// 包含后台清理协程，定期移除过期的计数器条目，防止内存泄漏。
type RateLimitMiddleware struct {
	mu       sync.Mutex
	counters map[string]*rateLimitCounter
	limit    int64
	window   time.Duration

	// 清理协程控制
	cleanupInterval time.Duration
	stopChan        chan struct{}
	cleanupOnce     sync.Once
}

// rateLimitCounter 限流计数器，包含最后访问时间用于过期清理。
type rateLimitCounter struct {
	count      int64
	windowEnd  time.Time
	lastAccess time.Time // 最后访问时间，用于清理判断
}

// NewRateLimitMiddleware 创建新的限流中间件。
func NewRateLimitMiddleware(limit int64, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		counters:        make(map[string]*rateLimitCounter),
		limit:           limit,
		window:          window,
		cleanupInterval: defaultCleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

// SetCleanupInterval 设置清理间隔（主要用于测试）。
// 必须在调用Handler()之前设置。
func (r *RateLimitMiddleware) SetCleanupInterval(interval time.Duration) {
	r.cleanupInterval = interval
}

// CountersLen 返回当前计数器map中的条目数量（用于测试和监控）。
func (r *RateLimitMiddleware) CountersLen() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.counters)
}

// StopCleanup 停止后台清理协程。
func (r *RateLimitMiddleware) StopCleanup() {
	select {
	case <-r.stopChan:
		// 已经关闭
	default:
		close(r.stopChan)
	}
}

// startCleanup 启动后台清理协程，定期移除超过窗口期的过期条目。
// 使用sync.Once确保只启动一次。
func (r *RateLimitMiddleware) startCleanup() {
	r.cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(r.cleanupInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					r.cleanup()
				case <-r.stopChan:
					return
				}
			}
		}()
	})
}

// cleanup 清理过期的计数器条目。
// 移除最后访问时间超过窗口期的所有条目。
func (r *RateLimitMiddleware) cleanup() {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	for ip, counter := range r.counters {
		// 如果最后访问时间距今已超过窗口期，则删除该条目
		if now.Sub(counter.lastAccess) > r.window {
			delete(r.counters, ip)
		}
	}
}

// Handler 返回Gin中间件处理函数。
func (r *RateLimitMiddleware) Handler() gin.HandlerFunc {
	// 首次调用Handler时启动清理协程
	r.startCleanup()

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		r.mu.Lock()
		counter, exists := r.counters[clientIP]
		if !exists || now.After(counter.windowEnd) {
			r.counters[clientIP] = &rateLimitCounter{
				count:      1,
				windowEnd:  now.Add(r.window),
				lastAccess: now,
			}
			r.mu.Unlock()
			c.Next()
			return
		}

		if counter.count >= r.limit {
			// 即使被限流也更新最后访问时间
			counter.lastAccess = now
			r.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, APIResponse{
				Allowed: false,
				Code:    429,
				Message: "请求过于频繁，请稍后重试",
			})
			return
		}

		counter.count++
		counter.lastAccess = now
		r.mu.Unlock()
		c.Next()
	}
}
