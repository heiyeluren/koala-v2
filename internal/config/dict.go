// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Dict 表示条目字典（如白名单/黑名单）。
// 使用map实现O(1)查找性能。
type Dict struct {
	Entries map[string]struct{}
	mu      sync.RWMutex
}

// NewDict 创建一个新的空字典。
func NewDict() *Dict {
	return &Dict{
		Entries: make(map[string]struct{}),
	}
}

// LoadDict 从文本文件加载字典。
// 每行被视为一个条目，支持以下特性：
// - 注释：以#开头的行会被忽略
// - 空行：被忽略
// - 空白：前后空白会被去除
// - 行内注释：行中#后的文本会被去除
func LoadDict(path string) (*Dict, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dict file %s: %w", path, err)
	}
	defer file.Close()

	dict := NewDict()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		entry := parseDictLine(line)
		if entry != "" {
			dict.Entries[entry] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read dict file %s: %w", path, err)
	}

	return dict, nil
}

// parseDictLine 解析字典文件中的单行。
// 如果该行应被忽略则返回空字符串。
func parseDictLine(line string) string {
	// 去除空白
	line = strings.TrimSpace(line)

	// 跳过空行
	if line == "" {
		return ""
	}

	// 跳过注释行
	if strings.HasPrefix(line, "#") {
		return ""
	}

	// 处理行内注释 - 去除#后的所有内容
	if idx := strings.Index(line, " #"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}

	return line
}

// Contains 检查字典中是否存在某个条目。
func (d *Dict) Contains(entry string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, exists := d.Entries[entry]
	return exists
}

// Add 向字典添加一个条目。
func (d *Dict) Add(entry string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Entries[entry] = struct{}{}
}

// Remove 从字典中移除一个条目。
func (d *Dict) Remove(entry string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.Entries, entry)
}

// List 返回所有条目的切片。
func (d *Dict) List() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	list := make([]string, 0, len(d.Entries))
	for entry := range d.Entries {
		list = append(list, entry)
	}
	return list
}

// Size 返回字典中的条目数量。
func (d *Dict) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.Entries)
}

// Clear 清空字典中的所有条目。
func (d *Dict) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Entries = make(map[string]struct{})
}

// Merge 将另一个字典合并到当前字典。
func (d *Dict) Merge(other *Dict) {
	if other == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for entry := range other.Entries {
		d.Entries[entry] = struct{}{}
	}
}

// DictManager 管理多个字典。
type DictManager struct {
	dicts  map[string]*Dict
	paths  map[string]string
	mu     sync.RWMutex
}

// NewDictManager 创建一个新的字典管理器。
func NewDictManager() *DictManager {
	return &DictManager{
		dicts: make(map[string]*Dict),
		paths: make(map[string]string),
	}
}

// LoadDicts 从名称->路径的映射加载多个字典。
func LoadDicts(dictPaths map[string]string) (*DictManager, error) {
	manager := NewDictManager()
	manager.paths = dictPaths

	for name, path := range dictPaths {
		dict, err := LoadDict(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load dict '%s': %w", name, err)
		}
		manager.dicts[name] = dict
	}

	return manager, nil
}

// Get 按名称获取字典。
func (m *DictManager) Get(name string) (*Dict, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dict, ok := m.dicts[name]
	return dict, ok
}

// Set 按名称设置字典。
func (m *DictManager) Set(name string, dict *Dict) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dicts[name] = dict
}

// Reload 从原始路径重新加载所有字典。
func (m *DictManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, path := range m.paths {
		dict, err := LoadDict(path)
		if err != nil {
			return fmt.Errorf("failed to reload dict '%s': %w", name, err)
		}
		m.dicts[name] = dict
	}

	return nil
}

// ReloadDict 按名称重新加载指定字典。
func (m *DictManager) ReloadDict(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path, ok := m.paths[name]
	if !ok {
		return fmt.Errorf("dict '%s' not found", name)
	}

	dict, err := LoadDict(path)
	if err != nil {
		return fmt.Errorf("failed to reload dict '%s': %w", name, err)
	}

	m.dicts[name] = dict
	return nil
}

// Contains 检查指定字典中是否存在某个条目。
func (m *DictManager) Contains(dictName, entry string) bool {
	dict, ok := m.Get(dictName)
	if !ok {
		return false
	}
	return dict.Contains(entry)
}

// List 返回所有字典名称。
func (m *DictManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.dicts))
	for name := range m.dicts {
		names = append(names, name)
	}
	return names
}
