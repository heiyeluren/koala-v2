// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package manager 提供带自动故障转移功能的存储管理。
package manager

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"koala/internal/storage"
)

// Options 保存 StorageManager 的配置。
type Options struct {
	Primary             storage.Storage
	Fallback            storage.Storage
	MaxFailures         int
	HealthCheckInterval time.Duration
	RecoveryInterval    time.Duration
}

// DefaultOptions 返回默认的管理器选项。
func DefaultOptions() Options {
	return Options{
		MaxFailures:         3,
		HealthCheckInterval: 5 * time.Second,
		RecoveryInterval:    10 * time.Second,
	}
}

// StorageManager 管理存储并提供自动故障转移功能。
type StorageManager struct {
	primary  storage.Storage
	fallback storage.Storage

	current atomic.Pointer[storage.Storage]

	failures     int32
	maxFailures  int32
	degraded     atomic.Bool
	closed       atomic.Bool

	healthCheckInterval time.Duration
	recoveryInterval    time.Duration

	mu     sync.RWMutex
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New 创建一个新的 StorageManager，使用 fallback 作为主存储。
// 如果 primary 为 nil，则 fallback 同时作为主存储和备用存储。
func New(fallback storage.Storage, primary storage.Storage) *StorageManager {
	opts := DefaultOptions()
	opts.Primary = primary
	opts.Fallback = fallback
	return NewWithOptions(opts)
}

// NewWithOptions 使用自定义选项创建一个新的 StorageManager。
func NewWithOptions(opts Options) *StorageManager {
	if opts.MaxFailures <= 0 {
		opts.MaxFailures = 3
	}
	if opts.HealthCheckInterval <= 0 {
		opts.HealthCheckInterval = 5 * time.Second
	}
	if opts.RecoveryInterval <= 0 {
		opts.RecoveryInterval = 10 * time.Second
	}

	m := &StorageManager{
		primary:             opts.Primary,
		fallback:            opts.Fallback,
		maxFailures:         int32(opts.MaxFailures),
		healthCheckInterval: opts.HealthCheckInterval,
		recoveryInterval:    opts.RecoveryInterval,
		stopCh:              make(chan struct{}),
	}

	// 设置初始当前存储
	if opts.Primary != nil {
		m.current.Store(&opts.Primary)
	} else {
		m.current.Store(&opts.Fallback)
	}

	// 如果有主存储则启动健康检查
	if opts.Primary != nil {
		m.wg.Add(1)
		go m.healthCheckLoop()
	}

	return m
}

func (m *StorageManager) healthCheckLoop() {
	defer m.wg.Done()

	// 立即执行初始健康检查
	m.checkHealth()

	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkHealth()
		}
	}
}

func (m *StorageManager) checkHealth() {
	if m.primary == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.primary.Ping(ctx)
	if err != nil {
		m.recordFailure()
	} else {
		m.recordSuccess()
	}
}

func (m *StorageManager) recordFailure() {
	failures := atomic.AddInt32(&m.failures, 1)
	if failures >= m.maxFailures && !m.degraded.Load() {
		m.degrade()
	}
}

func (m *StorageManager) recordSuccess() {
	atomic.StoreInt32(&m.failures, 0)
	if m.degraded.Load() {
		m.recover()
	}
}

func (m *StorageManager) degrade() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.degraded.Load() {
		return
	}

	m.degraded.Store(true)
	m.current.Store(&m.fallback)
}

func (m *StorageManager) recover() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.degraded.Load() || m.primary == nil {
		return
	}

	m.degraded.Store(false)
	m.current.Store(&m.primary)
}

func (m *StorageManager) getCurrent() storage.Storage {
	return *m.current.Load()
}

// CurrentType 返回当前存储的类型。
func (m *StorageManager) CurrentType() string {
	return m.getCurrent().Type()
}

// IsDegraded 返回管理器是否已降级到备用存储。
func (m *StorageManager) IsDegraded() bool {
	return m.degraded.Load()
}

// =============================================================================
// 故障转移辅助方法
// =============================================================================

// executeWithFailover 对只返回 error 的操作执行故障转移。
// 先尝试当前存储，失败时记录故障并在降级后重试 fallback。
func (m *StorageManager) executeWithFailover(fn func(s storage.Storage) error) error {
	err := fn(m.getCurrent())
	if err != nil && m.primary != nil && !m.degraded.Load() {
		m.recordFailure()
		if m.degraded.Load() {
			return fn(m.fallback)
		}
	}
	return err
}

// executeWithFailoverString 对返回 (string, error) 的操作执行故障转移。
func (m *StorageManager) executeWithFailoverString(fn func(s storage.Storage) (string, error)) (string, error) {
	val, err := fn(m.getCurrent())
	if err != nil && m.primary != nil && !m.degraded.Load() {
		m.recordFailure()
		if m.degraded.Load() {
			return fn(m.fallback)
		}
	}
	return val, err
}

// executeWithFailoverInt64 对返回 (int64, error) 的操作执行故障转移。
func (m *StorageManager) executeWithFailoverInt64(fn func(s storage.Storage) (int64, error)) (int64, error) {
	val, err := fn(m.getCurrent())
	if err != nil && m.primary != nil && !m.degraded.Load() {
		m.recordFailure()
		if m.degraded.Load() {
			return fn(m.fallback)
		}
	}
	return val, err
}

// executeWithFailoverBool 对返回 (bool, error) 的操作执行故障转移。
func (m *StorageManager) executeWithFailoverBool(fn func(s storage.Storage) (bool, error)) (bool, error) {
	val, err := fn(m.getCurrent())
	if err != nil && m.primary != nil && !m.degraded.Load() {
		m.recordFailure()
		if m.degraded.Load() {
			return fn(m.fallback)
		}
	}
	return val, err
}

// executeWithFailoverSlice 对返回 ([]int64, error) 的操作执行故障转移。
func (m *StorageManager) executeWithFailoverSlice(fn func(s storage.Storage) ([]int64, error)) ([]int64, error) {
	val, err := fn(m.getCurrent())
	if err != nil && m.primary != nil && !m.degraded.Load() {
		m.recordFailure()
		if m.degraded.Load() {
			return fn(m.fallback)
		}
	}
	return val, err
}

// =============================================================================
// Storage 接口实现
// =============================================================================

func (m *StorageManager) Get(ctx context.Context, key string) (string, error) {
	return m.executeWithFailoverString(func(s storage.Storage) (string, error) {
		return s.Get(ctx, key)
	})
}

func (m *StorageManager) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return m.executeWithFailover(func(s storage.Storage) error {
		return s.Set(ctx, key, value, ttl)
	})
}

func (m *StorageManager) Delete(ctx context.Context, key string) error {
	return m.executeWithFailover(func(s storage.Storage) error {
		return s.Delete(ctx, key)
	})
}

func (m *StorageManager) Exists(ctx context.Context, key string) (bool, error) {
	return m.executeWithFailoverBool(func(s storage.Storage) (bool, error) {
		return s.Exists(ctx, key)
	})
}

func (m *StorageManager) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return m.executeWithFailover(func(s storage.Storage) error {
		return s.Expire(ctx, key, ttl)
	})
}

func (m *StorageManager) GetInt(ctx context.Context, key string) (int64, error) {
	return m.executeWithFailoverInt64(func(s storage.Storage) (int64, error) {
		return s.GetInt(ctx, key)
	})
}

func (m *StorageManager) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	return m.executeWithFailover(func(s storage.Storage) error {
		return s.SetInt(ctx, key, value, ttl)
	})
}

func (m *StorageManager) Incr(ctx context.Context, key string) (int64, error) {
	return m.executeWithFailoverInt64(func(s storage.Storage) (int64, error) {
		return s.Incr(ctx, key)
	})
}

func (m *StorageManager) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return m.executeWithFailoverInt64(func(s storage.Storage) (int64, error) {
		return s.IncrBy(ctx, key, delta)
	})
}

func (m *StorageManager) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return m.executeWithFailoverInt64(func(s storage.Storage) (int64, error) {
		return s.IncrWithTTL(ctx, key, ttl)
	})
}

func (m *StorageManager) LPush(ctx context.Context, key string, values ...int64) error {
	return m.executeWithFailover(func(s storage.Storage) error {
		return s.LPush(ctx, key, values...)
	})
}

func (m *StorageManager) LLen(ctx context.Context, key string) (int64, error) {
	return m.executeWithFailoverInt64(func(s storage.Storage) (int64, error) {
		return s.LLen(ctx, key)
	})
}

func (m *StorageManager) LIndex(ctx context.Context, key string, index int64) (int64, error) {
	return m.executeWithFailoverInt64(func(s storage.Storage) (int64, error) {
		return s.LIndex(ctx, key, index)
	})
}

func (m *StorageManager) LTrim(ctx context.Context, key string, start, stop int64) error {
	return m.executeWithFailover(func(s storage.Storage) error {
		return s.LTrim(ctx, key, start, stop)
	})
}

func (m *StorageManager) LRange(ctx context.Context, key string, start, stop int64) ([]int64, error) {
	return m.executeWithFailoverSlice(func(s storage.Storage) ([]int64, error) {
		return s.LRange(ctx, key, start, stop)
	})
}

func (m *StorageManager) Ping(ctx context.Context) error {
	return m.getCurrent().Ping(ctx)
}

func (m *StorageManager) Close() error {
	if m.closed.Swap(true) {
		return nil
	}

	close(m.stopCh)
	m.wg.Wait()

	// 不关闭底层存储 - 由调用者负责管理
	return nil
}

func (m *StorageManager) Type() string {
	return m.getCurrent().Type()
}

// 确保 StorageManager 实现了 storage.Storage 接口
var _ storage.Storage = (*StorageManager)(nil)
