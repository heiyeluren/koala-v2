// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher 提供Koala频率限制规则的模式匹配功能。
package matcher

import "sync"

// DictMatcher 根据命名字典匹配值。
// 模式格式："@dict_name"表示当值在名为"dict_name"的字典中时匹配。
// 使用前必须通过RegisterDict或RegisterDictSlice注册字典。
type DictMatcher struct {
	mu    sync.RWMutex
	dicts map[string]map[string]bool
}

// NewDictMatcher 创建新的DictMatcher实例。
func NewDictMatcher() *DictMatcher {
	return &DictMatcher{
		dicts: make(map[string]map[string]bool),
	}
}

// Match 如果值在模式指定的字典中则返回true。
func (m *DictMatcher) Match(pattern string, value string) bool {
	// 模式格式："@dict_name"
	if len(pattern) < 2 || pattern[0] != '@' {
		return false
	}

	dictName := pattern[1:]
	if dictName == "" {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	dict, exists := m.dicts[dictName]
	if !exists {
		return false
	}

	return dict[value]
}

// Type 返回匹配器的类型名称。
func (m *DictMatcher) Type() string {
	return "dict"
}

// RegisterDict 使用给定名称注册字典。
// values应为一个map，其中键是要匹配的值，值为true。
func (m *DictMatcher) RegisterDict(name string, values map[string]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dicts[name] = values
}

// RegisterDictSlice 从值切片注册字典。
func (m *DictMatcher) RegisterDictSlice(name string, values []string) {
	dict := make(map[string]bool, len(values))
	for _, v := range values {
		dict[v] = true
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.dicts[name] = dict
}

// UnregisterDict 按名称移除字典。
func (m *DictMatcher) UnregisterDict(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.dicts, name)
}

// HasDict 检查字典是否存在。
func (m *DictMatcher) HasDict(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.dicts[name]
	return exists
}
