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

// Direct 实现直接判断算法（白名单/黑名单）。
// 它始终返回 hit=true，表示规则已匹配。
// 实际的允许/拒绝决策由规则配置决定。
type Direct struct{}

// NewDirect 创建一个新的Direct算法实例。
func NewDirect() *Direct {
	return &Direct{}
}

// Browse 对于Direct算法始终返回 hit=true。
func (d *Direct) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
	return true, nil
}

// Update 对于Direct算法是空操作。
func (d *Direct) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
	return nil
}

// Name 返回算法名称。
func (d *Direct) Name() string {
	return "direct"
}
