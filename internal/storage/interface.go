// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package storage 定义 Koala 限流器的存储接口。
package storage

import (
	"context"
	"errors"
	"time"
)

// 通用错误
var (
	ErrKeyNotFound     = errors.New("key not found")
	ErrIndexOutOfRange = errors.New("index out of range")
	ErrTypeMismatch    = errors.New("type mismatch")
	ErrStorageClosed   = errors.New("storage is closed")
)

// Storage 定义所有存储后端的接口。
// 实现必须是并发安全的。
type Storage interface {
	// 字符串操作
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// 计数器操作（必须是原子操作）
	GetInt(ctx context.Context, key string) (int64, error)
	SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error)

	// 列表操作（用于漏桶算法）
	LPush(ctx context.Context, key string, values ...int64) error
	LLen(ctx context.Context, key string) (int64, error)
	LIndex(ctx context.Context, key string, index int64) (int64, error)
	LTrim(ctx context.Context, key string, start, stop int64) error
	LRange(ctx context.Context, key string, start, stop int64) ([]int64, error)

	// 连接管理
	Ping(ctx context.Context) error
	Close() error

	// Type 返回存储类型名称（"local" 或 "redis"）
	Type() string
}
