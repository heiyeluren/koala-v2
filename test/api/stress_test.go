// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供 API 压力测试。
package api

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== 并发压力测试 ==========

// TestStress_BrowseConcurrent100 测试100并发Browse请求。
func TestStress_BrowseConcurrent100(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	concurrency := 100
	var successCount int64
	var failCount int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()

			resp, httpResp, err := s.Browse(APIRequest{
				Act: "stress_test",
				UID: UniqueID("user"),
			})

			if err != nil || httpResp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failCount, 1)
				return
			}

			if resp.Allowed {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("并发100: 成功=%d, 失败=%d", successCount, failCount)
	assert.Equal(t, int64(0), failCount, "不应有失败请求")
}

// TestStress_BrowseConcurrent500 测试500并发Browse请求。
func TestStress_BrowseConcurrent500(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	concurrency := 500
	var successCount int64
	var failCount int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()

			resp, httpResp, err := s.Browse(APIRequest{
				Act: "stress_test_500",
				UID: UniqueID("user"),
			})

			if err != nil || httpResp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failCount, 1)
				return
			}

			if resp.Allowed {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("并发500: 成功=%d, 失败=%d", successCount, failCount)
	assert.True(t, failCount < 10, "失败请求应少于10个")
}

// TestStress_UpdateConcurrent100 测试100并发Update请求。
func TestStress_UpdateConcurrent100(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	concurrency := 100
	uid := UniqueID("concurrent_update_user")
	var successCount int64
	var failCount int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			resp, httpResp, err := s.Update(APIRequest{
				Act: "concurrent_update",
				UID: uid, // 相同用户，测试并发安全
			})

			if err != nil || httpResp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failCount, 1)
				return
			}

			if resp.Allowed {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("并发Update: 成功=%d, 失败=%d", successCount, failCount)
	assert.Equal(t, int64(0), failCount, "不应有失败请求")
}

// TestStress_BatchConcurrent50 测试50并发Batch请求。
func TestStress_BatchConcurrent50(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	concurrency := 50
	var successCount int64
	var failCount int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()

			// 每个batch包含10个请求
			requests := make([]BatchItem, 10)
			for j := 0; j < 10; j++ {
				requests[j] = BatchItem{
					ID:  intToStr(idx*10 + j),
					Act: "batch_stress",
					UID: UniqueID("user"),
				}
			}

			resp, httpResp, err := s.Batch(BatchRequest{Requests: requests})

			if err != nil || httpResp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failCount, 1)
				return
			}

			if len(resp.Results) == 10 {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("并发Batch: 成功=%d, 失败=%d", successCount, failCount)
	assert.Equal(t, int64(0), failCount, "不应有失败请求")
}

// ========== 持续压力测试 ==========

// TestStress_SustainedLoad 测试持续负载。
func TestStress_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	duration := 5 * time.Second
	concurrency := 50

	var totalRequests int64
	var failedRequests int64

	ctx := make(chan struct{})
	var wg sync.WaitGroup

	// 启动工作协程
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx:
					return
				default:
					_, httpResp, err := s.Browse(APIRequest{
						Act: "sustained_load",
						UID: UniqueID("user"),
					})

					atomic.AddInt64(&totalRequests, 1)

					if err != nil || httpResp.StatusCode != http.StatusOK {
						atomic.AddInt64(&failedRequests, 1)
					}
				}
			}
		}()
	}

	// 运行指定时间
	time.Sleep(duration)
	close(ctx)
	wg.Wait()

	total := atomic.LoadInt64(&totalRequests)
	failed := atomic.LoadInt64(&failedRequests)
	qps := float64(total) / duration.Seconds()

	t.Logf("持续负载测试: 总请求=%d, 失败=%d, QPS=%.2f", total, failed, qps)
	assert.True(t, failed < total/100, "失败率应低于1%%")
}

// ========== 混合负载测试 ==========

// TestStress_MixedWorkload 测试混合负载。
func TestStress_MixedWorkload(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	duration := 3 * time.Second

	var browseCount, updateCount, batchCount int64
	var browseErr, updateErr, batchErr int64

	ctx := make(chan struct{})
	var wg sync.WaitGroup

	// Browse 工作协程
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx:
					return
				default:
					_, httpResp, err := s.Browse(APIRequest{
						Act: "mixed_browse",
						UID: UniqueID("user"),
					})
					atomic.AddInt64(&browseCount, 1)
					if err != nil || httpResp.StatusCode != http.StatusOK {
						atomic.AddInt64(&browseErr, 1)
					}
				}
			}
		}()
	}

	// Update 工作协程
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx:
					return
				default:
					_, httpResp, err := s.Update(APIRequest{
						Act: "mixed_update",
						UID: UniqueID("user"),
					})
					atomic.AddInt64(&updateCount, 1)
					if err != nil || httpResp.StatusCode != http.StatusOK {
						atomic.AddInt64(&updateErr, 1)
					}
				}
			}
		}()
	}

	// Batch 工作协程
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx:
					return
				default:
					_, httpResp, err := s.Batch(BatchRequest{
						Requests: []BatchItem{
							{ID: "1", Act: "mixed_batch", UID: UniqueID("user")},
							{ID: "2", Act: "mixed_batch", UID: UniqueID("user")},
						},
					})
					atomic.AddInt64(&batchCount, 1)
					if err != nil || httpResp.StatusCode != http.StatusOK {
						atomic.AddInt64(&batchErr, 1)
					}
				}
			}
		}()
	}

	// Health 检查协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx:
				return
			default:
				s.Health()
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	time.Sleep(duration)
	close(ctx)
	wg.Wait()

	t.Logf("混合负载测试:")
	t.Logf("  Browse: %d (错误: %d)", browseCount, browseErr)
	t.Logf("  Update: %d (错误: %d)", updateCount, updateErr)
	t.Logf("  Batch: %d (错误: %d)", batchCount, batchErr)

	totalErr := browseErr + updateErr + batchErr
	assert.Equal(t, int64(0), totalErr, "不应有错误请求")
}

// ========== 同一用户高并发测试 ==========

// TestStress_SameUserConcurrent 测试同一用户高并发请求。
func TestStress_SameUserConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	uid := UniqueID("single_user")
	concurrency := 100
	var successCount int64
	var rateLimitedCount int64
	var errorCount int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			resp, httpResp, err := s.Browse(APIRequest{
				Act:    "login", // 会触发限流规则
				UID:    uid,
				Update: true,
			})

			if err != nil || httpResp.StatusCode != http.StatusOK {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			if resp.Allowed {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&rateLimitedCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("同用户并发: 成功=%d, 限流=%d, 错误=%d", successCount, rateLimitedCount, errorCount)
	assert.Equal(t, int64(0), errorCount, "不应有错误请求")
	assert.True(t, rateLimitedCount > 0, "应该有请求被限流")
}

// ========== 渐进式负载测试 ==========

// TestStress_RampUp 测试渐进式增加负载。
func TestStress_RampUp(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)

	levels := []int{10, 50, 100, 200}

	for _, concurrency := range levels {
		t.Run(intToStr(concurrency)+"并发", func(t *testing.T) {
			var successCount int64
			var failCount int64

			var wg sync.WaitGroup
			wg.Add(concurrency)

			start := time.Now()

			for i := 0; i < concurrency; i++ {
				go func() {
					defer wg.Done()

					resp, httpResp, err := s.Browse(APIRequest{
						Act: "ramp_up",
						UID: UniqueID("user"),
					})

					if err != nil || httpResp.StatusCode != http.StatusOK {
						atomic.AddInt64(&failCount, 1)
						return
					}

					if resp.Allowed {
						atomic.AddInt64(&successCount, 1)
					}
				}()
			}

			wg.Wait()
			elapsed := time.Since(start)

			t.Logf("并发%d: 成功=%d, 失败=%d, 耗时=%v", concurrency, successCount, failCount, elapsed)
			assert.Equal(t, int64(0), failCount, "不应有失败请求")
		})
	}
}

// ========== 大请求体测试 ==========

// TestStress_LargeRequestBody 测试大请求体。
func TestStress_LargeRequestBody(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)

	// 创建大型扩展字段
	ext := make(map[string]string)
	for i := 0; i < 100; i++ {
		ext["key_"+intToStr(i)] = RepeatString("value", 50)
	}

	concurrency := 50
	var successCount int64
	var failCount int64

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			resp, httpResp, err := s.Browse(APIRequest{
				Act: "large_body",
				UID: UniqueID("user"),
				Ext: ext,
			})

			if err != nil || httpResp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failCount, 1)
				return
			}

			if resp.Allowed {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("大请求体: 成功=%d, 失败=%d", successCount, failCount)
	assert.Equal(t, int64(0), failCount, "不应有失败请求")
}

// ========== 健康检查压力测试 ==========

// TestStress_HealthCheckUnderLoad 测试高负载下的健康检查。
func TestStress_HealthCheckUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	s := GetTestServer(t)
	duration := 3 * time.Second

	var browseCount int64
	var healthCheckOK int64
	var healthCheckFail int64

	ctx := make(chan struct{})
	var wg sync.WaitGroup

	// 高负载 Browse 请求
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx:
					return
				default:
					s.Browse(APIRequest{
						Act: "health_under_load",
						UID: UniqueID("user"),
					})
					atomic.AddInt64(&browseCount, 1)
				}
			}
		}()
	}

	// 健康检查
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx:
				return
			default:
				resp, httpResp, err := s.Health()
				if err != nil || httpResp.StatusCode != http.StatusOK || resp.Status != "ok" {
					atomic.AddInt64(&healthCheckFail, 1)
				} else {
					atomic.AddInt64(&healthCheckOK, 1)
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	time.Sleep(duration)
	close(ctx)
	wg.Wait()

	t.Logf("高负载健康检查: Browse=%d, 健康检查成功=%d, 失败=%d",
		browseCount, healthCheckOK, healthCheckFail)
	assert.Equal(t, int64(0), healthCheckFail, "健康检查不应失败")
}

// ========== 基准测试 ==========

// BenchmarkBrowse 基准测试Browse接口。
func BenchmarkBrowse(b *testing.B) {
	s, err := NewTestServer()
	require.NoError(b, err)
	require.NoError(b, s.Start())
	defer s.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Browse(APIRequest{
				Act: "benchmark",
				UID: UniqueID("user"),
			})
		}
	})
}

// BenchmarkUpdate 基准测试Update接口。
func BenchmarkUpdate(b *testing.B) {
	s, err := NewTestServer()
	require.NoError(b, err)
	require.NoError(b, s.Start())
	defer s.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Update(APIRequest{
				Act: "benchmark",
				UID: UniqueID("user"),
			})
		}
	})
}

// BenchmarkBatch10 基准测试Batch接口（10条）。
func BenchmarkBatch10(b *testing.B) {
	s, err := NewTestServer()
	require.NoError(b, err)
	require.NoError(b, s.Start())
	defer s.Stop()

	requests := make([]BatchItem, 10)
	for i := 0; i < 10; i++ {
		requests[i] = BatchItem{
			ID:  intToStr(i),
			Act: "benchmark",
			UID: UniqueID("user"),
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Batch(BatchRequest{Requests: requests})
		}
	})
}

// BenchmarkHealth 基准测试Health接口。
func BenchmarkHealth(b *testing.B) {
	s, err := NewTestServer()
	require.NoError(b, err)
	require.NoError(b, s.Start())
	defer s.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Health()
		}
	})
}
