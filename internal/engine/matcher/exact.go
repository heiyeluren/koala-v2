// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
package matcher

// ExactMatcher 当值与模式完全相等时匹配。
type ExactMatcher struct{}

// Match 如果值与模式完全相等则返回true。
func (m *ExactMatcher) Match(pattern string, value string) bool {
	return pattern == value
}

// Type 返回匹配器的类型名称。
func (m *ExactMatcher) Type() string {
	return "exact"
}
