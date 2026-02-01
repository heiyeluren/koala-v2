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
