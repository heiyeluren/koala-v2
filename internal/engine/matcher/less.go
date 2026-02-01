// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
package matcher

import "strconv"

// LessMatcher 当数值小于阈值时匹配。
// 模式格式："<10"表示当value < 10时匹配。
type LessMatcher struct{}

// Match 如果值小于模式中的阈值则返回true。
func (m *LessMatcher) Match(pattern string, value string) bool {
	// 从模式中解析阈值（移除"<"前缀）
	if len(pattern) < 2 {
		return false
	}

	thresholdStr := pattern[1:]
	threshold, err := strconv.ParseInt(thresholdStr, 10, 64)
	if err != nil {
		return false
	}

	// 解析值
	val, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false
	}

	return val < threshold
}

// Type 返回匹配器的类型名称。
func (m *LessMatcher) Type() string {
	return "less"
}
