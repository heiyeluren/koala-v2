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

// IPMatcher 使用通配符模式匹配IP地址。
// 模式格式："192.168.*.*"表示匹配以192.168开头的任何IP。
// 通配符可用于任何八位组位置。
type IPMatcher struct{}

// Match 如果IP值与通配符模式匹配则返回true。
func (m *IPMatcher) Match(pattern string, value string) bool {
	patternParts := strings.Split(pattern, ".")
	valueParts := strings.Split(value, ".")

	// 两者必须都有4个部分（IPv4）
	if len(patternParts) != 4 || len(valueParts) != 4 {
		return false
	}

	// 比较每个八位组
	for i := 0; i < 4; i++ {
		// 通配符匹配任何值
		if patternParts[i] == "*" {
			continue
		}
		// 需要精确匹配
		if patternParts[i] != valueParts[i] {
			return false
		}
	}

	return true
}

// Type 返回匹配器的类型名称。
func (m *IPMatcher) Type() string {
	return "ip"
}
