// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package redis provides a Redis-based storage implementation for Koala.
package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koala/internal/storage"
)

// Note: These tests require a running Redis instance.
// Skip tests if Redis is not available.

func skipIfNoRedis(t *testing.T, store *RedisStorage) {
	if err := store.Ping(context.Background()); err != nil {
		t.Skip("Redis not available, skipping test")
	}
}

// =============================================================================
// String Operations Tests
// =============================================================================

func TestRedisStorage_Get_NotFound(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	val, err := store.Get(context.Background(), "test:nonexistent:"+randomKey())
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
	assert.Equal(t, "", val)
}

func TestRedisStorage_Set_And_Get(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:key:" + randomKey()

	err := store.Set(ctx, key, "value1", 0)
	require.NoError(t, err)

	val, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// Cleanup
	store.Delete(ctx, key)
}

func TestRedisStorage_Set_WithTTL(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:ttl:" + randomKey()

	err := store.Set(ctx, key, "value1", 200*time.Millisecond)
	require.NoError(t, err)

	val, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	time.Sleep(250 * time.Millisecond)

	_, err = store.Get(ctx, key)
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestRedisStorage_Delete(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:delete:" + randomKey()

	err := store.Set(ctx, key, "value1", 0)
	require.NoError(t, err)

	err = store.Delete(ctx, key)
	assert.NoError(t, err)

	_, err = store.Get(ctx, key)
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

func TestRedisStorage_Exists(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:exists:" + randomKey()

	exists, err := store.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = store.Set(ctx, key, "value1", 0)
	require.NoError(t, err)

	exists, err = store.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	store.Delete(ctx, key)
}

func TestRedisStorage_Expire(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:expire:" + randomKey()

	err := store.Set(ctx, key, "value1", 0)
	require.NoError(t, err)

	err = store.Expire(ctx, key, 200*time.Millisecond)
	assert.NoError(t, err)

	val, err := store.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	time.Sleep(250 * time.Millisecond)

	_, err = store.Get(ctx, key)
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
}

// =============================================================================
// Counter Operations Tests
// =============================================================================

func TestRedisStorage_GetInt_NotFound(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	val, err := store.GetInt(context.Background(), "test:counter:"+randomKey())
	assert.ErrorIs(t, err, storage.ErrKeyNotFound)
	assert.Equal(t, int64(0), val)
}

func TestRedisStorage_SetInt_And_GetInt(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:setint:" + randomKey()

	err := store.SetInt(ctx, key, 100, 0)
	require.NoError(t, err)

	val, err := store.GetInt(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), val)

	store.Delete(ctx, key)
}

func TestRedisStorage_Incr(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:incr:" + randomKey()

	val, err := store.Incr(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = store.Incr(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)

	store.Delete(ctx, key)
}

func TestRedisStorage_IncrBy(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:incrby:" + randomKey()

	val, err := store.IncrBy(ctx, key, 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), val)

	val, err = store.IncrBy(ctx, key, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), val)

	store.Delete(ctx, key)
}

func TestRedisStorage_IncrWithTTL(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:incrttl:" + randomKey()

	val, err := store.IncrWithTTL(ctx, key, 200*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	val, err = store.IncrWithTTL(ctx, key, 200*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), val)

	time.Sleep(250 * time.Millisecond)

	val, err = store.IncrWithTTL(ctx, key, 200*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), val)

	store.Delete(ctx, key)
}

func TestRedisStorage_Incr_Concurrent(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:concurrent:" + randomKey()
	const goroutines = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := store.Incr(ctx, key)
				assert.NoError(t, err)
			}
		}()
	}

	wg.Wait()

	val, err := store.GetInt(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(goroutines*iterations), val)

	store.Delete(ctx, key)
}

// =============================================================================
// List Operations Tests
// =============================================================================

func TestRedisStorage_LPush(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:lpush:" + randomKey()

	err := store.LPush(ctx, key, 1, 2, 3)
	assert.NoError(t, err)

	length, err := store.LLen(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)

	store.Delete(ctx, key)
}

func TestRedisStorage_LIndex(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:lindex:" + randomKey()

	err := store.LPush(ctx, key, 10, 20, 30)
	require.NoError(t, err)

	// LPush adds to the front, so order is [30, 20, 10]
	val, err := store.LIndex(ctx, key, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(30), val)

	val, err = store.LIndex(ctx, key, -1)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	store.Delete(ctx, key)
}

func TestRedisStorage_LTrim(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:ltrim:" + randomKey()

	err := store.LPush(ctx, key, 1, 2, 3, 4, 5)
	require.NoError(t, err)

	err = store.LTrim(ctx, key, 0, 2)
	assert.NoError(t, err)

	length, err := store.LLen(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), length)

	store.Delete(ctx, key)
}

func TestRedisStorage_LRange(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()
	skipIfNoRedis(t, store)

	ctx := context.Background()
	key := "test:lrange:" + randomKey()

	err := store.LPush(ctx, key, 1, 2, 3, 4, 5)
	require.NoError(t, err)

	values, err := store.LRange(ctx, key, 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, []int64{5, 4, 3}, values)

	values, err = store.LRange(ctx, key, 0, -1)
	assert.NoError(t, err)
	assert.Equal(t, []int64{5, 4, 3, 2, 1}, values)

	store.Delete(ctx, key)
}

// =============================================================================
// Connection Tests
// =============================================================================

func TestRedisStorage_Ping(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()

	err := store.Ping(context.Background())
	if err != nil {
		t.Skip("Redis not available")
	}
	assert.NoError(t, err)
}

func TestRedisStorage_Type(t *testing.T) {
	store := newTestRedisStorage(t)
	defer store.Close()

	assert.Equal(t, "redis", store.Type())
}

// =============================================================================
// Helper Functions
// =============================================================================

func newTestRedisStorage(t *testing.T) *RedisStorage {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	return store
}

func randomKey() string {
	return time.Now().Format("20060102150405.000000000")
}
