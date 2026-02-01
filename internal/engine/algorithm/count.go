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

// Count 实现时间窗口计数器算法。
// 它追踪滑动时间窗口内的请求数量。
type Count struct{}

// NewCount 创建一个新的Count算法实例。
func NewCount() *Count {
	return &Count{}
}

// Browse 检查计数器是否已达到限制。
func (c *Count) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
	count, err := store.GetInt(ctx, key)
	if err != nil {
		if err == storage.ErrKeyNotFound {
			return false, nil // 尚无计数，未达到限制
		}
		return false, err
	}

	return count >= limit.Count, nil
}

// Update 递增计数器并设置TTL过期时间。
func (c *Count) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
	_, err := store.IncrWithTTL(ctx, key, limit.Time)
	return err
}

// Name 返回算法名称。
func (c *Count) Name() string {
	return "count"
}
