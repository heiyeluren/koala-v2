// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供Koala反作弊频率控制系统的HTTP路由配置。
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RouterConfig 路由配置。
type RouterConfig struct {
	// 请求超时时间
	RequestTimeout time.Duration
	// 是否启用CORS
	EnableCORS bool
	// CORS允许的来源列表
	CORSAllowOrigins []string
	// 是否启用指标收集
	EnableMetrics bool
	// API限流配置（每秒请求数）
	RateLimitPerSecond int64
}

// DefaultRouterConfig 返回默认路由配置。
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		RequestTimeout:     30 * time.Second,
		EnableCORS:         true,
		EnableMetrics:      true,
		RateLimitPerSecond: 10000,
	}
}

// NewRouter 创建并配置Gin路由器。
func NewRouter(handler *Handler, config *RouterConfig) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由器
	router := gin.New()

	// 使用默认配置
	if config == nil {
		config = DefaultRouterConfig()
	}

	// 添加全局中间件
	router.Use(RecoveryMiddleware())
	router.Use(LoggingMiddleware())

	if config.EnableCORS {
		router.Use(CORSMiddlewareWithConfig(CORSConfig{
			AllowOrigins: config.CORSAllowOrigins,
		}))
	}

	if config.RequestTimeout > 0 {
		router.Use(TimeoutMiddleware(config.RequestTimeout))
	}

	// 指标中间件
	var metricsMiddleware *MetricsMiddleware
	if config.EnableMetrics {
		metricsMiddleware = NewMetricsMiddleware()
		router.Use(metricsMiddleware.Handler())
	}

	// ========== 健康检查端点 ==========
	router.GET("/health", handler.Health)
	router.GET("/ready", handler.Ready)

	// ========== 指标端点 ==========
	router.GET("/metrics", func(c *gin.Context) {
		// 简化的Prometheus格式指标输出
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, buildPrometheusMetrics(metricsMiddleware))
	})

	// ========== API v1 端点 ==========
	v1 := router.Group("/api/v1")
	{
		// 频率检查
		v1.POST("/browse", handler.Browse)
		// 计数器更新
		v1.POST("/update", handler.Update)
		// 批量检查
		v1.POST("/batch", handler.Batch)
	}

	return router
}

// buildPrometheusMetrics 构建Prometheus格式的指标输出。
func buildPrometheusMetrics(m *MetricsMiddleware) string {
	output := "# HELP koala_http_requests_total Koala HTTP请求总数\n"
	output += "# TYPE koala_http_requests_total counter\n"

	if m == nil {
		output += "koala_http_requests_total 0\n"
		return output
	}

	metrics := m.GetMetrics()
	for path, data := range metrics {
		if pathMetrics, ok := data.(map[string]interface{}); ok {
			total := pathMetrics["total_requests"]
			failed := pathMetrics["failed_requests"]
			avgLatency := pathMetrics["avg_latency_ms"]
			output += formatMetricLine("koala_http_requests_total", path, total)
			output += formatMetricLine("koala_http_requests_failed", path, failed)
			output += formatMetricLine("koala_http_latency_ms", path, avgLatency)
		}
	}

	// 添加基本的运行时指标
	output += "\n# HELP koala_up Koala服务是否运行\n"
	output += "# TYPE koala_up gauge\n"
	output += "koala_up 1\n"

	return output
}

// formatMetricLine 格式化单行指标。
func formatMetricLine(name string, path string, value interface{}) string {
	return name + "{path=\"" + path + "\"} " + formatValue(value) + "\n"
}

// formatValue 格式化指标值。
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case int64:
		return intToString(val)
	case int:
		return intToString(int64(val))
	case float64:
		return floatToString(val)
	default:
		return "0"
	}
}

// intToString 将int64转换为字符串。
func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// floatToString 将float64转换为字符串（简化版）。
func floatToString(f float64) string {
	return intToString(int64(f))
}

// Server HTTP服务器封装。
type Server struct {
	router     *gin.Engine
	addr       string
	httpServer *http.Server
}

// NewServer 创建新的HTTP服务器。
func NewServer(handler *Handler, addr string, config *RouterConfig) *Server {
	router := NewRouter(handler, config)
	return &Server{
		router: router,
		addr:   addr,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: router,
		},
	}
}

// Run 启动HTTP服务器。
func (s *Server) Run() error {
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 优雅关闭服务器。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Handler 返回底层的HTTP处理器（用于测试）。
func (s *Server) Handler() http.Handler {
	return s.router
}
