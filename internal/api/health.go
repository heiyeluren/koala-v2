// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供Koala反作弊频率控制系统的健康检查功能。
package api

import (
	"context"
	"sync"
	"time"
)

// HealthChecker 健康检查器接口。
type HealthChecker interface {
	// Check 执行健康检查。
	Check(ctx context.Context) HealthStatus
	// Name 返回检查器名称。
	Name() string
}

// HealthStatus 健康状态。
type HealthStatus struct {
	Healthy bool          // 是否健康
	Message string        // 状态消息
	Latency time.Duration // 检查耗时
}

// HealthManager 管理多个健康检查器。
type HealthManager struct {
	mu       sync.RWMutex
	checkers []HealthChecker
	timeout  time.Duration
}

// NewHealthManager 创建新的健康管理器。
func NewHealthManager(timeout time.Duration) *HealthManager {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HealthManager{
		checkers: make([]HealthChecker, 0),
		timeout:  timeout,
	}
}

// Register 注册健康检查器。
func (m *HealthManager) Register(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

// CheckAll 执行所有健康检查。
func (m *HealthManager) CheckAll(ctx context.Context) map[string]HealthStatus {
	m.mu.RLock()
	checkers := make([]HealthChecker, len(m.checkers))
	copy(checkers, m.checkers)
	m.mu.RUnlock()

	// 为所有检查设置超时
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	results := make(map[string]HealthStatus)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	for _, checker := range checkers {
		wg.Add(1)
		go func(c HealthChecker) {
			defer wg.Done()
			start := time.Now()
			status := c.Check(ctx)
			status.Latency = time.Since(start)

			resultsMu.Lock()
			results[c.Name()] = status
			resultsMu.Unlock()
		}(checker)
	}

	wg.Wait()
	return results
}

// IsHealthy 检查所有组件是否健康。
func (m *HealthManager) IsHealthy(ctx context.Context) bool {
	results := m.CheckAll(ctx)
	for _, status := range results {
		if !status.Healthy {
			return false
		}
	}
	return true
}

// ========== 常用健康检查器实现 ==========

// StorageHealthChecker 存储健康检查器。
type StorageHealthChecker struct {
	name    string
	pingFn  func(ctx context.Context) error
}

// NewStorageHealthChecker 创建存储健康检查器。
func NewStorageHealthChecker(name string, pingFn func(ctx context.Context) error) *StorageHealthChecker {
	return &StorageHealthChecker{
		name:   name,
		pingFn: pingFn,
	}
}

// Check 实现HealthChecker接口。
func (c *StorageHealthChecker) Check(ctx context.Context) HealthStatus {
	if c.pingFn == nil {
		return HealthStatus{
			Healthy: true,
			Message: "无存储检查函数",
		}
	}

	err := c.pingFn(ctx)
	if err != nil {
		return HealthStatus{
			Healthy: false,
			Message: "存储连接失败: " + err.Error(),
		}
	}

	return HealthStatus{
		Healthy: true,
		Message: "存储连接正常",
	}
}

// Name 实现HealthChecker接口。
func (c *StorageHealthChecker) Name() string {
	return c.name
}

// EngineHealthChecker 引擎健康检查器。
type EngineHealthChecker struct {
	engine Engine
}

// NewEngineHealthChecker 创建引擎健康检查器。
func NewEngineHealthChecker(engine Engine) *EngineHealthChecker {
	return &EngineHealthChecker{
		engine: engine,
	}
}

// Check 实现HealthChecker接口。
func (c *EngineHealthChecker) Check(ctx context.Context) HealthStatus {
	if c.engine == nil {
		return HealthStatus{
			Healthy: false,
			Message: "引擎未初始化",
		}
	}

	// 执行简单的Browse检查
	_, err := c.engine.Browse(ctx, &EngineRequest{
		Act: "__health_check__",
	})
	if err != nil {
		return HealthStatus{
			Healthy: false,
			Message: "引擎检查失败: " + err.Error(),
		}
	}

	return HealthStatus{
		Healthy: true,
		Message: "引擎运行正常",
	}
}

// Name 实现HealthChecker接口。
func (c *EngineHealthChecker) Name() string {
	return "engine"
}

// ========== 就绪状态管理 ==========

// ReadinessManager 就绪状态管理器。
type ReadinessManager struct {
	mu    sync.RWMutex
	ready bool
}

// NewReadinessManager 创建就绪状态管理器。
func NewReadinessManager() *ReadinessManager {
	return &ReadinessManager{
		ready: false,
	}
}

// SetReady 设置就绪状态。
func (m *ReadinessManager) SetReady(ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = ready
}

// IsReady 检查是否就绪。
func (m *ReadinessManager) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// ========== 启动检查 ==========

// StartupChecker 启动检查器。
// 用于确保所有依赖组件在服务启动前就绪。
type StartupChecker struct {
	healthManager *HealthManager
	maxRetries    int
	retryInterval time.Duration
}

// NewStartupChecker 创建启动检查器。
func NewStartupChecker(healthManager *HealthManager, maxRetries int, retryInterval time.Duration) *StartupChecker {
	if maxRetries <= 0 {
		maxRetries = 10
	}
	if retryInterval <= 0 {
		retryInterval = time.Second
	}
	return &StartupChecker{
		healthManager: healthManager,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

// WaitForReady 等待所有组件就绪。
func (c *StartupChecker) WaitForReady(ctx context.Context) error {
	for i := 0; i < c.maxRetries; i++ {
		if c.healthManager.IsHealthy(ctx) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.retryInterval):
			// 继续重试
		}
	}

	return &StartupError{
		Message: "启动超时，部分组件未就绪",
		Results: c.healthManager.CheckAll(ctx),
	}
}

// StartupError 启动错误。
type StartupError struct {
	Message string
	Results map[string]HealthStatus
}

// Error 实现error接口。
func (e *StartupError) Error() string {
	return e.Message
}
