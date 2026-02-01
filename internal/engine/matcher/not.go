// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
package matcher

// NotMatcher 当值不等于模式时匹配。
// 模式格式："!value"表示当value不等于"value"时匹配。
type NotMatcher struct{}

// Match 如果值不等于模式（去除"!"前缀后）则返回true。
func (m *NotMatcher) Match(pattern string, value string) bool {
	// 移除"!"前缀以获取要取反的实际模式
	if len(pattern) == 0 {
		return true
	}

	negatedValue := pattern[1:] // 移除"!"前缀
	return value != negatedValue
}

// Type 返回匹配器的类型名称。
func (m *NotMatcher) Type() string {
	return "not"
}
