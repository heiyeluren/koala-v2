// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
// 支持多种匹配策略，包括精确匹配、通配符匹配、范围匹配和基于字典的匹配。
package matcher

import (
	"strings"
	"unicode"
)

// Matcher 定义模式匹配的接口。
type Matcher interface {
	// Match 检查值是否与模式匹配。
	// 如果值与模式匹配则返回true。
	Match(pattern string, value string) bool

	// Type 返回匹配器的类型名称。
	Type() string
}

// defaultDictMatcher 是共享的字典匹配器实例。
var defaultDictMatcher = NewDictMatcher()

// Parse 分析模式并返回适当的匹配器。
// 模式前缀决定匹配器类型：
//   - "+" -> AnyMatcher（匹配任何非空值）
//   - "!" -> NotMatcher（当值不等于模式时匹配）
//   - "@" -> DictMatcher（当值在字典中时匹配）
//   - ">" -> GreaterMatcher（当值大于阈值时匹配）
//   - "<" -> LessMatcher（当值小于阈值时匹配）
//   - 包含"," -> MultiMatcher（当值在列表中时匹配）
//   - 包含"-"且两侧为数字 -> RangeMatcher
//   - 包含"*"的IP格式 -> IPMatcher
//   - 默认 -> ExactMatcher
func Parse(pattern string) Matcher {
	if len(pattern) == 0 {
		return &ExactMatcher{}
	}

	// 首先检查基于前缀的匹配器
	switch pattern[0] {
	case '+':
		return &AnyMatcher{}
	case '!':
		return &NotMatcher{}
	case '@':
		return defaultDictMatcher
	case '>':
		return &GreaterMatcher{}
	case '<':
		return &LessMatcher{}
	}

	// 检查多值模式（包含逗号）
	if strings.Contains(pattern, ",") {
		return &MultiMatcher{}
	}

	// 检查范围模式（数字之间包含短横线）
	if isRangePattern(pattern) {
		return &RangeMatcher{}
	}

	// 检查IP通配符模式
	if isIPWildcardPattern(pattern) {
		return &IPMatcher{}
	}

	// 默认使用精确匹配
	return &ExactMatcher{}
}

// isRangePattern 检查模式是否为有效范围（例如："1-100"、"-10-10"）。
func isRangePattern(pattern string) bool {
	// 查找分隔最小值和最大值的短横线
	// 处理负数："-10-10"表示从-10到10
	dashIdx := findRangeDash(pattern)
	if dashIdx == -1 {
		return false
	}

	minStr := pattern[:dashIdx]
	maxStr := pattern[dashIdx+1:]

	return isInteger(minStr) && isInteger(maxStr)
}

// findRangeDash 查找范围模式中分隔最小值和最大值的短横线索引。
// 对于"-10-10"，返回3（第二个短横线）。
// 对于"1-100"，返回1。
func findRangeDash(pattern string) int {
	// 从索引1开始，跳过可能的前导负号
	startIdx := 0
	if len(pattern) > 0 && pattern[0] == '-' {
		startIdx = 1
	}

	for i := startIdx; i < len(pattern); i++ {
		if pattern[i] == '-' {
			return i
		}
	}
	return -1
}

// isInteger 检查字符串是否表示有效整数。
func isInteger(s string) bool {
	if len(s) == 0 {
		return false
	}

	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}

	if start >= len(s) {
		return false
	}

	for i := start; i < len(s); i++ {
		if !unicode.IsDigit(rune(s[i])) {
			return false
		}
	}
	return true
}

// isIPWildcardPattern 检查模式是否为带通配符的IP地址。
func isIPWildcardPattern(pattern string) bool {
	// 必须包含通配符
	if !strings.Contains(pattern, "*") {
		return false
	}

	// 必须有4个由点分隔的部分
	parts := strings.Split(pattern, ".")
	return len(parts) == 4
}

// RegisterDict 向默认字典匹配器注册字典。
func RegisterDict(name string, values map[string]bool) {
	defaultDictMatcher.RegisterDict(name, values)
}

// RegisterDictSlice 使用切片向默认匹配器注册字典。
func RegisterDictSlice(name string, values []string) {
	defaultDictMatcher.RegisterDictSlice(name, values)
}
