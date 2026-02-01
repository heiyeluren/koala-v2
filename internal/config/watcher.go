// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package config

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatcherCallback 当被监视的文件发生变化时调用。
type WatcherCallback func(path string) error

// Watcher 监视配置文件的变化并触发回调。
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	callbacks    map[string][]WatcherCallback
	debounce     time.Duration
	lastModified map[string]time.Time
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
	started      bool
}

// WatcherConfig 包含文件监视器的配置。
type WatcherConfig struct {
	// Debounce 是同一文件触发回调的最小间隔时间。
	// 这有助于避免单次保存操作触发多次回调。
	// 默认为100毫秒。
	Debounce time.Duration
}

// DefaultWatcherConfig 返回默认的监视器配置。
func DefaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
		Debounce: 100 * time.Millisecond,
	}
}

// NewWatcher 创建一个新的文件监视器。
func NewWatcher(cfg WatcherConfig) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	if cfg.Debounce == 0 {
		cfg.Debounce = 100 * time.Millisecond
	}

	w := &Watcher{
		fsWatcher:    fsWatcher,
		callbacks:    make(map[string][]WatcherCallback),
		debounce:     cfg.Debounce,
		lastModified: make(map[string]time.Time),
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
	}

	return w, nil
}

// Watch 添加要监视的文件并注册回调。
func (w *Watcher) Watch(path string, callback WatcherCallback) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	// 监视包含该文件的目录
	// 这是必要的，因为某些编辑器（如vim）会创建新文件而不是修改现有文件
	dir := filepath.Dir(absPath)

	w.mu.Lock()
	defer w.mu.Unlock()

	// 为此文件添加回调
	w.callbacks[absPath] = append(w.callbacks[absPath], callback)

	// 如果尚未监视则监视该目录
	if err := w.fsWatcher.Add(dir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	return nil
}

// Unwatch 从监视列表中移除文件。
func (w *Watcher) Unwatch(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.callbacks, absPath)

	return nil
}

// Start 开始监视文件变化。
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return
	}
	w.started = true
	w.mu.Unlock()
	go w.eventLoop()
}

// eventLoop 处理文件系统事件。
func (w *Watcher) eventLoop() {
	defer close(w.done)

	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

// handleEvent 处理单个文件系统事件。
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// 仅处理写入和创建事件
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
		return
	}

	absPath, err := filepath.Abs(event.Name)
	if err != nil {
		log.Printf("failed to get absolute path for %s: %v", event.Name, err)
		return
	}

	w.mu.RLock()
	callbacks, ok := w.callbacks[absPath]
	lastMod := w.lastModified[absPath]
	w.mu.RUnlock()

	if !ok || len(callbacks) == 0 {
		return
	}

	// 防抖：如果最近刚修改过则跳过
	now := time.Now()
	if now.Sub(lastMod) < w.debounce {
		return
	}

	w.mu.Lock()
	w.lastModified[absPath] = now
	w.mu.Unlock()

	// 执行回调
	for _, callback := range callbacks {
		if err := callback(absPath); err != nil {
			log.Printf("callback error for %s: %v", absPath, err)
		}
	}
}

// Stop 停止监视器。
func (w *Watcher) Stop() error {
	w.cancel()

	// 仅当监视器已启动时才等待done通道
	w.mu.RLock()
	started := w.started
	w.mu.RUnlock()

	if started {
		<-w.done
	}

	if err := w.fsWatcher.Close(); err != nil {
		return fmt.Errorf("failed to close fsnotify watcher: %w", err)
	}

	return nil
}

// WatchedFiles 返回正在监视的文件列表。
func (w *Watcher) WatchedFiles() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	files := make([]string, 0, len(w.callbacks))
	for path := range w.callbacks {
		files = append(files, path)
	}
	return files
}

// ConfigWatcher 是配置文件的专用监视器。
type ConfigWatcher struct {
	watcher    *Watcher
	config     *Config
	rules      *RulesConfig
	dicts      *DictManager
	configPath string
	rulesPath  string
	mu         sync.RWMutex
	onChange   func(event ConfigChangeEvent)
}

// ConfigChangeEvent 表示配置变更事件。
type ConfigChangeEvent struct {
	Type      ConfigChangeType
	Path      string
	Error     error
	Timestamp time.Time
}

// ConfigChangeType 表示变更的配置类型。
type ConfigChangeType int

const (
	ConfigChangeTypeConfig ConfigChangeType = iota
	ConfigChangeTypeRules
	ConfigChangeTypeDict
)

func (t ConfigChangeType) String() string {
	switch t {
	case ConfigChangeTypeConfig:
		return "config"
	case ConfigChangeTypeRules:
		return "rules"
	case ConfigChangeTypeDict:
		return "dict"
	default:
		return "unknown"
	}
}

// NewConfigWatcher 创建一个新的配置监视器。
func NewConfigWatcher(configPath string, onChange func(ConfigChangeEvent)) (*ConfigWatcher, error) {
	watcher, err := NewWatcher(DefaultWatcherConfig())
	if err != nil {
		return nil, err
	}

	cw := &ConfigWatcher{
		watcher:    watcher,
		configPath: configPath,
		onChange:   onChange,
	}

	return cw, nil
}

// Load 加载初始配置。
func (cw *ConfigWatcher) Load() error {
	// 加载主配置
	config, err := LoadConfig(cw.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cw.mu.Lock()
	cw.config = config
	cw.rulesPath = config.Rules.File
	cw.mu.Unlock()

	// 如果指定了规则文件则加载
	if cw.rulesPath != "" {
		rules, err := LoadRules(cw.rulesPath)
		if err != nil {
			return fmt.Errorf("failed to load rules: %w", err)
		}

		cw.mu.Lock()
		cw.rules = rules
		cw.mu.Unlock()

		// 加载字典
		if len(rules.Dicts) > 0 {
			dicts, err := LoadDicts(rules.Dicts)
			if err != nil {
				return fmt.Errorf("failed to load dicts: %w", err)
			}

			cw.mu.Lock()
			cw.dicts = dicts
			cw.mu.Unlock()
		}
	}

	return nil
}

// StartWatching 开始监视配置文件的变化。
func (cw *ConfigWatcher) StartWatching() error {
	// 监视主配置文件
	if err := cw.watcher.Watch(cw.configPath, cw.onConfigChange); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	// 监视规则文件
	cw.mu.RLock()
	rulesPath := cw.rulesPath
	rules := cw.rules
	cw.mu.RUnlock()

	if rulesPath != "" {
		if err := cw.watcher.Watch(rulesPath, cw.onRulesChange); err != nil {
			return fmt.Errorf("failed to watch rules file: %w", err)
		}
	}

	// 监视字典文件
	if rules != nil {
		for name, path := range rules.Dicts {
			dictName := name // 为闭包捕获变量
			if err := cw.watcher.Watch(path, func(p string) error {
				return cw.onDictChange(dictName, p)
			}); err != nil {
				return fmt.Errorf("failed to watch dict file %s: %w", name, err)
			}
		}
	}

	cw.watcher.Start()
	return nil
}

// onConfigChange 处理主配置文件的变化。
func (cw *ConfigWatcher) onConfigChange(path string) error {
	config, err := LoadConfig(path)

	event := ConfigChangeEvent{
		Type:      ConfigChangeTypeConfig,
		Path:      path,
		Error:     err,
		Timestamp: time.Now(),
	}

	if err == nil {
		cw.mu.Lock()
		cw.config = config
		cw.mu.Unlock()
	}

	if cw.onChange != nil {
		cw.onChange(event)
	}

	return err
}

// onRulesChange 处理规则文件的变化。
func (cw *ConfigWatcher) onRulesChange(path string) error {
	rules, err := LoadRules(path)

	event := ConfigChangeEvent{
		Type:      ConfigChangeTypeRules,
		Path:      path,
		Error:     err,
		Timestamp: time.Now(),
	}

	if err == nil {
		cw.mu.Lock()
		cw.rules = rules
		cw.mu.Unlock()
	}

	if cw.onChange != nil {
		cw.onChange(event)
	}

	return err
}

// onDictChange 处理字典文件的变化。
func (cw *ConfigWatcher) onDictChange(name, path string) error {
	dict, err := LoadDict(path)

	event := ConfigChangeEvent{
		Type:      ConfigChangeTypeDict,
		Path:      path,
		Error:     err,
		Timestamp: time.Now(),
	}

	if err == nil {
		cw.mu.Lock()
		if cw.dicts != nil {
			cw.dicts.Set(name, dict)
		}
		cw.mu.Unlock()
	}

	if cw.onChange != nil {
		cw.onChange(event)
	}

	return err
}

// GetConfig 返回当前配置。
func (cw *ConfigWatcher) GetConfig() *Config {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.config
}

// GetRules 返回当前规则配置。
func (cw *ConfigWatcher) GetRules() *RulesConfig {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.rules
}

// GetDicts 返回当前字典管理器。
func (cw *ConfigWatcher) GetDicts() *DictManager {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.dicts
}

// Stop 停止配置监视器。
func (cw *ConfigWatcher) Stop() error {
	return cw.watcher.Stop()
}
