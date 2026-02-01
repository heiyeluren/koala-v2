// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package algorithm

import (
	"context"

	"koala/internal/storage"
)

// Base 实现基于阈值的算法。
// 在达到基础阈值之前允许无限制访问，
// 之后在时间窗口内应用二级限流（Count/Time）。
type Base struct{}

// NewBase 创建一个新的Base算法实例。
func NewBase() *Base {
	return &Base{}
}

// Browse 检查是否达到限制。
// 未达到/刚好达到基础阈值时：始终返回 false（未命中）。
// 超过基础阈值时：检查二级限制。
func (b *Base) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
	// 获取总计数
	totalKey := key + ":total"
	total, err := store.GetInt(ctx, totalKey)
	if err != nil && err != storage.ErrKeyNotFound {
		return false, err
	}

	// 未达到或刚好达到基础阈值，未命中
	if total <= limit.Base {
		return false, nil
	}

	// 超过基础阈值，检查二级限制
	secondaryKey := key + ":secondary"
	count, err := store.GetInt(ctx, secondaryKey)
	if err != nil && err != storage.ErrKeyNotFound {
		return false, err
	}

	return count >= limit.Count, nil
}

// Update 递增总计数器和二级计数器。
func (b *Base) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
	// 递增总计数（无TTL）
	totalKey := key + ":total"
	total, err := store.Incr(ctx, totalKey)
	if err != nil {
		return err
	}

	// 如果超过基础阈值（严格大于），同时递增二级计数器
	if total > limit.Base {
		secondaryKey := key + ":secondary"
		_, err = store.IncrWithTTL(ctx, secondaryKey, limit.Time)
		if err != nil {
			return err
		}
	}

	return nil
}

// Name 返回算法名称。
func (b *Base) Name() string {
	return "base"
}
