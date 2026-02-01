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
// Helper Functions
// =============================================================================

func newTestStorage(t *testing.T) storage.Storage {
	store, err := New(DefaultConfig())
	require.NoError(t, err)
	return store
}
