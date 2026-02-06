// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package manager provides storage management with automatic failover.
package manager

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koala/internal/storage"
	"koala/internal/storage/local"
)

// =============================================================================
// Basic Manager Tests
// =============================================================================

func TestStorageManager_LocalOnly(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mgr := New(localStorage, nil)
	defer mgr.Close()

	ctx := context.Background()

	// Should work with local storage
	err = mgr.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	val, err := mgr.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestStorageManager_CurrentStorage(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mgr := New(localStorage, nil)
	defer mgr.Close()

	assert.Equal(t, "local", mgr.CurrentType())
	assert.False(t, mgr.IsDegraded())
}

func TestStorageManager_WithPrimary(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	// Use a mock primary that fails
	mockPrimary := &mockStorage{failPing: true}

	mgr := NewWithOptions(Options{
		Primary:             mockPrimary,
		Fallback:            localStorage,
		MaxFailures:         1,
		HealthCheckInterval: 100 * time.Millisecond,
	})
	defer mgr.Close()

	// Should degrade to local since primary fails ping
	time.Sleep(50 * time.Millisecond)

	assert.True(t, mgr.IsDegraded())
	assert.Equal(t, "local", mgr.CurrentType())
}

func TestStorageManager_Failover(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mockPrimary := &mockStorage{failAll: true}

	mgr := NewWithOptions(Options{
		Primary:        mockPrimary,
		Fallback:       localStorage,
		MaxFailures:    2,
	})
	defer mgr.Close()

	ctx := context.Background()

	// First few operations should fail and trigger failover
	_, _ = mgr.Get(ctx, "key1")
	_, _ = mgr.Get(ctx, "key1")
	_, _ = mgr.Get(ctx, "key1")

	// After failures, should have degraded
	assert.True(t, mgr.IsDegraded())

	// Operations should now use local storage
	err = mgr.Set(ctx, "key1", "value1", 0)
	assert.NoError(t, err)

	val, err := mgr.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestStorageManager_Recovery(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mockPrimary := &mockStorage{failPing: true}

	mgr := NewWithOptions(Options{
		Primary:             mockPrimary,
		Fallback:            localStorage,
		MaxFailures:         1,
		HealthCheckInterval: 50 * time.Millisecond,
		RecoveryInterval:    50 * time.Millisecond,
	})
	defer mgr.Close()

	// Wait for degradation (initial check should trigger it)
	time.Sleep(30 * time.Millisecond)
	assert.True(t, mgr.IsDegraded())

	// Fix the primary
	mockPrimary.mu.Lock()
	mockPrimary.failPing = false
	mockPrimary.failAll = false
	mockPrimary.mu.Unlock()

	// Wait for recovery (next health check should recover)
	time.Sleep(100 * time.Millisecond)

	// Should recover to primary
	assert.False(t, mgr.IsDegraded())
}

// =============================================================================
// Counter Operations Tests
// =============================================================================

func TestStorageManager_Incr(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mgr := New(localStorage, nil)
	defer mgr.Close()

	ctx := context.Background()

	val, err := mgr.Incr(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = mgr.Incr(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestStorageManager_IncrWithTTL(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mgr := New(localStorage, nil)
	defer mgr.Close()

	ctx := context.Background()

	val, err := mgr.IncrWithTTL(ctx, "counter1", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	time.Sleep(150 * time.Millisecond)

	val, err = mgr.IncrWithTTL(ctx, "counter1", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val) // Reset after TTL
}

// =============================================================================
// List Operations Tests
// =============================================================================

func TestStorageManager_List(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mgr := New(localStorage, nil)
	defer mgr.Close()

	ctx := context.Background()

	err = mgr.LPush(ctx, "list1", 1, 2, 3)
	assert.NoError(t, err)

	length, err := mgr.LLen(ctx, "list1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)

	values, err := mgr.LRange(ctx, "list1", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []int64{3, 2, 1}, values)
}

// =============================================================================
// Failover Tests - 故障转移测试
// =============================================================================

// newFailoverManager 创建一个用于故障转移测试的管理器。
// primary 设置为 failAll=true，MaxFailures=1 以便在首次失败时立即降级。
// fallback 使用真实的 local 存储。
func newFailoverManager(t *testing.T) (*StorageManager, *local.LocalStorage) {
	t.Helper()
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)

	mockPrimary := &mockStorage{failAll: true}

	mgr := NewWithOptions(Options{
		Primary:             mockPrimary,
		Fallback:            localStorage,
		MaxFailures:         1, // 一次失败就降级
		HealthCheckInterval: 1 * time.Hour, // 禁用健康检查避免干扰测试
	})

	return mgr, localStorage
}

func TestManager_Incr_Failover(t *testing.T) {
	// 测试 Incr 在主存储失败时能降级到备用存储
	mgr, _ := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 第一次调用触发主存储失败并记录故障，主存储 MaxFailures=1 所以立即降级
	// 降级后应使用 fallback 重试
	val, err := mgr.Incr(ctx, "counter1")
	// 降级后 fallback 应成功，第一次 Incr 应返回 1
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// 已降级，后续操作直接使用 fallback
	val, err = mgr.Incr(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestManager_IncrWithTTL_Failover(t *testing.T) {
	// 测试 IncrWithTTL 在主存储失败时能降级到备用存储
	mgr, _ := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	val, err := mgr.IncrWithTTL(ctx, "counter1", 10*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = mgr.IncrWithTTL(ctx, "counter1", 10*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestManager_GetInt_Failover(t *testing.T) {
	// 测试 GetInt 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中设置数据
	err := localStorage.SetInt(ctx, "intkey", 42, 0)
	require.NoError(t, err)

	// GetInt 应该在主存储失败后降级到 fallback 并返回正确值
	val, err := mgr.GetInt(ctx, "intkey")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), val)
}

func TestManager_LPush_Failover(t *testing.T) {
	// 测试 LPush 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// LPush 应该在主存储失败后降级到 fallback
	err := mgr.LPush(ctx, "list1", 10, 20, 30)
	assert.NoError(t, err)

	// 验证数据确实写入了 fallback
	length, err := localStorage.LLen(ctx, "list1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

func TestManager_LLen_Failover(t *testing.T) {
	// 测试 LLen 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.LPush(ctx, "list1", 1, 2, 3)
	require.NoError(t, err)

	// LLen 应该在主存储失败后降级到 fallback
	length, err := mgr.LLen(ctx, "list1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

func TestManager_LRange_Failover(t *testing.T) {
	// 测试 LRange 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.LPush(ctx, "list1", 1, 2, 3)
	require.NoError(t, err)

	// LRange 应该在主存储失败后降级到 fallback
	values, err := mgr.LRange(ctx, "list1", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []int64{3, 2, 1}, values)
}

func TestManager_Delete_Failover(t *testing.T) {
	// 测试 Delete 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	// Delete 应该在主存储失败后降级到 fallback
	err = mgr.Delete(ctx, "key1")
	assert.NoError(t, err)

	// 验证数据确实从 fallback 中删除了
	_, err = localStorage.Get(ctx, "key1")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestManager_Exists_Failover(t *testing.T) {
	// 测试 Exists 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	// Exists 应该在主存储失败后降级到 fallback
	exists, err := mgr.Exists(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestManager_Expire_Failover(t *testing.T) {
	// 测试 Expire 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	// Expire 应该在主存储失败后降级到 fallback
	err = mgr.Expire(ctx, "key1", 10*time.Second)
	assert.NoError(t, err)
}

func TestManager_SetInt_Failover(t *testing.T) {
	// 测试 SetInt 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// SetInt 应该在主存储失败后降级到 fallback
	err := mgr.SetInt(ctx, "intkey", 100, 0)
	assert.NoError(t, err)

	// 验证数据确实写入了 fallback
	val, err := localStorage.GetInt(ctx, "intkey")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), val)
}

func TestManager_IncrBy_Failover(t *testing.T) {
	// 测试 IncrBy 在主存储失败时能降级到备用存储
	mgr, _ := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	val, err := mgr.IncrBy(ctx, "counter1", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), val)

	val, err = mgr.IncrBy(ctx, "counter1", 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(8), val)
}

func TestManager_LIndex_Failover(t *testing.T) {
	// 测试 LIndex 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.LPush(ctx, "list1", 10, 20, 30)
	require.NoError(t, err)

	// LIndex 应该在主存储失败后降级到 fallback
	val, err := mgr.LIndex(ctx, "list1", 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(30), val) // LPush 左插入，最后一个在最前
}

func TestManager_LTrim_Failover(t *testing.T) {
	// 测试 LTrim 在主存储失败时能降级到备用存储
	mgr, localStorage := newFailoverManager(t)
	defer mgr.Close()

	ctx := context.Background()

	// 先在 fallback 中准备数据
	err := localStorage.LPush(ctx, "list1", 1, 2, 3, 4, 5)
	require.NoError(t, err)

	// LTrim 应该在主存储失败后降级到 fallback
	err = mgr.LTrim(ctx, "list1", 0, 2)
	assert.NoError(t, err)

	// 验证列表确实被裁剪了
	length, err := localStorage.LLen(ctx, "list1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

// TestManager_FailoverAlreadyDegraded 测试已经降级后操作直接使用 fallback
func TestManager_FailoverAlreadyDegraded(t *testing.T) {
	localStorage, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	defer localStorage.Close()

	mockPrimary := &mockStorage{failAll: true}

	mgr := NewWithOptions(Options{
		Primary:             mockPrimary,
		Fallback:            localStorage,
		MaxFailures:         1,
		HealthCheckInterval: 1 * time.Hour,
	})
	defer mgr.Close()

	ctx := context.Background()

	// 先触发降级
	_, _ = mgr.Get(ctx, "trigger")
	assert.True(t, mgr.IsDegraded())

	// 降级后所有操作应该直接使用 fallback，不会出错
	err = mgr.Set(ctx, "k1", "v1", 0)
	assert.NoError(t, err)

	val, err := mgr.Get(ctx, "k1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", val)

	_, err = mgr.Incr(ctx, "c1")
	assert.NoError(t, err)

	_, err = mgr.IncrWithTTL(ctx, "c2", time.Second)
	assert.NoError(t, err)

	err = mgr.LPush(ctx, "l1", 1, 2)
	assert.NoError(t, err)

	_, err = mgr.LLen(ctx, "l1")
	assert.NoError(t, err)

	_, err = mgr.LRange(ctx, "l1", 0, -1)
	assert.NoError(t, err)

	_, err = mgr.Exists(ctx, "k1")
	assert.NoError(t, err)

	err = mgr.Delete(ctx, "k1")
	assert.NoError(t, err)
}

// =============================================================================
// Mock Storage for Testing
// =============================================================================

type mockStorage struct {
	mu       sync.Mutex
	failPing bool
	failAll  bool
	data     map[string]string
}

func (m *mockStorage) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failAll {
		return "", errMockFail
	}
	if m.data == nil {
		return "", storage.ErrKeyNotFound
	}
	val, ok := m.data[key]
	if !ok {
		return "", storage.ErrKeyNotFound
	}
	return val, nil
}

func (m *mockStorage) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failAll {
		return errMockFail
	}
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = value
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failAll {
		return errMockFail
	}
	delete(m.data, key)
	return nil
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failAll {
		return false, errMockFail
	}
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if m.failAll {
		return errMockFail
	}
	return nil
}

func (m *mockStorage) GetInt(ctx context.Context, key string) (int64, error) {
	if m.failAll {
		return 0, errMockFail
	}
	return 0, storage.ErrKeyNotFound
}

func (m *mockStorage) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	if m.failAll {
		return errMockFail
	}
	return nil
}

func (m *mockStorage) Incr(ctx context.Context, key string) (int64, error) {
	if m.failAll {
		return 0, errMockFail
	}
	return 1, nil
}

func (m *mockStorage) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	if m.failAll {
		return 0, errMockFail
	}
	return delta, nil
}

func (m *mockStorage) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if m.failAll {
		return 0, errMockFail
	}
	return 1, nil
}

func (m *mockStorage) LPush(ctx context.Context, key string, values ...int64) error {
	if m.failAll {
		return errMockFail
	}
	return nil
}

func (m *mockStorage) LLen(ctx context.Context, key string) (int64, error) {
	if m.failAll {
		return 0, errMockFail
	}
	return 0, nil
}

func (m *mockStorage) LIndex(ctx context.Context, key string, index int64) (int64, error) {
	if m.failAll {
		return 0, errMockFail
	}
	return 0, nil
}

func (m *mockStorage) LTrim(ctx context.Context, key string, start, stop int64) error {
	if m.failAll {
		return errMockFail
	}
	return nil
}

func (m *mockStorage) LRange(ctx context.Context, key string, start, stop int64) ([]int64, error) {
	if m.failAll {
		return nil, errMockFail
	}
	return []int64{}, nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failPing || m.failAll {
		return errMockFail
	}
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func (m *mockStorage) Type() string {
	return "mock"
}

var errMockFail = storage.ErrStorageClosed
