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
	"time"

	"koala/internal/storage"
)

// Leak 实现漏桶算法。
// 它维护一个时间戳列表，并随着时间"泄漏"旧条目。
// 桶在时间窗口内最多可容纳Count个请求。
type Leak struct{}

// NewLeak 创建一个新的Leak算法实例。
func NewLeak() *Leak {
	return &Leak{}
}

// Browse 检查桶是否已满。
func (l *Leak) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
	// 首先，清理过期条目
	err := l.cleanExpired(ctx, key, limit, store)
	if err != nil {
		return false, err
	}

	// 检查桶大小
	length, err := store.LLen(ctx, key)
	if err != nil {
		return false, err
	}

	return length >= limit.Count, nil
}

// Update 向桶中添加新的时间戳。
func (l *Leak) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
	// 先清理过期条目
	err := l.cleanExpired(ctx, key, limit, store)
	if err != nil {
		return err
	}

	// 添加当前时间戳
	now := time.Now().UnixMilli()
	return store.LPush(ctx, key, now)
}

// cleanExpired 移除超出时间窗口的旧时间戳。
func (l *Leak) cleanExpired(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
	now := time.Now().UnixMilli()
	windowStart := now - limit.Time.Milliseconds()

	// 获取所有条目
	values, err := store.LRange(ctx, key, 0, -1)
	if err != nil {
		return err
	}

	if len(values) == 0 {
		return nil
	}

	// 找到窗口内第一个有效条目的索引
	validStart := -1
	for i, ts := range values {
		if ts >= windowStart {
			validStart = i
			break
		}
	}

	if validStart == -1 {
		// 所有条目都已过期，清空列表
		return store.LTrim(ctx, key, 1, 0) // 这会清空列表
	}

	if validStart > 0 {
		// 从前端修剪过期条目
		// 由于LPush将元素添加到前端，旧条目位于末尾
		// 我们需要保留从0到len-validStart-1的条目
		return store.LTrim(ctx, key, 0, int64(len(values)-validStart-1))
	}

	return nil
}

// Name 返回算法名称。
func (l *Leak) Name() string {
	return "leak"
}
