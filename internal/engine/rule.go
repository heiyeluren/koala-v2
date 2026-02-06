// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package engine

import (
	"fmt"
	"sort"
	"time"

	"koala/internal/config"
	"koala/internal/engine/algorithm"
	"koala/internal/engine/matcher"
)

// RuleType 规则类型。
type RuleType int

const (
	// RuleTypeAccess 访问控制规则（白名单/黑名单）。
	RuleTypeAccess RuleType = iota
	// RuleTypeRate 限流规则。
	RuleTypeRate
)

// RulePhase 规则执行阶段。
type RulePhase int

const (
	// PhaseWhitelist 白名单阶段（优先级最高）。
	PhaseWhitelist RulePhase = iota
	// PhaseBlacklist 黑名单阶段。
	PhaseBlacklist
	// PhaseBusiness 业务规则阶段。
	PhaseBusiness
	// PhasePost 发帖规则阶段。
	PhasePost
	// PhaseAdvanced 高级规则阶段。
	PhaseAdvanced
	// PhaseDefault 默认规则阶段（优先级最低）。
	PhaseDefault
)

// Rule 表示一个规则。
type Rule struct {
	Name      string              // 规则名称
	Type      RuleType            // 规则类型
	Phase     RulePhase           // 执行阶段
	Match     map[string]string   // 匹配条件（字段名 -> 模式）
	Matchers  map[string]matcher.Matcher // 预编译的匹配器
	Limit     LimitConfig         // 限流配置（仅限流规则有效）
	Algorithm algorithm.Algorithm // 限流算法（仅限流规则有效）
	Result    ResultConfig        // 匹配后的结果
}

// LimitConfig 限流配置。
type LimitConfig struct {
	Time  time.Duration // 时间窗口
	Count int64         // 窗口内最大请求数
	Base  int64         // 累积阈值（仅 Base 算法使用）
}

// ResultConfig 结果配置。
type ResultConfig struct {
	Code     int    // 响应码
	Message  string // 响应消息
	AuthType int    // 验证类型
}

// RuleSet 规则集合，按阶段组织规则。
type RuleSet struct {
	Whitelist []*Rule // 白名单规则
	Blacklist []*Rule // 黑名单规则
	Business  []*Rule // 业务规则
	Post      []*Rule // 发帖规则
	Advanced  []*Rule // 高级规则
	Default   []*Rule // 默认规则
}

// NewRuleSet 创建空规则集。
func NewRuleSet() *RuleSet {
	return &RuleSet{
		Whitelist: make([]*Rule, 0),
		Blacklist: make([]*Rule, 0),
		Business:  make([]*Rule, 0),
		Post:      make([]*Rule, 0),
		Advanced:  make([]*Rule, 0),
		Default:   make([]*Rule, 0),
	}
}

// Matches 检查请求是否匹配规则的所有条件。
func (r *Rule) Matches(req *Request) bool {
	for field, pattern := range r.Match {
		value := req.GetField(field)
		m := r.Matchers[field]
		if m == nil {
			// 如果没有预编译的匹配器，动态解析
			m = matcher.Parse(pattern)
		}
		if !m.Match(pattern, value) {
			return false
		}
	}
	return true
}

// GenerateKey 生成用于限流计数的存储键。
// 键格式：koala:{ruleName}:{field1}={value1}:{field2}={value2}...
// 注意：字段按字母顺序排序以确保键的一致性。
func (r *Rule) GenerateKey(req *Request) string {
	key := fmt.Sprintf("koala:%s", r.Name)

	// 收集字段名并排序，确保键的顺序一致
	fields := make([]string, 0, len(r.Match))
	for field := range r.Match {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// 按排序后的顺序生成键
	for _, field := range fields {
		value := req.GetField(field)
		key = fmt.Sprintf("%s:%s=%s", key, field, value)
	}
	return key
}

// BuildRuleSet 从规则配置构建规则集。
func BuildRuleSet(rulesConfig *config.RulesConfig) (*RuleSet, error) {
	rs := NewRuleSet()

	// 构建白名单规则
	for _, ar := range rulesConfig.Access.Whitelist {
		rule, err := buildAccessRule(ar, PhaseWhitelist, rulesConfig)
		if err != nil {
			return nil, fmt.Errorf("build whitelist rule '%s' failed: %w", ar.Name, err)
		}
		rs.Whitelist = append(rs.Whitelist, rule)
	}

	// 构建黑名单规则
	for _, ar := range rulesConfig.Access.Blacklist {
		rule, err := buildAccessRule(ar, PhaseBlacklist, rulesConfig)
		if err != nil {
			return nil, fmt.Errorf("build blacklist rule '%s' failed: %w", ar.Name, err)
		}
		rs.Blacklist = append(rs.Blacklist, rule)
	}

	// 构建业务规则
	for _, rr := range rulesConfig.Rules.Business {
		rule, err := buildRateRule(rr, PhaseBusiness, rulesConfig)
		if err != nil {
			return nil, fmt.Errorf("build business rule '%s' failed: %w", rr.Name, err)
		}
		rs.Business = append(rs.Business, rule)
	}

	// 构建发帖规则
	for _, rr := range rulesConfig.Rules.Post {
		rule, err := buildRateRule(rr, PhasePost, rulesConfig)
		if err != nil {
			return nil, fmt.Errorf("build post rule '%s' failed: %w", rr.Name, err)
		}
		rs.Post = append(rs.Post, rule)
	}

	// 构建高级规则
	for _, rr := range rulesConfig.Rules.Advanced {
		rule, err := buildRateRule(rr, PhaseAdvanced, rulesConfig)
		if err != nil {
			return nil, fmt.Errorf("build advanced rule '%s' failed: %w", rr.Name, err)
		}
		rs.Advanced = append(rs.Advanced, rule)
	}

	// 构建默认规则
	for _, rr := range rulesConfig.Rules.Default {
		rule, err := buildRateRule(rr, PhaseDefault, rulesConfig)
		if err != nil {
			return nil, fmt.Errorf("build default rule '%s' failed: %w", rr.Name, err)
		}
		rs.Default = append(rs.Default, rule)
	}

	return rs, nil
}

// buildAccessRule 构建访问控制规则。
func buildAccessRule(ar config.AccessRule, phase RulePhase, rulesConfig *config.RulesConfig) (*Rule, error) {
	result, ok := rulesConfig.GetResult(ar.Result)
	if !ok {
		return nil, fmt.Errorf("result '%s' not found", ar.Result)
	}

	rule := &Rule{
		Name:     ar.Name,
		Type:     RuleTypeAccess,
		Phase:    phase,
		Match:    ar.Match,
		Matchers: make(map[string]matcher.Matcher),
		// 访问控制规则使用Direct算法
		Algorithm: algorithm.NewDirect(),
		Result: ResultConfig{
			Code:     result.Code,
			Message:  result.Message,
			AuthType: result.AuthType,
		},
	}

	// 预编译匹配器
	for field, pattern := range ar.Match {
		rule.Matchers[field] = matcher.Parse(pattern)
	}

	return rule, nil
}

// buildRateRule 构建限流规则。
func buildRateRule(rr config.RateRule, phase RulePhase, rulesConfig *config.RulesConfig) (*Rule, error) {
	result, ok := rulesConfig.GetResult(rr.Result)
	if !ok {
		return nil, fmt.Errorf("result '%s' not found", rr.Result)
	}

	// 根据规则类型选择算法
	var alg algorithm.Algorithm
	switch rr.Type {
	case config.RuleTypeCount:
		alg = algorithm.NewCount()
	case config.RuleTypeFreq:
		alg = algorithm.NewLeak()
	case config.RuleTypeAccumulate:
		alg = algorithm.NewBase()
	default:
		alg = algorithm.NewCount()
	}

	rule := &Rule{
		Name:      rr.Name,
		Type:      RuleTypeRate,
		Phase:     phase,
		Match:     rr.Match,
		Matchers:  make(map[string]matcher.Matcher),
		Algorithm: alg,
		Limit: LimitConfig{
			Time:  rr.Limit.Time,
			Count: int64(rr.Limit.Count),
			Base:  int64(rr.Limit.Base),
		},
		Result: ResultConfig{
			Code:     result.Code,
			Message:  result.Message,
			AuthType: result.AuthType,
		},
	}

	// 预编译匹配器
	for field, pattern := range rr.Match {
		rule.Matchers[field] = matcher.Parse(pattern)
	}

	return rule, nil
}
