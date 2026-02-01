// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package engine

import (
	"context"
	"fmt"
	"sync/atomic"

	"koala/internal/config"
	"koala/internal/engine/algorithm"
	"koala/internal/storage"
)

// Engine 是Koala反作弊频率控制系统的核心规则引擎。
// 支持规则热加载和并发安全的规则执行。
type Engine struct {
	rules   atomic.Pointer[RuleSet]     // 原子指针，支持热加载
	storage storage.Storage             // 存储后端
	dicts   *config.DictManager         // 字典管理器
}

// Option 引擎配置选项。
type Option func(*Engine)

// WithStorage 设置存储后端。
func WithStorage(s storage.Storage) Option {
	return func(e *Engine) {
		e.storage = s
	}
}

// WithDicts 设置字典管理器。
func WithDicts(dicts *config.DictManager) Option {
	return func(e *Engine) {
		e.dicts = dicts
	}
}

// New 创建新的规则引擎实例。
func New(opts ...Option) *Engine {
	e := &Engine{}
	for _, opt := range opts {
		opt(e)
	}
	// 初始化空规则集
	e.rules.Store(NewRuleSet())
	return e
}

// LoadRules 从配置加载规则。
// 使用原子指针交换实现热加载，无需停机。
func (e *Engine) LoadRules(rulesConfig *config.RulesConfig) error {
	ruleSet, err := BuildRuleSet(rulesConfig)
	if err != nil {
		return fmt.Errorf("build rule set failed: %w", err)
	}

	// 原子替换规则集
	e.rules.Store(ruleSet)
	return nil
}

// GetRuleSet 获取当前规则集。
func (e *Engine) GetRuleSet() *RuleSet {
	return e.rules.Load()
}

// SetStorage 设置存储后端。
func (e *Engine) SetStorage(s storage.Storage) {
	e.storage = s
}

// SetDicts 设置字典管理器。
func (e *Engine) SetDicts(dicts *config.DictManager) {
	e.dicts = dicts
}

// Check 执行限流检查。
// 按照规定的阶段顺序执行规则：
//   1. 阶段1（访问控制）：白名单 -> 黑名单
//   2. 阶段2（限流规则）：业务 -> 发帖 -> 高级 -> 默认
//
// 白名单匹配：直接允许
// 黑名单匹配：直接拒绝
// 限流规则匹配：检查是否超过限制
func (e *Engine) Check(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	ruleSet := e.rules.Load()
	if ruleSet == nil {
		// 没有规则，默认允许
		return NewAllowedResponse(), nil
	}

	// 阶段1：访问控制
	// 1.1 检查白名单（匹配则直接允许）
	for _, rule := range ruleSet.Whitelist {
		if rule.Matches(req) {
			return &Response{
				Allowed:  true,
				Code:     rule.Result.Code,
				Message:  rule.Result.Message,
				RuleName: rule.Name,
				AuthType: rule.Result.AuthType,
			}, nil
		}
	}

	// 1.2 检查黑名单（匹配则直接拒绝）
	for _, rule := range ruleSet.Blacklist {
		if rule.Matches(req) {
			return &Response{
				Allowed:  false,
				Code:     rule.Result.Code,
				Message:  rule.Result.Message,
				RuleName: rule.Name,
				AuthType: rule.Result.AuthType,
			}, nil
		}
	}

	// 阶段2：限流规则
	// 按优先级顺序检查：业务 -> 发帖 -> 高级 -> 默认
	phases := [][]*Rule{
		ruleSet.Business,
		ruleSet.Post,
		ruleSet.Advanced,
		ruleSet.Default,
	}

	for _, rules := range phases {
		for _, rule := range rules {
			if rule.Matches(req) {
				// 检查是否触发限流
				hit, err := e.checkRateLimit(ctx, rule, req)
				if err != nil {
					return nil, fmt.Errorf("check rate limit failed: %w", err)
				}

				if hit {
					// 触发限流，拒绝请求
					return &Response{
						Allowed:  false,
						Code:     rule.Result.Code,
						Message:  rule.Result.Message,
						RuleName: rule.Name,
						AuthType: rule.Result.AuthType,
					}, nil
				}

				// 未触发限流，更新计数器并允许
				if err := e.updateCounter(ctx, rule, req); err != nil {
					// 更新计数器失败不应阻止请求
					// 但应记录错误（实际应用中应使用日志）
					_ = err
				}

				// 规则已匹配且未触发限流，允许请求
				return NewAllowedResponse(), nil
			}
		}
	}

	// 没有匹配任何规则，默认允许
	return NewAllowedResponse(), nil
}

// checkRateLimit 检查是否触发限流。
func (e *Engine) checkRateLimit(ctx context.Context, rule *Rule, req *Request) (bool, error) {
	if e.storage == nil {
		// 没有存储后端，无法检查限流，默认不触发
		return false, nil
	}

	key := rule.GenerateKey(req)
	limitCfg := algorithm.LimitConfig{
		Time:  rule.Limit.Time,
		Count: rule.Limit.Count,
	}

	return rule.Algorithm.Browse(ctx, key, limitCfg, e.storage)
}

// updateCounter 更新限流计数器。
func (e *Engine) updateCounter(ctx context.Context, rule *Rule, req *Request) error {
	if e.storage == nil {
		return nil
	}

	key := rule.GenerateKey(req)
	limitCfg := algorithm.LimitConfig{
		Time:  rule.Limit.Time,
		Count: rule.Limit.Count,
	}

	return rule.Algorithm.Update(ctx, key, limitCfg, e.storage)
}

// Browse 仅检查限流状态，不更新计数器。
// 用于只需要查询当前状态的场景。
func (e *Engine) Browse(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	ruleSet := e.rules.Load()
	if ruleSet == nil {
		return NewAllowedResponse(), nil
	}

	// 阶段1：访问控制
	for _, rule := range ruleSet.Whitelist {
		if rule.Matches(req) {
			return &Response{
				Allowed:  true,
				Code:     rule.Result.Code,
				Message:  rule.Result.Message,
				RuleName: rule.Name,
				AuthType: rule.Result.AuthType,
			}, nil
		}
	}

	for _, rule := range ruleSet.Blacklist {
		if rule.Matches(req) {
			return &Response{
				Allowed:  false,
				Code:     rule.Result.Code,
				Message:  rule.Result.Message,
				RuleName: rule.Name,
				AuthType: rule.Result.AuthType,
			}, nil
		}
	}

	// 阶段2：限流规则（只检查不更新）
	phases := [][]*Rule{
		ruleSet.Business,
		ruleSet.Post,
		ruleSet.Advanced,
		ruleSet.Default,
	}

	for _, rules := range phases {
		for _, rule := range rules {
			if rule.Matches(req) {
				hit, err := e.checkRateLimit(ctx, rule, req)
				if err != nil {
					return nil, fmt.Errorf("check rate limit failed: %w", err)
				}

				if hit {
					return &Response{
						Allowed:  false,
						Code:     rule.Result.Code,
						Message:  rule.Result.Message,
						RuleName: rule.Name,
						AuthType: rule.Result.AuthType,
					}, nil
				}

				return NewAllowedResponse(), nil
			}
		}
	}

	return NewAllowedResponse(), nil
}
