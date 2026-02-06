// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// 规则类型常量。
const (
	RuleTypeCount      = "count"
	RuleTypeFreq       = "freq"
	RuleTypeAccumulate = "accumulate"
)

// dayRegexp 用于匹配天数格式时间字符串的预编译正则表达式。
var dayRegexp = regexp.MustCompile(`^(\d+)d$`)

// RulesConfig 表示从rules.toml加载的规则配置。
type RulesConfig struct {
	Meta    Meta                 `toml:"meta"`
	Dicts   map[string]string    `toml:"dicts"`
	Results map[string]Result    `toml:"results"`
	Access  AccessRules          `toml:"access"`
	Rules   RateRules            `toml:"rules"`
}

// Meta 包含规则配置的元数据。
type Meta struct {
	Version     string `toml:"version"`
	Description string `toml:"description"`
}

// Result 表示预定义的结果模板。
type Result struct {
	Code     int    `toml:"code"`
	Message  string `toml:"message"`
	AuthType int    `toml:"auth_type"`
}

// AccessRules 包含访问控制规则（白名单/黑名单）。
type AccessRules struct {
	Whitelist []AccessRule `toml:"whitelist"`
	Blacklist []AccessRule `toml:"blacklist"`
}

// AccessRule 表示单个访问控制规则。
type AccessRule struct {
	Name   string            `toml:"name"`
	Match  map[string]string `toml:"match"`
	Result string            `toml:"result"`
}

// RateRules 包含按类别组织的限流规则。
type RateRules struct {
	Business []RateRule `toml:"business"`
	Post     []RateRule `toml:"post"`
	Advanced []RateRule `toml:"advanced"`
	Default  []RateRule `toml:"default"`
}

// RateRule 表示单个限流规则。
type RateRule struct {
	Name   string            `toml:"name"`
	Type   string            `toml:"type"`
	Match  map[string]string `toml:"match"`
	Limit  Limit             `toml:"limit"`
	Result string            `toml:"result"`
	Desc   string            `toml:"desc"`
}

// Limit 表示限流参数。
type Limit struct {
	Time  time.Duration `toml:"time"`
	Count int           `toml:"count"`
	Base  int           `toml:"base"` // 累积阈值（仅 accumulate 类型使用）
}

// LoadRules 从TOML文件加载并验证规则配置。
func LoadRules(path string) (*RulesConfig, error) {
	var rules RulesConfig

	_, err := toml.DecodeFile(path, &rules)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rules file %s: %w", path, err)
	}

	// 如果map为nil则初始化
	if rules.Dicts == nil {
		rules.Dicts = make(map[string]string)
	}
	if rules.Results == nil {
		rules.Results = make(map[string]Result)
	}

	// 验证规则
	if err := validateRules(&rules); err != nil {
		return nil, err
	}

	return &rules, nil
}

// validateRules 验证规则配置。
func validateRules(rules *RulesConfig) error {
	// 验证版本
	if rules.Meta.Version == "" {
		return fmt.Errorf("rules version is required")
	}

	// 收集所有已定义的结果名称
	resultNames := make(map[string]bool)
	for name := range rules.Results {
		resultNames[name] = true
	}

	// 验证访问规则
	for _, rule := range rules.Access.Whitelist {
		if err := validateAccessRule(rule, resultNames); err != nil {
			return fmt.Errorf("invalid whitelist rule '%s': %w", rule.Name, err)
		}
	}
	for _, rule := range rules.Access.Blacklist {
		if err := validateAccessRule(rule, resultNames); err != nil {
			return fmt.Errorf("invalid blacklist rule '%s': %w", rule.Name, err)
		}
	}

	// 收集所有限流规则（使用独立切片，避免 append 链修改原始切片）
	var allRateRules []RateRule
	allRateRules = append(allRateRules, rules.Rules.Business...)
	allRateRules = append(allRateRules, rules.Rules.Post...)
	allRateRules = append(allRateRules, rules.Rules.Advanced...)
	allRateRules = append(allRateRules, rules.Rules.Default...)

	for _, rule := range allRateRules {
		if err := validateRateRule(rule, resultNames); err != nil {
			return fmt.Errorf("invalid rate rule '%s': %w", rule.Name, err)
		}
	}

	return nil
}

// validateAccessRule 验证单个访问规则。
func validateAccessRule(rule AccessRule, resultNames map[string]bool) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if len(rule.Match) == 0 {
		return fmt.Errorf("rule match is required")
	}
	if rule.Result == "" {
		return fmt.Errorf("rule result is required")
	}
	if !resultNames[rule.Result] {
		return fmt.Errorf("result '%s' is not defined", rule.Result)
	}
	return nil
}

// validateRateRule 验证单个限流规则。
func validateRateRule(rule RateRule, resultNames map[string]bool) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	// 验证规则类型
	validTypes := map[string]bool{
		RuleTypeCount:      true,
		RuleTypeFreq:       true,
		RuleTypeAccumulate: true,
	}
	if !validTypes[rule.Type] {
		return fmt.Errorf("invalid rule type '%s' (must be count, freq, or accumulate)", rule.Type)
	}

	if len(rule.Match) == 0 {
		return fmt.Errorf("rule match is required")
	}
	if rule.Limit.Count <= 0 {
		return fmt.Errorf("rule limit count must be positive")
	}
	if rule.Limit.Time <= 0 {
		return fmt.Errorf("rule limit time must be positive")
	}
	if rule.Result == "" {
		return fmt.Errorf("rule result is required")
	}
	if !resultNames[rule.Result] {
		return fmt.Errorf("result '%s' is not defined", rule.Result)
	}
	return nil
}

// ParseLimitTime 将时间字符串（如"1m"、"1h"、"1d"）解析为时间间隔。
func ParseLimitTime(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty time string")
	}

	// 首先检查天格式（time.ParseDuration不支持），使用预编译的包级别正则表达式
	if matches := dayRegexp.FindStringSubmatch(s); matches != nil {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid day value: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// 其他格式使用标准的time.ParseDuration
	dur, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid time format: %s", s)
	}
	return dur, nil
}

// IsDictReference 检查匹配值是否为字典引用（以@开头）。
func IsDictReference(value string) bool {
	return strings.HasPrefix(value, "@")
}

// GetDictName 从引用中提取字典名称（移除@前缀）。
func GetDictName(value string) string {
	if strings.HasPrefix(value, "@") {
		return value[1:]
	}
	return value
}

// IsWildcard 检查匹配值是否为通配符（+）。
func IsWildcard(value string) bool {
	return value == "+"
}

// GetAllRules 返回所有类别的所有限流规则。
func (r *RulesConfig) GetAllRules() []RateRule {
	var all []RateRule
	all = append(all, r.Rules.Business...)
	all = append(all, r.Rules.Post...)
	all = append(all, r.Rules.Advanced...)
	all = append(all, r.Rules.Default...)
	return all
}

// GetResult 按名称获取结果。
func (r *RulesConfig) GetResult(name string) (Result, bool) {
	result, ok := r.Results[name]
	return result, ok
}

// GetDictPath 按名称获取字典文件路径。
func (r *RulesConfig) GetDictPath(name string) (string, bool) {
	path, ok := r.Dicts[name]
	return path, ok
}
