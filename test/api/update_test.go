// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供 Update API 端到端测试。
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== 正常场景测试 ==========

// TestUpdate_BasicRequest 测试基本更新请求。
func TestUpdate_BasicRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Update(APIRequest{
		Act: "test_update",
		UID: UniqueID("user"),
		IP:  "192.168.1.100",
		DID: "device001",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Message)
}

// TestUpdate_MinimalRequest 测试最小更新请求。
func TestUpdate_MinimalRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Update(APIRequest{
		Act: "minimal_update",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// TestUpdate_WithExtFields 测试带扩展字段的更新请求。
func TestUpdate_WithExtFields(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Update(APIRequest{
		Act: "test_update_ext",
		UID: UniqueID("user"),
		Ext: map[string]string{
			"channel":  "app",
			"platform": "ios",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// ========== 计数器验证测试 ==========

// TestUpdate_CounterIncrement 测试计数器递增。
func TestUpdate_CounterIncrement(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("counter_user")
	act := UniqueID("counter_act") // 使用唯一的action避免与其他测试干扰

	// 先调用Browse检查初始状态（应该允许）
	browseResp, _, err := s.Browse(APIRequest{
		Act: act,
		UID: uid,
	})
	require.NoError(t, err)
	assert.True(t, browseResp.Allowed)

	// 通过Update增加计数器（默认规则是60秒100次）
	for i := 0; i < 100; i++ {
		updateResp, httpResp, err := s.Update(APIRequest{
			Act: act,
			UID: uid,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		assert.True(t, updateResp.Allowed)
	}

	// 再次Browse应该触发限流（因为默认规则是60秒100次）
	browseResp, _, err = s.Browse(APIRequest{
		Act: act,
		UID: uid,
	})
	require.NoError(t, err)
	assert.False(t, browseResp.Allowed)
	assert.Equal(t, 4999, browseResp.Code) // global_default_limit
}

// TestUpdate_DifferentActions 测试不同行为的计数器独立。
func TestUpdate_DifferentActions(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("multi_action_user")
	act1 := UniqueID("action1")
	act2 := UniqueID("action2")

	// 更新 act1 计数器
	for i := 0; i < 50; i++ {
		s.Update(APIRequest{Act: act1, UID: uid})
	}

	// 更新 act2 计数器
	for i := 0; i < 50; i++ {
		s.Update(APIRequest{Act: act2, UID: uid})
	}

	// act1 应该还没触发限流（100次限制）
	browseResp, _, _ := s.Browse(APIRequest{Act: act1, UID: uid})
	assert.True(t, browseResp.Allowed)

	// act2 应该还没触发限流（100次限制）
	browseResp, _, _ = s.Browse(APIRequest{Act: act2, UID: uid})
	assert.True(t, browseResp.Allowed)
}

// TestUpdate_DifferentUsers 测试不同用户的计数器独立。
func TestUpdate_DifferentUsers(t *testing.T) {
	s := GetTestServer(t)
	uid1 := UniqueID("user1")
	uid2 := UniqueID("user2")
	act := UniqueID("diff_users_act") // 使用唯一action避免干扰

	// 用户1更新101次触发限流（默认规则：60秒内100次）
	for i := 0; i < 101; i++ {
		s.Update(APIRequest{Act: act, UID: uid1})
	}

	// 用户2应该不受影响
	browseResp, _, _ := s.Browse(APIRequest{Act: act, UID: uid2})
	assert.True(t, browseResp.Allowed)

	// 用户1应该被限流
	browseResp, _, _ = s.Browse(APIRequest{Act: act, UID: uid1})
	assert.False(t, browseResp.Allowed)
}

// ========== 边界场景测试 ==========

// TestUpdate_EmptyUID 测试空UID。
func TestUpdate_EmptyUID(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Update(APIRequest{
		Act: "test",
		UID: "",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// TestUpdate_SpecialCharacters 测试特殊字符。
func TestUpdate_SpecialCharacters(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		uid  string
	}{
		{"中文", "用户_123"},
		{"特殊符号", "user!@#$%"},
		{"Emoji", "user_😀"},
		{"空格", "user with space"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, httpResp, err := s.Update(APIRequest{
				Act: "test",
				UID: tc.uid,
			})
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Allowed)
		})
	}
}

// TestUpdate_LongStrings 测试超长字符串。
func TestUpdate_LongStrings(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		uid  string
	}{
		{"100字符", RepeatString("a", 100)},
		{"500字符", RepeatString("b", 500)},
		{"1000字符", RepeatString("c", 1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, httpResp, err := s.Update(APIRequest{
				Act: "test",
				UID: tc.uid,
			})
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Allowed)
		})
	}
}

// ========== 错误场景测试 ==========

// TestUpdate_MissingAct 测试缺少Act字段。
func TestUpdate_MissingAct(t *testing.T) {
	s := GetTestServer(t)

	resp, err := s.PostJSON("/api/v1/update", map[string]string{
		"uid": "user123",
	})
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var apiResp APIResponse
	json.Unmarshal(body, &apiResp)
	assert.False(t, apiResp.Allowed)
}

// TestUpdate_InvalidJSON 测试无效JSON。
func TestUpdate_InvalidJSON(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		body string
	}{
		{"空字符串", ""},
		{"无效JSON", "invalid"},
		{"不完整JSON", `{"act":`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.PostRaw("/api/v1/update", []byte(tc.body), "application/json")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// TestUpdate_WrongMethod 测试错误的HTTP方法。
func TestUpdate_WrongMethod(t *testing.T) {
	s := GetTestServer(t)

	methods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, s.BaseURL()+"/api/v1/update", nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// ========== 响应验证测试 ==========

// TestUpdate_ResponseFormat 测试响应格式。
func TestUpdate_ResponseFormat(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Update(APIRequest{
		Act: "test",
		UID: UniqueID("user"),
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", httpResp.Header.Get("Content-Type"))

	// Update成功应该返回固定响应
	assert.True(t, resp.Allowed)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Message)
}

// TestUpdate_ResponseHeaders 测试响应头。
func TestUpdate_ResponseHeaders(t *testing.T) {
	s := GetTestServer(t)

	_, httpResp, err := s.Update(APIRequest{
		Act: "test",
	})

	require.NoError(t, err)

	// 检查CORS头
	assert.Equal(t, "*", httpResp.Header.Get("Access-Control-Allow-Origin"))

	// 检查请求ID头
	assert.NotEmpty(t, httpResp.Header.Get("X-Request-ID"))
}

// ========== 白名单/黑名单影响测试 ==========

// TestUpdate_WhitelistUserStillUpdates 测试白名单用户仍然更新计数器。
func TestUpdate_WhitelistUserStillUpdates(t *testing.T) {
	s := GetTestServer(t)

	// 白名单用户更新计数器
	resp, httpResp, err := s.Update(APIRequest{
		Act: "login",
		UID: "vip_user_001", // 白名单用户
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, resp.Allowed)
}

// TestUpdate_BlacklistIPStillProcessed 测试黑名单IP的Update请求。
func TestUpdate_BlacklistIPStillProcessed(t *testing.T) {
	s := GetTestServer(t)

	// 黑名单IP的Update请求仍然会被处理
	resp, httpResp, err := s.Update(APIRequest{
		Act: "login",
		UID: UniqueID("user"),
		IP:  "192.168.100.1", // 黑名单IP
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	// Update不检查黑名单，只更新计数器
	assert.True(t, resp.Allowed)
}

// ========== 并发测试 ==========

// TestUpdate_ConcurrentUpdates 测试并发更新。
func TestUpdate_ConcurrentUpdates(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("concurrent_user")

	done := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func() {
			resp, httpResp, err := s.Update(APIRequest{
				Act: "test_concurrent",
				UID: uid,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Allowed)
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 50; i++ {
		<-done
	}
}

// TestUpdate_ConcurrentDifferentUsers 测试并发更新不同用户。
func TestUpdate_ConcurrentDifferentUsers(t *testing.T) {
	s := GetTestServer(t)

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(idx int) {
			uid := UniqueID("user") + string(rune(idx))
			resp, httpResp, err := s.Update(APIRequest{
				Act: "test_concurrent",
				UID: uid,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.True(t, resp.Allowed)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}
}
