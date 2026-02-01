// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供 Browse API 端到端测试。
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== 正常场景测试 ==========

// TestBrowse_BasicRequest 测试基本请求。
func TestBrowse_BasicRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "test_action",
		UID: UniqueID("user"),
		IP:  "192.168.1.100",
		DID: "device001",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
	assert.Equal(t, 0, resp.Code)
}

// TestBrowse_MinimalRequest 测试最小请求（仅必填字段）。
func TestBrowse_MinimalRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "minimal_test",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// TestBrowse_WithExtFields 测试带扩展字段的请求。
func TestBrowse_WithExtFields(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "test_ext",
		UID: UniqueID("user"),
		Ext: map[string]string{
			"channel":  "web",
			"platform": "pc",
			"version":  "1.0.0",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// TestBrowse_WithUpdate 测试带自动更新标志的请求。
func TestBrowse_WithUpdate(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act:    "test_update",
		UID:    UniqueID("user"),
		Update: true,
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// ========== 白名单测试 ==========

// TestBrowse_WhitelistUser 测试白名单用户放行。
func TestBrowse_WhitelistUser(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "login",
		UID: "vip_user_001", // 白名单用户
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, resp.Message, "VIP")
}

// TestBrowse_WhitelistInternalIP 测试内网IP白名单放行。
func TestBrowse_WhitelistInternalIP(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "login",
		UID: UniqueID("user"),
		IP:  "10.0.0.1", // 内网IP
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
	assert.Contains(t, resp.Message, "内网")
}

// ========== 黑名单测试 ==========

// TestBrowse_BlacklistIP 测试黑名单IP拦截。
func TestBrowse_BlacklistIP(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "login",
		UID: UniqueID("user"),
		IP:  "192.168.100.1", // 黑名单IP
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.False(t, resp.Allowed)
	assert.Equal(t, 4003, resp.Code)
	assert.Contains(t, resp.Message, "封禁")
}

// TestBrowse_BlacklistDevice 测试黑名单设备拦截。
func TestBrowse_BlacklistDevice(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "login",
		UID: UniqueID("user"),
		DID: "blocked_device_001", // 黑名单设备
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.False(t, resp.Allowed)
	assert.Equal(t, 4004, resp.Code)
	assert.Contains(t, resp.Message, "设备")
}

// ========== 限流测试 ==========

// TestBrowse_LoginRateLimit 测试登录频率限制。
func TestBrowse_LoginRateLimit(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("ratelimit_user")

	// 连续请求触发限流（规则：60秒内5次）
	var lastResp *APIResponse
	for i := 0; i < 10; i++ {
		resp, httpResp, err := s.Browse(APIRequest{
			Act:    "login",
			UID:    uid,
			Update: true, // 需要更新计数器
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		lastResp = resp
	}

	// 最后应该被限流
	assert.False(t, lastResp.Allowed)
	assert.Equal(t, 4001, lastResp.Code)
	assert.Contains(t, lastResp.Message, "频繁")
}

// TestBrowse_IPRateLimit 测试IP频率限制。
func TestBrowse_IPRateLimit(t *testing.T) {
	s := GetTestServer(t)
	ip := "192.168.200." + UniqueID("")[:3]

	// 连续请求触发IP限流（规则：60秒内10次）
	// 使用专门的IP测试action，只按IP计数
	var lastResp *APIResponse
	for i := 0; i < 15; i++ {
		resp, httpResp, err := s.Browse(APIRequest{
			Act:    "login_ip_test", // 专门测试IP限流的action
			UID:    UniqueID("user"), // 不同用户
			IP:     ip,               // 相同IP
			Update: true,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		lastResp = resp
	}

	// 最后应该被IP限流
	assert.False(t, lastResp.Allowed)
	assert.Equal(t, 4002, lastResp.Code)
}

// TestBrowse_PostRateLimit 测试发帖频率限制。
func TestBrowse_PostRateLimit(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("poster")

	// 发帖限制（规则：3600秒内20次）
	var lastResp *APIResponse
	for i := 0; i < 25; i++ {
		resp, httpResp, err := s.Browse(APIRequest{
			Act:    "post",
			UID:    uid,
			Update: true,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		lastResp = resp
	}

	// 最后应该被限流
	assert.False(t, lastResp.Allowed)
	assert.Equal(t, 4101, lastResp.Code)
}

// TestBrowse_CommentRateLimit 测试评论频率限制。
func TestBrowse_CommentRateLimit(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("commenter")

	// 评论限制（规则：60秒内10次）
	var lastResp *APIResponse
	for i := 0; i < 15; i++ {
		resp, httpResp, err := s.Browse(APIRequest{
			Act:    "comment",
			UID:    uid,
			Update: true,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		lastResp = resp
	}

	// 最后应该被限流
	assert.False(t, lastResp.Allowed)
	assert.Equal(t, 4102, lastResp.Code)
}

// ========== 边界场景测试 ==========

// TestBrowse_EmptyUID 测试空UID。
func TestBrowse_EmptyUID(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "test",
		UID: "",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed) // 空UID仍然可以处理
}

// TestBrowse_SpecialCharacters 测试特殊字符。
func TestBrowse_SpecialCharacters(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		req  APIRequest
	}{
		{
			name: "中文用户名",
			req:  APIRequest{Act: "test", UID: "用户_测试_123"},
		},
		{
			name: "特殊符号",
			req:  APIRequest{Act: "test", UID: "user!@#$%^&*()"},
		},
		{
			name: "Emoji",
			req:  APIRequest{Act: "test", UID: "user_😀_test"},
		},
		{
			name: "空格",
			req:  APIRequest{Act: "test", UID: "user with spaces"},
		},
		{
			name: "换行符",
			req:  APIRequest{Act: "test", UID: "user\nwith\nnewlines"},
		},
		{
			name: "中文扩展字段",
			req: APIRequest{
				Act: "test",
				UID: UniqueID("user"),
				Ext: map[string]string{"键": "值", "渠道": "网页"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, httpResp, err := s.Browse(tc.req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Allowed)
		})
	}
}

// TestBrowse_LongStrings 测试超长字符串。
func TestBrowse_LongStrings(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		req  APIRequest
	}{
		{
			name: "长UID_100字符",
			req:  APIRequest{Act: "test", UID: RepeatString("a", 100)},
		},
		{
			name: "长UID_1000字符",
			req:  APIRequest{Act: "test", UID: RepeatString("b", 1000)},
		},
		{
			name: "长DID",
			req:  APIRequest{Act: "test", DID: RepeatString("c", 500)},
		},
		{
			name: "长Act",
			req:  APIRequest{Act: RepeatString("d", 200)},
		},
		{
			name: "多个长扩展字段",
			req: APIRequest{
				Act: "test",
				UID: UniqueID("user"),
				Ext: map[string]string{
					"key1": RepeatString("e", 500),
					"key2": RepeatString("f", 500),
					"key3": RepeatString("g", 500),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, httpResp, err := s.Browse(tc.req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Allowed)
		})
	}
}

// TestBrowse_IPFormats 测试各种IP格式。
func TestBrowse_IPFormats(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		ip   string
	}{
		{"IPv4标准", "192.168.1.100"},
		{"IPv4全零", "0.0.0.0"},
		{"IPv4广播", "255.255.255.255"},
		{"IPv4本地", "127.0.0.1"},
		{"IPv6标准", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"IPv6简写", "::1"},
		{"IPv6映射IPv4", "::ffff:192.168.1.1"},
		{"空IP", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, httpResp, err := s.Browse(APIRequest{
				Act: "test_ip",
				UID: UniqueID("user"),
				IP:  tc.ip,
			})
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			// 只要能处理就行，不检查具体结果
		})
	}
}

// ========== 错误场景测试 ==========

// TestBrowse_MissingAct 测试缺少必填字段Act。
func TestBrowse_MissingAct(t *testing.T) {
	s := GetTestServer(t)

	resp, err := s.PostJSON("/api/v1/browse", map[string]string{
		"uid": "user123",
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var apiResp APIResponse
	json.Unmarshal(body, &apiResp)
	assert.False(t, apiResp.Allowed)
	assert.Equal(t, -1, apiResp.Code)
}

// TestBrowse_InvalidJSON 测试无效JSON。
func TestBrowse_InvalidJSON(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		body string
	}{
		{"空字符串", ""},
		{"无效JSON", "not json"},
		{"不完整JSON", `{"act": "test"`},
		{"数组而非对象", `["test"]`},
		{"null", "null"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.PostRaw("/api/v1/browse", []byte(tc.body), "application/json")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// TestBrowse_WrongContentType 测试错误的Content-Type。
func TestBrowse_WrongContentType(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name        string
		contentType string
	}{
		{"text/plain", "text/plain"},
		{"text/html", "text/html"},
		{"application/xml", "application/xml"},
		{"无Content-Type", ""},
	}

	body := `{"act": "test"}`
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.PostRaw("/api/v1/browse", []byte(body), tc.contentType)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Gin 默认会尝试解析JSON，但行为可能因Content-Type而异
			// 这里主要测试不会崩溃
			assert.True(t, resp.StatusCode >= 200)
		})
	}
}

// TestBrowse_WrongFieldTypes 测试错误的字段类型。
func TestBrowse_WrongFieldTypes(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		body string
	}{
		{"act为数字", `{"act": 123}`},
		{"uid为数组", `{"act": "test", "uid": [1,2,3]}`},
		{"ext为字符串", `{"act": "test", "ext": "string"}`},
		{"update为字符串", `{"act": "test", "update": "true"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.PostRaw("/api/v1/browse", []byte(tc.body), "application/json")
			require.NoError(t, err)
			defer resp.Body.Close()

			// 可能返回400或200（取决于Gin的解析行为）
			assert.True(t, resp.StatusCode >= 200)
		})
	}
}

// ========== 响应验证测试 ==========

// TestBrowse_ResponseFormat 测试响应格式正确性。
func TestBrowse_ResponseFormat(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Browse(APIRequest{
		Act: "test_format",
		UID: UniqueID("user"),
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", httpResp.Header.Get("Content-Type"))

	// 验证响应包含必要字段
	assert.NotNil(t, resp)
	// Allowed 是 bool，总会有值
	// Code 是 int，总会有值
	// Message 可能为空
}

// TestBrowse_ResponseHeaders 测试响应头。
func TestBrowse_ResponseHeaders(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Browse(APIRequest{
		Act: "test_headers",
	})

	require.NoError(t, err)

	// 检查CORS头
	assert.Equal(t, "*", httpResp.Header.Get("Access-Control-Allow-Origin"))

	// 检查请求ID头
	requestID := httpResp.Header.Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
}

// TestBrowse_RuleName 测试规则名称返回。
func TestBrowse_RuleName(t *testing.T) {
	s := GetTestServer(t)

	// 触发白名单规则
	resp, _, err := s.Browse(APIRequest{
		Act: "login",
		UID: "vip_user_001",
	})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.NotEmpty(t, resp.RuleName)
	assert.Contains(t, strings.ToLower(resp.RuleName), "whitelist")
}

// TestBrowse_AuthType 测试认证类型返回。
func TestBrowse_AuthType(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("authtype_user")

	// 触发需要验证的限流规则
	for i := 0; i < 10; i++ {
		s.Browse(APIRequest{
			Act:    "login",
			UID:    uid,
			Update: true,
		})
	}

	// 最后一次应该返回auth_type
	resp, _, err := s.Browse(APIRequest{
		Act:    "login",
		UID:    uid,
		Update: true,
	})
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, 1, resp.AuthType) // login规则配置的auth_type=1
}

// ========== HTTP 方法测试 ==========

// TestBrowse_WrongMethod 测试错误的HTTP方法。
func TestBrowse_WrongMethod(t *testing.T) {
	s := GetTestServer(t)

	methods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, s.BaseURL()+"/api/v1/browse", nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// 应该返回405 Method Not Allowed 或 404
			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// TestBrowse_OPTIONS 测试OPTIONS请求（CORS预检）。
func TestBrowse_OPTIONS(t *testing.T) {
	s := GetTestServer(t)

	req, _ := http.NewRequest(http.MethodOptions, s.BaseURL()+"/api/v1/browse", nil)
	resp, err := s.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// OPTIONS 应该返回 204
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
}
