// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
package matcher

// AnyMatcher 匹配任何非空值。
// 模式应为"+"表示任意匹配。
type AnyMatcher struct{}

// Match 如果值不为空则返回true。
func (m *AnyMatcher) Match(pattern string, value string) bool {
	return len(value) > 0
}

// Type 返回匹配器的类型名称。
func (m *AnyMatcher) Type() string {
	return "any"
}
