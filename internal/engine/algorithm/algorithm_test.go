// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package algorithm provides rate limiting algorithms for Koala.
package algorithm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koala/internal/storage"
	"koala/internal/storage/local"
)

// =============================================================================
// Direct Algorithm Tests
// =============================================================================

func TestDirect_Browse_AlwaysHit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewDirect()
	ctx := context.Background()

	// Direct algorithm always returns hit=true (meaning the rule matched)
	hit, err := algo.Browse(ctx, "uid:123", LimitConfig{}, store)
	assert.NoError(t, err)
	assert.True(t, hit)
}

func TestDirect_Update_NoOp(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewDirect()
	ctx := context.Background()

	// Update should be a no-op for Direct algorithm
	err := algo.Update(ctx, "uid:123", LimitConfig{}, store)
	assert.NoError(t, err)
}

func TestDirect_Name(t *testing.T) {
	algo := NewDirect()
	assert.Equal(t, "direct", algo.Name())
}

// =============================================================================
// Count Algorithm Tests
// =============================================================================

func TestCount_UnderLimit_NotHit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewCount()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 5,
		Time:  time.Minute,
	}

	// First request should not hit limit
	hit, err := algo.Browse(ctx, "count:test:1", cfg, store)
	assert.NoError(t, err)
	assert.False(t, hit)
}

func TestCount_AtLimit_Hit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewCount()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 3,
		Time:  time.Minute,
	}
	key := "count:test:2"

	// Simulate 3 updates (hit limit)
	for i := 0; i < 3; i++ {
		err := algo.Update(ctx, key, cfg, store)
		require.NoError(t, err)
	}

	// Now browse should hit limit
	hit, err := algo.Browse(ctx, key, cfg, store)
	assert.NoError(t, err)
	assert.True(t, hit)
}

func TestCount_Update_Increments(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewCount()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 10,
		Time:  time.Minute,
	}
	key := "count:test:3"

	// Update 5 times
	for i := 0; i < 5; i++ {
		err := algo.Update(ctx, key, cfg, store)
		require.NoError(t, err)
	}

	// Check counter value
	count, err := store.GetInt(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestCount_WindowReset(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewCount()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 2,
		Time:  100 * time.Millisecond,
	}
	key := "count:test:4"

	// Update to limit
	algo.Update(ctx, key, cfg, store)
	algo.Update(ctx, key, cfg, store)

	// Should hit limit
	hit, _ := algo.Browse(ctx, key, cfg, store)
	assert.True(t, hit)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should not hit limit (window reset)
	hit, _ = algo.Browse(ctx, key, cfg, store)
	assert.False(t, hit)
}

func TestCount_Name(t *testing.T) {
	algo := NewCount()
	assert.Equal(t, "count", algo.Name())
}

// =============================================================================
// Base Algorithm Tests
// =============================================================================

func TestBase_UnderBase_NotHit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewBase()
	ctx := context.Background()

	cfg := LimitConfig{
		Base:  5,
		Count: 1,
		Time:  time.Minute,
	}
	key := "base:test:1"

	// First 5 requests (under base) should not hit
	for i := 0; i < 5; i++ {
		hit, err := algo.Browse(ctx, key, cfg, store)
		assert.NoError(t, err)
		assert.False(t, hit)
		algo.Update(ctx, key, cfg, store)
	}
}

func TestBase_OverBase_SecondaryLimit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewBase()
	ctx := context.Background()

	cfg := LimitConfig{
		Base:  3,
		Count: 1,
		Time:  time.Second,
	}
	key := "base:test:2"

	// Update 3 times (reach base)
	for i := 0; i < 3; i++ {
		algo.Update(ctx, key, cfg, store)
	}

	// First browse after base should not hit (1 allowed per second)
	hit, err := algo.Browse(ctx, key, cfg, store)
	assert.NoError(t, err)
	assert.False(t, hit)

	// Update for secondary counter
	algo.Update(ctx, key, cfg, store)

	// Second browse should hit (exceeded 1/sec)
	hit, err = algo.Browse(ctx, key, cfg, store)
	assert.NoError(t, err)
	assert.True(t, hit)
}

func TestBase_Name(t *testing.T) {
	algo := NewBase()
	assert.Equal(t, "base", algo.Name())
}

// =============================================================================
// Leak Algorithm Tests
// =============================================================================

func TestLeak_UnderLimit_NotHit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewLeak()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 5,
		Time:  time.Minute,
	}
	key := "leak:test:1"

	// First request should not hit limit
	hit, err := algo.Browse(ctx, key, cfg, store)
	assert.NoError(t, err)
	assert.False(t, hit)
}

func TestLeak_BucketFull_Hit(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewLeak()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 3,
		Time:  3 * time.Second,
	}
	key := "leak:test:2"

	// Fill the bucket
	for i := 0; i < 3; i++ {
		algo.Update(ctx, key, cfg, store)
	}

	// Bucket full, should hit
	hit, err := algo.Browse(ctx, key, cfg, store)
	assert.NoError(t, err)
	assert.True(t, hit)
}

func TestLeak_Drain(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	algo := NewLeak()
	ctx := context.Background()

	cfg := LimitConfig{
		Count: 2,
		Time:  200 * time.Millisecond,
	}
	key := "leak:test:3"

	// Fill bucket
	algo.Update(ctx, key, cfg, store)
	algo.Update(ctx, key, cfg, store)

	// Should hit
	hit, _ := algo.Browse(ctx, key, cfg, store)
	assert.True(t, hit)

	// Wait for leak (bucket drains)
	time.Sleep(250 * time.Millisecond)

	// Should not hit (bucket drained)
	hit, _ = algo.Browse(ctx, key, cfg, store)
	assert.False(t, hit)
}

func TestLeak_Name(t *testing.T) {
	algo := NewLeak()
	assert.Equal(t, "leak", algo.Name())
}

// =============================================================================
// Algorithm Factory Tests
// =============================================================================

func TestNew_Direct(t *testing.T) {
	algo := New(TypeDirect)
	assert.Equal(t, "direct", algo.Name())
}

func TestNew_Count(t *testing.T) {
	algo := New(TypeCount)
	assert.Equal(t, "count", algo.Name())
}

func TestNew_Base(t *testing.T) {
	algo := New(TypeBase)
	assert.Equal(t, "base", algo.Name())
}

func TestNew_Leak(t *testing.T) {
	algo := New(TypeLeak)
	assert.Equal(t, "leak", algo.Name())
}

func TestNew_Default(t *testing.T) {
	algo := New("unknown")
	assert.Equal(t, "count", algo.Name())
}

// =============================================================================
// Helper Functions
// =============================================================================

func newTestStore(t *testing.T) storage.Storage {
	store, err := local.New(local.DefaultConfig())
	require.NoError(t, err)
	return store
}
