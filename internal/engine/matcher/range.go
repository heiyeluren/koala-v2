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

// RangeMatcher 当数值在指定范围内时匹配。
// 模式格式："min-max"表示当min <= value <= max时匹配。
// 支持负数："-10-10"表示从-10到10。
type RangeMatcher struct{}

// Match 如果值在模式指定的范围内则返回true。
func (m *RangeMatcher) Match(pattern string, value string) bool {
	// 首先解析值
	val, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false
	}

	// 解析模式范围
	min, max, ok := parseRange(pattern)
	if !ok {
		return false
	}

	return val >= min && val <= max
}

// parseRange 解析范围模式，如"1-100"或"-10-10"。
// 返回（最小值，最大值，是否成功）。
func parseRange(pattern string) (int64, int64, bool) {
	// 查找分隔最小值和最大值的短横线
	dashIdx := findRangeDashIndex(pattern)
	if dashIdx == -1 {
		return 0, 0, false
	}

	minStr := pattern[:dashIdx]
	maxStr := pattern[dashIdx+1:]

	min, err := strconv.ParseInt(minStr, 10, 64)
	if err != nil {
		return 0, 0, false
	}

	max, err := strconv.ParseInt(maxStr, 10, 64)
	if err != nil {
		return 0, 0, false
	}

	return min, max, true
}

// findRangeDashIndex 查找分隔最小值和最大值的短横线索引。
// 对于"-10-10"，返回3（第二个短横线）。
// 对于"1-100"，返回1。
func findRangeDashIndex(pattern string) int {
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

// Type 返回匹配器的类型名称。
func (m *RangeMatcher) Type() string {
	return "range"
}
