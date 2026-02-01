// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package redis 提供 Koala 的 Redis 存储实现。
package redis

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"koala/internal/storage"
)

// Config 保存 RedisStorage 的配置。
type Config struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeyPrefix    string
}

// DefaultConfig 返回默认的 Redis 配置。
func DefaultConfig() Config {
	return Config{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		KeyPrefix:    "koala:",
	}
}

// RedisStorage 使用 Redis 实现 storage.Storage 接口。
type RedisStorage struct {
	client    *redis.Client
	keyPrefix string
	closed    atomic.Bool
}

// New 创建一个新的 RedisStorage 实例。
func New(cfg Config) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	return &RedisStorage{
		client:    client,
		keyPrefix: cfg.KeyPrefix,
	}, nil
}

func (s *RedisStorage) key(k string) string {
	return s.keyPrefix + k
}

// =============================================================================
// 字符串操作
// =============================================================================

func (s *RedisStorage) Get(ctx context.Context, key string) (string, error) {
	if s.closed.Load() {
		return "", storage.ErrStorageClosed
	}

	val, err := s.client.Get(ctx, s.key(key)).Result()
	if err == redis.Nil {
		return "", storage.ErrKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("redis get: %w", err)
	}
	return val, nil
}

func (s *RedisStorage) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	err := s.client.Set(ctx, s.key(key), value, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func (s *RedisStorage) Delete(ctx context.Context, key string) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	err := s.client.Del(ctx, s.key(key)).Err()
	if err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}

func (s *RedisStorage) Exists(ctx context.Context, key string) (bool, error) {
	if s.closed.Load() {
		return false, storage.ErrStorageClosed
	}

	n, err := s.client.Exists(ctx, s.key(key)).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists: %w", err)
	}
	return n > 0, nil
}

func (s *RedisStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	ok, err := s.client.Expire(ctx, s.key(key), ttl).Result()
	if err != nil {
		return fmt.Errorf("redis expire: %w", err)
	}
	if !ok {
		return storage.ErrKeyNotFound
	}
	return nil
}

// =============================================================================
// 计数器操作
// =============================================================================

func (s *RedisStorage) GetInt(ctx context.Context, key string) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	val, err := s.client.Get(ctx, s.key(key)).Int64()
	if err == redis.Nil {
		return 0, storage.ErrKeyNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("redis getint: %w", err)
	}
	return val, nil
}

func (s *RedisStorage) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	err := s.client.Set(ctx, s.key(key), value, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis setint: %w", err)
	}
	return nil
}

func (s *RedisStorage) Incr(ctx context.Context, key string) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	val, err := s.client.Incr(ctx, s.key(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr: %w", err)
	}
	return val, nil
}

func (s *RedisStorage) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	val, err := s.client.IncrBy(ctx, s.key(key), delta).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incrby: %w", err)
	}
	return val, nil
}

func (s *RedisStorage) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	k := s.key(key)

	// 使用 Lua 脚本实现原子性的自增和首次设置时的 TTL
	script := redis.NewScript(`
		local val = redis.call('INCR', KEYS[1])
		if val == 1 then
			redis.call('PEXPIRE', KEYS[1], ARGV[1])
		end
		return val
	`)

	val, err := script.Run(ctx, s.client, []string{k}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("redis incrwithttl: %w", err)
	}
	return val, nil
}

// =============================================================================
// 列表操作
// =============================================================================

func (s *RedisStorage) LPush(ctx context.Context, key string, values ...int64) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	// 将 int64 转换为 interface{}
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}

	err := s.client.LPush(ctx, s.key(key), args...).Err()
	if err != nil {
		return fmt.Errorf("redis lpush: %w", err)
	}
	return nil
}

func (s *RedisStorage) LLen(ctx context.Context, key string) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	val, err := s.client.LLen(ctx, s.key(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("redis llen: %w", err)
	}
	return val, nil
}

func (s *RedisStorage) LIndex(ctx context.Context, key string, index int64) (int64, error) {
	if s.closed.Load() {
		return 0, storage.ErrStorageClosed
	}

	val, err := s.client.LIndex(ctx, s.key(key), index).Result()
	if err == redis.Nil {
		return 0, storage.ErrIndexOutOfRange
	}
	if err != nil {
		return 0, fmt.Errorf("redis lindex: %w", err)
	}

	intVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("redis lindex parse: %w", err)
	}
	return intVal, nil
}

func (s *RedisStorage) LTrim(ctx context.Context, key string, start, stop int64) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	err := s.client.LTrim(ctx, s.key(key), start, stop).Err()
	if err != nil {
		return fmt.Errorf("redis ltrim: %w", err)
	}
	return nil
}

func (s *RedisStorage) LRange(ctx context.Context, key string, start, stop int64) ([]int64, error) {
	if s.closed.Load() {
		return nil, storage.ErrStorageClosed
	}

	vals, err := s.client.LRange(ctx, s.key(key), start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lrange: %w", err)
	}

	result := make([]int64, len(vals))
	for i, v := range vals {
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("redis lrange parse: %w", err)
		}
		result[i] = intVal
	}
	return result, nil
}

// =============================================================================
// 连接管理
// =============================================================================

func (s *RedisStorage) Ping(ctx context.Context) error {
	if s.closed.Load() {
		return storage.ErrStorageClosed
	}

	err := s.client.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

func (s *RedisStorage) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	return s.client.Close()
}

func (s *RedisStorage) Type() string {
	return "redis"
}

// Client 返回底层的 Redis 客户端，用于高级操作。
func (s *RedisStorage) Client() *redis.Client {
	return s.client
}

// 确保 RedisStorage 实现了 storage.Storage 接口
var _ storage.Storage = (*RedisStorage)(nil)
