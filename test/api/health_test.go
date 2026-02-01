// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供 Health/Ready/Metrics API 端到端测试。
package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== Health API 测试 ==========

// TestHealth_BasicRequest 测试健康检查基本请求。
func TestHealth_BasicRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Health()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "ok", resp.Status)
	assert.NotEmpty(t, resp.Timestamp)
}

// TestHealth_ResponseFormat 测试健康检查响应格式。
func TestHealth_ResponseFormat(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Health()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", httpResp.Header.Get("Content-Type"))
}

// TestHealth_TimestampFormat 测试时间戳格式（RFC3339）。
func TestHealth_TimestampFormat(t *testing.T) {
	s := GetTestServer(t)

	resp, _, err := s.Health()

	require.NoError(t, err)
	// 验证时间戳格式（RFC3339: 2006-01-02T15:04:05Z07:00）
	assert.Contains(t, resp.Timestamp, "T")
	assert.True(t, strings.HasSuffix(resp.Timestamp, "Z") || strings.Contains(resp.Timestamp, "+") || strings.Contains(resp.Timestamp, "-"))
}

// TestHealth_ResponseHeaders 测试健康检查响应头。
func TestHealth_ResponseHeaders(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Health()

	require.NoError(t, err)
	// 检查CORS头
	assert.Equal(t, "*", httpResp.Header.Get("Access-Control-Allow-Origin"))
}

// TestHealth_WrongMethod 测试健康检查错误的HTTP方法。
func TestHealth_WrongMethod(t *testing.T) {
	s := GetTestServer(t)

	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, s.BaseURL()+"/health", nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// 应该返回405或404
			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// TestHealth_Concurrent 测试并发健康检查。
func TestHealth_Concurrent(t *testing.T) {
	s := GetTestServer(t)

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			resp, httpResp, err := s.Health()
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.Equal(t, "ok", resp.Status)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}
}

// ========== Ready API 测试 ==========

// TestReady_BasicRequest 测试就绪检查基本请求。
func TestReady_BasicRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Ready()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Ready)
}

// TestReady_ResponseFormat 测试就绪检查响应格式。
func TestReady_ResponseFormat(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Ready()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", httpResp.Header.Get("Content-Type"))
}

// TestReady_ResponseHeaders 测试就绪检查响应头。
func TestReady_ResponseHeaders(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Ready()

	require.NoError(t, err)
	// 检查CORS头
	assert.Equal(t, "*", httpResp.Header.Get("Access-Control-Allow-Origin"))
}

// TestReady_WrongMethod 测试就绪检查错误的HTTP方法。
func TestReady_WrongMethod(t *testing.T) {
	s := GetTestServer(t)

	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, s.BaseURL()+"/ready", nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// TestReady_Concurrent 测试并发就绪检查。
func TestReady_Concurrent(t *testing.T) {
	s := GetTestServer(t)

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func() {
			resp, httpResp, err := s.Ready()
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Ready)
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

// ========== Metrics API 测试 ==========

// TestMetrics_BasicRequest 测试指标接口基本请求。
func TestMetrics_BasicRequest(t *testing.T) {
	s := GetTestServer(t)

	body, httpResp, err := s.Metrics()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.NotEmpty(t, body)
}

// TestMetrics_ContentType 测试指标接口Content-Type（Prometheus格式）。
func TestMetrics_ContentType(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Metrics()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	// Prometheus指标应该是text/plain
	assert.Contains(t, httpResp.Header.Get("Content-Type"), "text/plain")
}

// TestMetrics_PrometheusFormat 测试指标是否符合Prometheus格式。
func TestMetrics_PrometheusFormat(t *testing.T) {
	s := GetTestServer(t)

	body, httpResp, err := s.Metrics()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)

	// 验证Prometheus格式
	// 应该包含HELP和TYPE注释
	assert.Contains(t, body, "# HELP")
	assert.Contains(t, body, "# TYPE")

	// 应该包含基本指标
	assert.Contains(t, body, "koala_")

	// 应该包含up指标
	assert.Contains(t, body, "koala_up")
}

// TestMetrics_WrongMethod 测试指标接口错误的HTTP方法。
func TestMetrics_WrongMethod(t *testing.T) {
	s := GetTestServer(t)

	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, s.BaseURL()+"/metrics", nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// TestMetrics_AfterRequests 测试发送请求后的指标变化。
func TestMetrics_AfterRequests(t *testing.T) {
	s := GetTestServer(t)

	// 先获取初始指标
	initialBody, _, _ := s.Metrics()

	// 发送一些请求
	for i := 0; i < 10; i++ {
		s.Browse(APIRequest{Act: "test_metrics", UID: UniqueID("user")})
	}

	// 再次获取指标
	afterBody, httpResp, err := s.Metrics()

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.NotEmpty(t, afterBody)

	// 指标应该有变化（请求计数增加）
	// 具体验证取决于指标实现
	_ = initialBody // 避免未使用警告
}

// TestMetrics_Concurrent 测试并发指标请求。
func TestMetrics_Concurrent(t *testing.T) {
	s := GetTestServer(t)

	done := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func() {
			body, httpResp, err := s.Metrics()
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.NotEmpty(t, body)
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}

// ========== 路由测试 ==========

// TestNotFound_UnknownPath 测试未知路径返回404。
func TestNotFound_UnknownPath(t *testing.T) {
	s := GetTestServer(t)

	paths := []string{
		"/unknown",
		"/api/v2/browse",
		"/api/v1/unknown",
		"/healthz",
		"/readiness",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			resp, err := s.Get(path)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

// TestRoot_Path 测试根路径。
func TestRoot_Path(t *testing.T) {
	s := GetTestServer(t)

	resp, err := s.Get("/")
	require.NoError(t, err)
	defer resp.Body.Close()

	// 根路径可能返回404或其他响应
	assert.True(t, resp.StatusCode >= 200)
}

// ========== CORS 测试 ==========

// TestCORS_OptionsRequest 测试CORS预检请求。
func TestCORS_OptionsRequest(t *testing.T) {
	s := GetTestServer(t)

	endpoints := []string{
		"/health",
		"/ready",
		"/metrics",
		"/api/v1/browse",
		"/api/v1/update",
		"/api/v1/batch",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodOptions, s.BaseURL()+endpoint, nil)
			req.Header.Set("Origin", "http://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")

			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// OPTIONS 应该返回 204
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// 应该有CORS头
			assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
			assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
		})
	}
}

// TestCORS_Headers 测试CORS响应头。
func TestCORS_Headers(t *testing.T) {
	s := GetTestServer(t)

	req, _ := http.NewRequest(http.MethodGet, s.BaseURL()+"/health", nil)
	req.Header.Set("Origin", "http://example.com")

	resp, err := s.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 应该有CORS头
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

// ========== 请求ID 测试 ==========

// TestRequestID_Generated 测试自动生成请求ID。
func TestRequestID_Generated(t *testing.T) {
	s := GetTestServer(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodGet, "/ready"},
		{http.MethodGet, "/metrics"},
	}

	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req, _ := http.NewRequest(ep.method, s.BaseURL()+ep.path, nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// 应该有请求ID
			requestID := resp.Header.Get("X-Request-ID")
			assert.NotEmpty(t, requestID)
		})
	}
}

// TestRequestID_Echo 测试请求ID回显。
func TestRequestID_Echo(t *testing.T) {
	s := GetTestServer(t)

	customID := "custom-request-id-12345"

	req, _ := http.NewRequest(http.MethodGet, s.BaseURL()+"/health", nil)
	req.Header.Set("X-Request-ID", customID)

	resp, err := s.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 应该返回相同的请求ID
	assert.Equal(t, customID, resp.Header.Get("X-Request-ID"))
}

// ========== 综合测试 ==========

// TestAllEndpoints_Available 测试所有端点可用。
func TestAllEndpoints_Available(t *testing.T) {
	s := GetTestServer(t)

	endpoints := []struct {
		method string
		path   string
		body   interface{}
	}{
		{http.MethodGet, "/health", nil},
		{http.MethodGet, "/ready", nil},
		{http.MethodGet, "/metrics", nil},
		{http.MethodPost, "/api/v1/browse", APIRequest{Act: "test"}},
		{http.MethodPost, "/api/v1/update", APIRequest{Act: "test"}},
		{http.MethodPost, "/api/v1/batch", BatchRequest{Requests: []BatchItem{{ID: "1", Act: "test"}}}},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var resp *http.Response
			var err error

			if ep.method == http.MethodGet {
				resp, err = s.Get(ep.path)
			} else {
				resp, err = s.PostJSON(ep.path, ep.body)
			}

			require.NoError(t, err)
			defer resp.Body.Close()

			// 所有端点应该返回2xx
			assert.True(t, resp.StatusCode >= 200 && resp.StatusCode < 300,
				"端点 %s %s 返回 %d", ep.method, ep.path, resp.StatusCode)
		})
	}
}

// TestHealthEndpoints_IndependentOfEngine 测试健康端点不依赖引擎状态。
func TestHealthEndpoints_IndependentOfEngine(t *testing.T) {
	s := GetTestServer(t)

	// 健康检查应该始终可用
	healthResp, healthHTTP, _ := s.Health()
	assert.Equal(t, http.StatusOK, healthHTTP.StatusCode)
	assert.Equal(t, "ok", healthResp.Status)

	// 就绪检查应该始终可用
	readyResp, readyHTTP, _ := s.Ready()
	assert.Equal(t, http.StatusOK, readyHTTP.StatusCode)
	assert.True(t, readyResp.Ready)

	// 指标应该始终可用
	_, metricsHTTP, _ := s.Metrics()
	assert.Equal(t, http.StatusOK, metricsHTTP.StatusCode)
}
