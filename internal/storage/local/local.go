// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package local 提供 Koala 的内存存储实现。
// 使用 Ristretto 作为字符串缓存，使用自定义结构存储计数器和列表。
package local

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"

	"koala/internal/storage"
)

// Config 保存 LocalStorage 的配置。
type Config struct {
	// MaxCost 是缓存的最大容量（以字节为单位）
	MaxCost int64
	// NumCounters 是 4 位访问计数器的数量
	NumCounters int64
	// BufferItems 是每个 Get 缓冲区的键数量
	BufferItems int64
	// CleanupInterval 是后台清理过期列表条目的时间间隔。
	// 设置为 0 可禁用后台清理。
	CleanupInterval time.Duration
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		MaxCost:         1 << 30, // 1GB
		NumCounters:     1e7,     // 1000万
		BufferItems:     64,
		CleanupInterval: 5 * time.Minute, // 每5分钟清理一次过期列表
	}
}

// LocalStorage 使用内存结构实现 storage.Storage 接口。
type LocalStorage struct {
	// 使用 Ristretto 的字符串缓存
	cache *ristretto.Cache[string, string]

	// 支持原子操作的计数器存储
	counters   map[string]*counterEntry
	countersMu sync.RWMutex

	// 用于漏桶算法的列表存储
	lists   map[string]*listEntry
	listsMu sync.RWMutex

	// TTL 管理
	ttls   map[string]time.Time
	ttlsMu sync.RWMutex

	// 用于停止后台清理协程的通道
	stopCh chan struct{}

	closed atomic.Bool
}

type counterEntry struct {
	value   atomic.Int64
	expires time.Time
}

type listEntry struct {
	data    []int64
	expires time.Time
	mu      sync.RWMutex
}

// New 创建一个新的 LocalStorage 实例。
func New(cfg Config) (*LocalStorage, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, string]{
		MaxCost:     cfg.MaxCost,
		NumCounters: cfg.NumCounters,
		BufferItems: cfg.BufferItems,
	})
	if err != nil {
		return nil, err
	}

	s := &LocalStorage{
		cache:    cache,
		counters: make(map[string]*counterEntry),
		lists:    make(map[string]*listEntry),
		ttls:     make(map[string]time.Time),
		stopCh:   make(chan struct{}),
	}

	// 如果配置了清理间隔，启动后台清理协程
	if cfg.CleanupInterval > 0 {
		go s.cleanupLoop(cfg.CleanupInterval)
	}

	return s, nil
}

// =============================================================================
// 字符串操作
// =============================================================================

func (s *LocalStorage) Get(ctx context.Context, key string) (string, error) {
	if s.closed.Load() {
		return "", storage.ErrStorageClosed
	}

	// 先检查 TTL
	if s.isExpired(key) {
		s.cache.Del(key)
		return "", storage.ErrKeyNotFound
	}

	val, found := s.cache.Get(key)
	if !found {
		return "", storage.ErrKeyNotFound
	}
	return val, nil
}

func (s *LocalStorage) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	cost := int64(len(value))
	if ttl > 0 {
		s.cache.SetWithTTL(key, value, cost, ttl)
		s.setTTL(key, ttl)
	} else {
		s.cache.Set(key, value, cost)
		s.clearTTL(key)
	}

	// 等待值被存储
	s.cache.Wait()
	return nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	s.cache.Del(key)
	s.clearTTL(key)
	return nil
}

func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	if s.closed.Load() {
		return false, storage.ErrStorageClosed
	}

	if s.isExpired(key) {
		return false, nil
	}

	_, found := s.cache.Get(key)
	return found, nil
}

func (s *LocalStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	// 检查键是否存在
	val, found := s.cache.Get(key)
	if !found {
		return storage.ErrKeyNotFound
	}

	// 使用 TTL 重新设置
	cost := int64(len(val))
	s.cache.SetWithTTL(key, val, cost, ttl)
	s.setTTL(key, ttl)
	s.cache.Wait()
	return nil
}

// =============================================================================
// 计数器操作
// =============================================================================

func (s *LocalStorage) GetInt(ctx context.Context, key string) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	s.countersMu.RLock()
	entry, exists := s.counters[key]
	s.countersMu.RUnlock()

	if !exists {
		return 0, storage.ErrKeyNotFound
	}

	// 检查是否过期
	if !entry.expires.IsZero() && time.Now().After(entry.expires) {
		s.countersMu.Lock()
		delete(s.counters, key)
		s.countersMu.Unlock()
		return 0, storage.ErrKeyNotFound
	}

	return entry.value.Load(), nil
}

func (s *LocalStorage) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	entry := &counterEntry{}
	entry.value.Store(value)
	if ttl > 0 {
		entry.expires = time.Now().Add(ttl)
	}

	s.countersMu.Lock()
	s.counters[key] = entry
	s.countersMu.Unlock()
	return nil
}

func (s *LocalStorage) Incr(ctx context.Context, key string) (int64, error) {
	return s.IncrBy(ctx, key, 1)
}

func (s *LocalStorage) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	s.countersMu.Lock()
	entry, exists := s.counters[key]

	// 检查是否过期
	if exists && !entry.expires.IsZero() && time.Now().After(entry.expires) {
		delete(s.counters, key)
		exists = false
	}

	if !exists {
		entry = &counterEntry{}
		s.counters[key] = entry
	}
	s.countersMu.Unlock()

	newVal := entry.value.Add(delta)
	return newVal, nil
}

func (s *LocalStorage) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	s.countersMu.Lock()
	entry, exists := s.counters[key]

	// 检查是否过期
	if exists && !entry.expires.IsZero() && time.Now().After(entry.expires) {
		delete(s.counters, key)
		exists = false
	}

	if !exists {
		entry = &counterEntry{}
		if ttl > 0 {
			entry.expires = time.Now().Add(ttl)
		}
		s.counters[key] = entry
	}
	s.countersMu.Unlock()

	newVal := entry.value.Add(1)
	return newVal, nil
}

// =============================================================================
// 列表操作
// =============================================================================

func (s *LocalStorage) LPush(ctx context.Context, key string, values ...int64) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	s.listsMu.Lock()
	entry, exists := s.lists[key]
	if !exists {
		entry = &listEntry{data: make([]int64, 0)}
		s.lists[key] = entry
	}
	s.listsMu.Unlock()

	entry.mu.Lock()
	// 在前面添加值（LPush 添加到列表头部）
	newData := make([]int64, len(values)+len(entry.data))
	// 按 LPush 语义反转顺序
	for i, v := range values {
		newData[len(values)-1-i] = v
	}
	copy(newData[len(values):], entry.data)
	entry.data = newData
	entry.mu.Unlock()

	return nil
}

func (s *LocalStorage) LLen(ctx context.Context, key string) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	s.listsMu.RLock()
	entry, exists := s.lists[key]
	s.listsMu.RUnlock()

	if !exists {
		return 0, nil
	}

	entry.mu.RLock()
	length := int64(len(entry.data))
	entry.mu.RUnlock()

	return length, nil
}

func (s *LocalStorage) LIndex(ctx context.Context, key string, index int64) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	s.listsMu.RLock()
	entry, exists := s.lists[key]
	s.listsMu.RUnlock()

	if !exists {
		return 0, storage.ErrIndexOutOfRange
	}

	entry.mu.RLock()
	defer entry.mu.RUnlock()

	length := int64(len(entry.data))
	if length == 0 {
		return 0, storage.ErrIndexOutOfRange
	}

	// 处理负索引
	if index < 0 {
		index = length + index
	}

	if index < 0 || index >= length {
		return 0, storage.ErrIndexOutOfRange
	}

	return entry.data[index], nil
}

func (s *LocalStorage) LTrim(ctx context.Context, key string, start, stop int64) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	s.listsMu.RLock()
	entry, exists := s.lists[key]
	s.listsMu.RUnlock()

	if !exists {
		return nil
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	length := int64(len(entry.data))
	if length == 0 {
		return nil
	}

	// 处理负索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界值处理
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}

	if start > stop || start >= length {
		entry.data = []int64{}
		return nil
	}

	entry.data = entry.data[start : stop+1]
	return nil
}

func (s *LocalStorage) LRange(ctx context.Context, key string, start, stop int64) ([]int64, error) {
	if s.closed.Load() {
		return nil, storage.ErrStorageClosed
	}

	s.listsMu.RLock()
	entry, exists := s.lists[key]
	s.listsMu.RUnlock()

	if !exists {
		return []int64{}, nil
	}

	entry.mu.RLock()
	defer entry.mu.RUnlock()

	length := int64(len(entry.data))
	if length == 0 {
		return []int64{}, nil
	}

	// 处理负索引
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// 边界值处理
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}

	if start > stop || start >= length {
		return []int64{}, nil
	}

	result := make([]int64, stop-start+1)
	copy(result, entry.data[start:stop+1])
	return result, nil
}

// =============================================================================
// 连接管理
// =============================================================================

func (s *LocalStorage) Ping(ctx context.Context) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}
	return nil
}

func (s *LocalStorage) Close() error {
	if s.closed.Swap(true) {
		return nil // 已经关闭
	}
	// 通知清理协程停止
	close(s.stopCh)
	s.cache.Close()
	return nil
}

func (s *LocalStorage) Type() string {
	return "local"
}

// =============================================================================
// 辅助方法
// =============================================================================

func (s *LocalStorage) isExpired(key string) bool {
	s.ttlsMu.RLock()
	expires, exists := s.ttls[key]
	s.ttlsMu.RUnlock()

	if !exists {
		return false
	}
	return time.Now().After(expires)
}

func (s *LocalStorage) setTTL(key string, ttl time.Duration) {
	s.ttlsMu.Lock()
	s.ttls[key] = time.Now().Add(ttl)
	s.ttlsMu.Unlock()
}

func (s *LocalStorage) clearTTL(key string) {
	s.ttlsMu.Lock()
	delete(s.ttls, key)
	s.ttlsMu.Unlock()
}

// cleanupLoop 是后台清理协程的主循环。
// 定期扫描 lists map，移除空列表和已过期的列表条目。
func (s *LocalStorage) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupLists()
		}
	}
}

// cleanupLists 扫描并清理过期或空的列表条目。
// 采用保守策略：只移除数据为空的列表或已明确过期的列表条目。
func (s *LocalStorage) cleanupLists() {
	now := time.Now()

	// 收集需要删除的键
	var keysToDelete []string

	s.listsMu.RLock()
	for key, entry := range s.lists {
		entry.mu.RLock()
		isEmpty := len(entry.data) == 0
		isExpired := !entry.expires.IsZero() && now.After(entry.expires)
		entry.mu.RUnlock()

		if isEmpty || isExpired {
			keysToDelete = append(keysToDelete, key)
		}
	}
	s.listsMu.RUnlock()

	// 批量删除过期/空的条目
	if len(keysToDelete) > 0 {
		s.listsMu.Lock()
		for _, key := range keysToDelete {
			entry, exists := s.lists[key]
			if !exists {
				continue
			}
			// 二次检查：避免在读锁释放到写锁获取之间有新数据写入
			entry.mu.RLock()
			stillEmpty := len(entry.data) == 0
			stillExpired := !entry.expires.IsZero() && now.After(entry.expires)
			entry.mu.RUnlock()

			if stillEmpty || stillExpired {
				delete(s.lists, key)
			}
		}
		s.listsMu.Unlock()
	}
}

// 确保 LocalStorage 实现了 storage.Storage 接口
var _ storage.Storage = (*LocalStorage)(nil)
