// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package local provides a local in-memory storage implementation for Koala.
package local

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koala/internal/storage"
)

// =============================================================================
// String Operations Tests
// =============================================================================

func TestLocalStorage_Get_NotFound(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	val, err := store.Get(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
	assert.Equal(t, "", val)
}

func TestLocalStorage_Set_And_Get(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	val, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)
}

func TestLocalStorage_Set_WithTTL(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Set(ctx, "key1", "value1", 100*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	val, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should not exist after TTL
	_, err = store.Get(ctx, "key1")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestLocalStorage_Set_Overwrite(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	err = store.Set(ctx, "key1", "value2", 0)
	require.NoError(t, err)

	val, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value2", val)
}

func TestLocalStorage_Delete(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	err = store.Delete(ctx, "key1")
	assert.NoError(t, err)

	_, err = store.Get(ctx, "key1")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestLocalStorage_Delete_NotFound(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	// Delete non-existent key should not error
	err := store.Delete(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

func TestLocalStorage_Exists(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()

	exists, err := store.Exists(ctx, "key1")
	assert.NoError(t, err)
	assert.False(t, exists)

	err = store.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	exists, err = store.Exists(ctx, "key1")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestLocalStorage_Expire(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)

	err = store.Expire(ctx, "key1", 100*time.Millisecond)
	assert.NoError(t, err)

	// Should still exist
	val, err := store.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// Wait for TTL
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, err = store.Get(ctx, "key1")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestLocalStorage_Expire_NotFound(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	err := store.Expire(context.Background(), "nonexistent", time.Second)
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

// =============================================================================
// Counter Operations Tests
// =============================================================================

func TestLocalStorage_GetInt_NotFound(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	val, err := store.GetInt(context.Background(), "counter1")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
	assert.Equal(t, int64(0), val)
}

func TestLocalStorage_SetInt_And_GetInt(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.SetInt(ctx, "counter1", 100, 0)
	require.NoError(t, err)

	val, err := store.GetInt(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), val)
}

func TestLocalStorage_SetInt_WithTTL(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.SetInt(ctx, "counter1", 100, 100*time.Millisecond)
	require.NoError(t, err)

	val, err := store.GetInt(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), val)

	time.Sleep(150 * time.Millisecond)

	_, err = store.GetInt(ctx, "counter1")
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestLocalStorage_Incr(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()

	// Incr on non-existent key should create it with value 1
	val, err := store.Incr(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Incr again
	val, err = store.Incr(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)

	// Verify with GetInt
	val, err = store.GetInt(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)
}

func TestLocalStorage_IncrBy(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()

	val, err := store.IncrBy(ctx, "counter1", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), val)

	val, err = store.IncrBy(ctx, "counter1", 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), val)

	// Negative delta
	val, err = store.IncrBy(ctx, "counter1", -3)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), val)
}

func TestLocalStorage_IncrWithTTL(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()

	val, err := store.IncrWithTTL(ctx, "counter1", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = store.IncrWithTTL(ctx, "counter1", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)

	time.Sleep(150 * time.Millisecond)

	// Should be reset after TTL
	val, err = store.IncrWithTTL(ctx, "counter1", 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestLocalStorage_Incr_Concurrent(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := store.Incr(ctx, "counter1")
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()

	val, err := store.GetInt(ctx, "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(goroutines*iterations), val)
}

// =============================================================================
// List Operations Tests
// =============================================================================

func TestLocalStorage_LPush(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()

	err := store.LPush(ctx, "list1", 1, 2, 3)
	assert.NoError(t, err)

	length, err := store.LLen(ctx, "list1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)
}

func TestLocalStorage_LLen_NotFound(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	length, err := store.LLen(context.Background(), "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), length)
}

func TestLocalStorage_LIndex(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.LPush(ctx, "list1", 10, 20, 30)
	require.NoError(t, err)

	// LPush adds to the front, so order is [30, 20, 10]
	val, err := store.LIndex(ctx, "list1", 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(30), val)

	val, err = store.LIndex(ctx, "list1", 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(20), val)

	val, err = store.LIndex(ctx, "list1", 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	// Negative index: -1 is last element
	val, err = store.LIndex(ctx, "list1", -1)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)
}

func TestLocalStorage_LIndex_OutOfRange(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	err := store.LPush(ctx, "list1", 1, 2, 3)
	require.NoError(t, err)

	_, err = store.LIndex(ctx, "list1", 10)
	assert.ErrorIs(t, err, storage.ErrIndexOutOfRange)
}

func TestLocalStorage_LTrim(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	// Push 1, 2, 3, 4, 5 -> list is [5, 4, 3, 2, 1]
	err := store.LPush(ctx, "list1", 1, 2, 3, 4, 5)
	require.NoError(t, err)

	// Keep only elements 0-2: [5, 4, 3]
	err = store.LTrim(ctx, "list1", 0, 2)
	assert.NoError(t, err)

	length, err := store.LLen(ctx, "list1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)

	val, err := store.LIndex(ctx, "list1", 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), val)
}

func TestLocalStorage_LRange(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	ctx := context.Background()
	// Push 1, 2, 3, 4, 5 -> list is [5, 4, 3, 2, 1]
	err := store.LPush(ctx, "list1", 1, 2, 3, 4, 5)
	require.NoError(t, err)

	values, err := store.LRange(ctx, "list1", 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, []int64{5, 4, 3}, values)

	// Get all
	values, err = store.LRange(ctx, "list1", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []int64{5, 4, 3, 2, 1}, values)
}

func TestLocalStorage_LRange_NotFound(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	values, err := store.LRange(context.Background(), "nonexistent", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []int64{}, values)
}

// =============================================================================
// Connection Management Tests
// =============================================================================

func TestLocalStorage_Ping(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	err := store.Ping(context.Background())
	assert.NoError(t, err)
}

func TestLocalStorage_Close(t *testing.T) {
	store := newTestStorage(t)

	err := store.Close()
	assert.NoError(t, err)

	// Operations after close should fail
	_, err = store.Get(context.Background(), "key1")
	assert.ErrorIs(t, err, storage.ErrStorageClosed)
}

func TestLocalStorage_Type(t *testing.T) {
	store := newTestStorage(t)
	defer store.Close()

	assert.Equal(t, "local", store.Type())
}

// =============================================================================
// List Cleanup Tests（列表清理测试）
// =============================================================================

func TestLocalStorage_ListCleanup(t *testing.T) {
	// 测试后台清理协程能够移除空列表条目
	cfg := DefaultConfig()
	cfg.CleanupInterval = 50 * time.Millisecond

	store, err := New(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 插入列表数据
	err = store.LPush(ctx, "list1", 100, 200, 300)
	require.NoError(t, err)

	// 通过 LTrim 将列表清空（start > stop 会清空列表）
	err = store.LTrim(ctx, "list1", 1, 0)
	require.NoError(t, err)

	// 验证列表已被清空
	length, err := store.LLen(ctx, "list1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)

	// 等待清理协程运行（至少等待两个清理周期）
	time.Sleep(150 * time.Millisecond)

	// 验证空列表条目已从内部 map 中移除
	store.listsMu.RLock()
	_, exists := store.lists["list1"]
	store.listsMu.RUnlock()
	assert.False(t, exists, "空列表条目应该被清理协程移除")
}

func TestLocalStorage_CleanupDoesNotRemoveActiveData(t *testing.T) {
	// 测试清理协程不会移除仍有有效数据的列表
	cfg := DefaultConfig()
	cfg.CleanupInterval = 50 * time.Millisecond

	store, err := New(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 插入列表数据（模拟活跃的滑动窗口时间戳）
	now := time.Now().UnixMilli()
	err = store.LPush(ctx, "active_list", now, now-1000, now-2000)
	require.NoError(t, err)

	// 同时创建一个空列表用于对比
	err = store.LPush(ctx, "empty_list", 1)
	require.NoError(t, err)
	err = store.LTrim(ctx, "empty_list", 1, 0)
	require.NoError(t, err)

	// 等待清理协程运行
	time.Sleep(150 * time.Millisecond)

	// 验证活跃列表仍然存在
	length, err := store.LLen(ctx, "active_list")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)

	// 验证活跃列表数据完整
	values, err := store.LRange(ctx, "active_list", 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(values))

	// 验证空列表已被移除
	store.listsMu.RLock()
	_, emptyExists := store.lists["empty_list"]
	store.listsMu.RUnlock()
	assert.False(t, emptyExists, "空列表条目应该被清理协程移除")
}

func TestLocalStorage_CleanupStopsOnClose(t *testing.T) {
	// 测试 Close() 能正确停止清理协程
	cfg := DefaultConfig()
	cfg.CleanupInterval = 50 * time.Millisecond

	store, err := New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// 插入并清空一个列表
	err = store.LPush(ctx, "list1", 100)
	require.NoError(t, err)
	err = store.LTrim(ctx, "list1", 1, 0)
	require.NoError(t, err)

	// 关闭存储（应该停止清理协程）
	err = store.Close()
	require.NoError(t, err)

	// 关闭后不应 panic 或阻塞，重复关闭也应该安全
	err = store.Close()
	assert.NoError(t, err)
}

func TestLocalStorage_CleanupDisabledByDefault(t *testing.T) {
	// 测试当 CleanupInterval 为 0 时不启动清理协程
	cfg := DefaultConfig()
	// DefaultConfig 应该设置一个合理的默认值
	// 但设置为 0 应该禁用清理
	cfg.CleanupInterval = 0

	store, err := New(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 插入并清空一个列表
	err = store.LPush(ctx, "list1", 100)
	require.NoError(t, err)
	err = store.LTrim(ctx, "list1", 1, 0)
	require.NoError(t, err)

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 当清理被禁用时，空列表条目应该仍然存在
	store.listsMu.RLock()
	_, exists := store.lists["list1"]
	store.listsMu.RUnlock()
	assert.True(t, exists, "清理被禁用时，空列表条目不应被移除")
}

func TestLocalStorage_DefaultConfigHasCleanupInterval(t *testing.T) {
	// 测试默认配置包含合理的 CleanupInterval 值
	cfg := DefaultConfig()
	assert.Equal(t, 5*time.Minute, cfg.CleanupInterval,
		"默认 CleanupInterval 应为 5 分钟")
}

func TestLocalStorage_CleanupExpiredListEntries(t *testing.T) {
	// 测试清理协程能移除已过期的列表条目
	cfg := DefaultConfig()
	cfg.CleanupInterval = 50 * time.Millisecond

	store, err := New(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 创建一个带过期时间的列表条目
	err = store.LPush(ctx, "expiring_list", 100, 200)
	require.NoError(t, err)

	// 手动设置列表条目的过期时间为过去（模拟已过期）
	store.listsMu.RLock()
	entry := store.lists["expiring_list"]
	store.listsMu.RUnlock()

	entry.mu.Lock()
	entry.expires = time.Now().Add(-1 * time.Second) // 已过期
	entry.mu.Unlock()

	// 等待清理协程运行
	time.Sleep(150 * time.Millisecond)

	// 验证过期列表已被移除
	store.listsMu.RLock()
	_, exists := store.lists["expiring_list"]
	store.listsMu.RUnlock()
	assert.False(t, exists, "过期的列表条目应该被清理协程移除")
}

func TestLocalStorage_CleanupConcurrentAccess(t *testing.T) {
	// 测试清理协程与并发读写操作的安全性
	cfg := DefaultConfig()
	cfg.CleanupInterval = 10 * time.Millisecond

	store, err := New(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	done := make(chan struct{})

	// 启动并发写入
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("list_%d", i%10)
			_ = store.LPush(ctx, key, int64(i))
			if i%3 == 0 {
				_ = store.LTrim(ctx, key, 1, 0) // 偶尔清空列表
			}
			time.Sleep(time.Millisecond)
		}
	}()

	<-done

	// 不应发生 panic 或数据竞争
	// 如果存在竞争条件，使用 -race 标志运行时会被检测到
}

// =============================================================================
// Helper Functions
// =============================================================================

func newTestStorage(t *testing.T) storage.Storage {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	return store
}
