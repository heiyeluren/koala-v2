// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package algorithm 提供Koala限流算法实现。
package algorithm

import (
	"context"
	"time"

	"koala/internal/storage"
)

// Algorithm 定义限流算法的接口。
type Algorithm interface {
	// Browse 检查是否达到限流阈值。
	// 返回 true 表示达到限制（应拒绝），返回 false 表示未达到限制（应允许）。
	Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (hit bool, err error)

	// Update 递增指定键的计数器。
	Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error

	// Name 返回算法名称。
	Name() string
}

// LimitConfig 保存限流配置参数。
type LimitConfig struct {
	Time  time.Duration // 时间窗口
	Count int64         // 窗口内的计数限制
	Base  int64         // 基础阈值（用于Base算法）
}

// AlgorithmType 表示算法类型。
type AlgorithmType string

const (
	TypeDirect AlgorithmType = "direct"
	TypeCount  AlgorithmType = "count"
	TypeBase   AlgorithmType = "base"
	TypeLeak   AlgorithmType = "leak"
)

// New 根据类型创建算法实例。
func New(typ AlgorithmType) Algorithm {
	switch typ {
	case TypeDirect:
		return NewDirect()
	case TypeCount:
		return NewCount()
	case TypeBase:
		return NewBase()
	case TypeLeak:
		return NewLeak()
	default:
		return NewCount()
	}
}
