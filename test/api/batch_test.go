// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供 Batch API 端到端测试。
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

// TestBatch_SingleRequest 测试单条批量请求。
func TestBatch_SingleRequest(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "req1", Act: "test", UID: UniqueID("user")},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "req1", resp.Results[0].ID)
	assert.True(t, resp.Results[0].Allowed)
}

// TestBatch_MultipleRequests 测试多条批量请求。
func TestBatch_MultipleRequests(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "req1", Act: "login", UID: UniqueID("user1")},
			{ID: "req2", Act: "post", UID: UniqueID("user2")},
			{ID: "req3", Act: "comment", UID: UniqueID("user3")},
			{ID: "req4", Act: "test", UID: UniqueID("user4")},
			{ID: "req5", Act: "api_call", UID: UniqueID("user5")},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 5)

	// 验证每个结果都有正确的ID
	ids := make(map[string]bool)
	for _, result := range resp.Results {
		ids[result.ID] = true
		assert.True(t, result.Allowed)
	}
	assert.True(t, ids["req1"])
	assert.True(t, ids["req2"])
	assert.True(t, ids["req3"])
	assert.True(t, ids["req4"])
	assert.True(t, ids["req5"])
}

// TestBatch_MaxRequests 测试最大数量批量请求（100条）。
func TestBatch_MaxRequests(t *testing.T) {
	s := GetTestServer(t)

	requests := make([]BatchItem, 100)
	for i := 0; i < 100; i++ {
		requests[i] = BatchItem{
			ID:  intToStr(i),
			Act: "test",
			UID: UniqueID("user"),
		}
	}

	resp, httpResp, err := s.Batch(BatchRequest{Requests: requests})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Len(t, resp.Results, 100)
}

// TestBatch_WithAllFields 测试包含所有字段的批量请求。
func TestBatch_WithAllFields(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{
				ID:  "full_req",
				Act: "test",
				UID: UniqueID("user"),
				IP:  "192.168.1.100",
				DID: "device001",
				Ext: map[string]string{"channel": "web", "version": "1.0"},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 1)
	assert.True(t, resp.Results[0].Allowed)
}

// ========== 混合结果测试 ==========

// TestBatch_MixedResults_WhitelistAndNormal 测试白名单和普通请求混合。
func TestBatch_MixedResults_WhitelistAndNormal(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "vip", Act: "login", UID: "vip_user_001"},     // 白名单
			{ID: "normal", Act: "test", UID: UniqueID("user")}, // 普通
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 2)

	// 两个都应该通过
	for _, result := range resp.Results {
		assert.True(t, result.Allowed)
	}
}

// TestBatch_MixedResults_BlacklistAndNormal 测试黑名单和普通请求混合。
func TestBatch_MixedResults_BlacklistAndNormal(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "blocked", Act: "login", UID: UniqueID("user"), IP: "192.168.100.1"}, // 黑名单IP
			{ID: "normal", Act: "test", UID: UniqueID("user")},                        // 普通
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 2)

	// 找到各自的结果
	var blockedResult, normalResult *BatchResult
	for i := range resp.Results {
		if resp.Results[i].ID == "blocked" {
			blockedResult = &resp.Results[i]
		} else if resp.Results[i].ID == "normal" {
			normalResult = &resp.Results[i]
		}
	}

	require.NotNil(t, blockedResult)
	require.NotNil(t, normalResult)

	assert.False(t, blockedResult.Allowed)
	assert.Equal(t, 4003, blockedResult.Code)
	assert.True(t, normalResult.Allowed)
}

// TestBatch_MixedResults_RateLimited 测试限流和正常请求混合。
func TestBatch_MixedResults_RateLimited(t *testing.T) {
	s := GetTestServer(t)
	uid := UniqueID("batch_ratelimit")

	// 先触发限流
	for i := 0; i < 6; i++ {
		s.Update(APIRequest{Act: "login", UID: uid})
	}

	// 批量检查
	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "limited", Act: "login", UID: uid},            // 被限流
			{ID: "other", Act: "test", UID: UniqueID("other")}, // 正常
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 2)

	// 找到各自的结果
	var limitedResult, otherResult *BatchResult
	for i := range resp.Results {
		if resp.Results[i].ID == "limited" {
			limitedResult = &resp.Results[i]
		} else if resp.Results[i].ID == "other" {
			otherResult = &resp.Results[i]
		}
	}

	require.NotNil(t, limitedResult)
	require.NotNil(t, otherResult)

	assert.False(t, limitedResult.Allowed)
	assert.True(t, otherResult.Allowed)
}

// ========== 边界场景测试 ==========

// TestBatch_EmptyRequests 测试空请求列表。
func TestBatch_EmptyRequests(t *testing.T) {
	s := GetTestServer(t)

	httpResp, err := s.PostJSON("/api/v1/batch", BatchRequest{
		Requests: []BatchItem{},
	})
	require.NoError(t, err)
	defer httpResp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, httpResp.StatusCode)
}

// TestBatch_TooManyRequests 测试超过限制的请求数量（101条）。
func TestBatch_TooManyRequests(t *testing.T) {
	s := GetTestServer(t)

	requests := make([]BatchItem, 101)
	for i := 0; i < 101; i++ {
		requests[i] = BatchItem{
			ID:  intToStr(i),
			Act: "test",
			UID: UniqueID("user"),
		}
	}

	httpResp, err := s.PostJSON("/api/v1/batch", BatchRequest{Requests: requests})
	require.NoError(t, err)
	defer httpResp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, httpResp.StatusCode)
}

// TestBatch_MissingID 测试缺少ID的请求项。
func TestBatch_MissingID(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "", Act: "test", UID: UniqueID("user")}, // 空ID
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 1)

	// 缺少ID的请求应该返回错误
	assert.False(t, resp.Results[0].Allowed)
	assert.Equal(t, -1, resp.Results[0].Code)
}

// TestBatch_MissingAct 测试缺少Act的请求项。
func TestBatch_MissingAct(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "req1", Act: "", UID: UniqueID("user")}, // 空Act
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 1)

	// 缺少Act的请求应该返回错误
	assert.False(t, resp.Results[0].Allowed)
	assert.Equal(t, -1, resp.Results[0].Code)
}

// TestBatch_DuplicateIDs 测试重复ID。
func TestBatch_DuplicateIDs(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "same_id", Act: "test1", UID: UniqueID("user1")},
			{ID: "same_id", Act: "test2", UID: UniqueID("user2")}, // 相同ID
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	// 应该返回两个结果，即使ID相同
	assert.Len(t, resp.Results, 2)
}

// TestBatch_SpecialCharactersInID 测试ID中的特殊字符。
func TestBatch_SpecialCharactersInID(t *testing.T) {
	s := GetTestServer(t)

	testCases := []string{
		"id_中文",
		"id!@#$%",
		"id with space",
		"id\nwith\nnewline",
		"id_😀",
	}

	requests := make([]BatchItem, len(testCases))
	for i, id := range testCases {
		requests[i] = BatchItem{
			ID:  id,
			Act: "test",
			UID: UniqueID("user"),
		}
	}

	resp, httpResp, err := s.Batch(BatchRequest{Requests: requests})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Len(t, resp.Results, len(testCases))

	// 验证所有ID都正确返回
	returnedIDs := make(map[string]bool)
	for _, result := range resp.Results {
		returnedIDs[result.ID] = true
	}
	for _, id := range testCases {
		assert.True(t, returnedIDs[id], "ID %s not found in response", id)
	}
}

// TestBatch_LongID 测试超长ID。
func TestBatch_LongID(t *testing.T) {
	s := GetTestServer(t)

	longID := RepeatString("a", 1000)
	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: longID, Act: "test", UID: UniqueID("user")},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, longID, resp.Results[0].ID)
}

// ========== 错误场景测试 ==========

// TestBatch_InvalidJSON 测试无效JSON。
func TestBatch_InvalidJSON(t *testing.T) {
	s := GetTestServer(t)

	testCases := []struct {
		name string
		body string
	}{
		{"空字符串", ""},
		{"无效JSON", "invalid"},
		{"不完整JSON", `{"requests":`},
		{"requests不是数组", `{"requests": "string"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.PostRaw("/api/v1/batch", []byte(tc.body), "application/json")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// TestBatch_WrongMethod 测试错误的HTTP方法。
func TestBatch_WrongMethod(t *testing.T) {
	s := GetTestServer(t)

	methods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, s.BaseURL()+"/api/v1/batch", nil)
			resp, err := s.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.True(t, resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound)
		})
	}
}

// ========== 响应验证测试 ==========

// TestBatch_ResponseFormat 测试响应格式。
func TestBatch_ResponseFormat(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "req1", Act: "test", UID: UniqueID("user")},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", httpResp.Header.Get("Content-Type"))

	// 验证响应结构
	require.Len(t, resp.Results, 1)
	result := resp.Results[0]
	assert.NotEmpty(t, result.ID)
	// Allowed, Code 是必有值
}

// TestBatch_ResponseOrder 测试响应顺序是否与请求顺序一致。
func TestBatch_ResponseOrder(t *testing.T) {
	s := GetTestServer(t)

	requests := make([]BatchItem, 10)
	for i := 0; i < 10; i++ {
		requests[i] = BatchItem{
			ID:  intToStr(i),
			Act: "test",
			UID: UniqueID("user"),
		}
	}

	resp, httpResp, err := s.Batch(BatchRequest{Requests: requests})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 10)

	// 验证顺序
	for i, result := range resp.Results {
		assert.Equal(t, intToStr(i), result.ID)
	}
}

// TestBatch_RuleNameAndAuthType 测试规则名称和认证类型返回。
func TestBatch_RuleNameAndAuthType(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "whitelist", Act: "login", UID: "vip_user_001"},      // 白名单
			{ID: "blacklist", Act: "login", IP: "192.168.100.1"},      // 黑名单
			{ID: "normal", Act: "test", UID: UniqueID("normal_user")}, // 普通
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 3)

	// 找到各结果
	results := make(map[string]*BatchResult)
	for i := range resp.Results {
		results[resp.Results[i].ID] = &resp.Results[i]
	}

	// 白名单应该有rule_name
	assert.NotEmpty(t, results["whitelist"].RuleName)
	// 黑名单应该有rule_name和auth_type
	assert.NotEmpty(t, results["blacklist"].RuleName)
}

// ========== 并发测试 ==========

// TestBatch_ConcurrentBatchRequests 测试并发批量请求。
func TestBatch_ConcurrentBatchRequests(t *testing.T) {
	s := GetTestServer(t)

	done := make(chan bool, 20)

	for i := 0; i < 20; i++ {
		go func(idx int) {
			resp, httpResp, err := s.Batch(BatchRequest{
				Requests: []BatchItem{
					{ID: intToStr(idx) + "_1", Act: "test", UID: UniqueID("user")},
					{ID: intToStr(idx) + "_2", Act: "test", UID: UniqueID("user")},
					{ID: intToStr(idx) + "_3", Act: "test", UID: UniqueID("user")},
				},
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, httpResp.StatusCode)
			assert.Len(t, resp.Results, 3)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 20; i++ {
		<-done
	}
}

// ========== 性能相关测试 ==========

// TestBatch_LargeBatchPerformance 测试大批量请求性能。
func TestBatch_LargeBatchPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}

	s := GetTestServer(t)

	// 创建100个请求
	requests := make([]BatchItem, 100)
	for i := 0; i < 100; i++ {
		requests[i] = BatchItem{
			ID:  intToStr(i),
			Act: "test",
			UID: UniqueID("perf_user"),
			IP:  "192.168.1." + intToStr(i%256),
			DID: "device_" + intToStr(i),
			Ext: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}
	}

	resp, httpResp, err := s.Batch(BatchRequest{Requests: requests})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Len(t, resp.Results, 100)
}

// ========== 辅助函数 ==========

// intToStr 整数转字符串。
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}
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

// ========== 部分成功测试 ==========

// TestBatch_PartialValidationFailure 测试部分请求验证失败。
func TestBatch_PartialValidationFailure(t *testing.T) {
	s := GetTestServer(t)

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "valid", Act: "test", UID: UniqueID("user")}, // 有效
			{ID: "", Act: "test", UID: UniqueID("user")},      // ID为空
			{ID: "valid2", Act: "", UID: UniqueID("user")},    // Act为空
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 3)

	// 第一个应该成功
	assert.True(t, resp.Results[0].Allowed)
	// 后两个应该失败
	assert.False(t, resp.Results[1].Allowed)
	assert.False(t, resp.Results[2].Allowed)
}

// ========== 请求体大小测试 ==========

// TestBatch_LargeExtFields 测试大型扩展字段。
func TestBatch_LargeExtFields(t *testing.T) {
	s := GetTestServer(t)

	// 创建包含大量扩展字段的请求
	ext := make(map[string]string)
	for i := 0; i < 50; i++ {
		ext["key_"+intToStr(i)] = RepeatString("value", 20)
	}

	resp, httpResp, err := s.Batch(BatchRequest{
		Requests: []BatchItem{
			{ID: "large_ext", Act: "test", UID: UniqueID("user"), Ext: ext},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	require.Len(t, resp.Results, 1)
	assert.True(t, resp.Results[0].Allowed)
}

// TestBatch_MixedValidAndInvalid 测试有效和无效请求混合。
func TestBatch_MixedValidAndInvalid(t *testing.T) {
	s := GetTestServer(t)

	httpResp, err := s.PostJSON("/api/v1/batch", map[string]interface{}{
		"requests": []map[string]interface{}{
			{"id": "valid", "act": "test", "uid": UniqueID("user")},
			{"id": 123, "act": "test"}, // id应该是字符串
		},
	})
	require.NoError(t, err)
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)
	var resp BatchResponse
	json.Unmarshal(body, &resp)

	// 应该能处理，可能有部分失败
	assert.True(t, httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusBadRequest)
}
