// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
package matcher

import "strings"

// MultiMatcher 当值是模式中逗号分隔值之一时匹配。
// 模式格式："a,b,c"表示当值为"a"、"b"或"c"时匹配。
type MultiMatcher struct{}

// Match 如果值是模式中逗号分隔值之一则返回true。
func (m *MultiMatcher) Match(pattern string, value string) bool {
	parts := strings.Split(pattern, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == value {
			return true
		}
	}
	return false
}

// Type 返回匹配器的类型名称。
func (m *MultiMatcher) Type() string {
	return "multi"
}
