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
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

// ValidationError 表示配置验证错误。
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for '%s': %s", e.Field, e.Message)
}

// ValidationErrors 是验证错误的集合。
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors 如果存在验证错误则返回true。
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator 提供配置验证功能。
type Validator struct {
	errors ValidationErrors
}

// NewValidator 创建一个新的验证器。
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError 添加一个验证错误。
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{Field: field, Message: message})
}

// GetErrors 返回所有验证错误。
func (v *Validator) GetErrors() ValidationErrors {
	return v.errors
}

// HasErrors 如果存在验证错误则返回true。
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// ValidateConfig 对服务配置进行全面验证。
func ValidateConfig(cfg *Config) ValidationErrors {
	v := NewValidator()

	// 验证服务器配置
	v.validateServer(&cfg.Server)

	// 验证存储配置
	v.validateStorage(&cfg.Storage)

	// 验证日志配置
	v.validateLogging(&cfg.Logging)

	// 验证指标配置
	v.validateMetrics(&cfg.Metrics)

	// 验证规则引用
	v.validateRulesRef(&cfg.Rules)

	return v.GetErrors()
}

// validateServer 验证服务器配置。
func (v *Validator) validateServer(cfg *ServerConfig) {
	// 验证监听地址
	if cfg.Listen == "" {
		v.AddError("server.listen", "listen address is required")
	} else if !isValidListenAddress(cfg.Listen) {
		v.AddError("server.listen", "invalid listen address format")
	}

	// 验证超时设置
	if cfg.ReadTimeout < 0 {
		v.AddError("server.read_timeout", "must be non-negative")
	}
	if cfg.WriteTimeout < 0 {
		v.AddError("server.write_timeout", "must be non-negative")
	}
	if cfg.ShutdownTimeout < 0 {
		v.AddError("server.shutdown_timeout", "must be non-negative")
	}

	// 警告异常长的超时时间
	if cfg.ReadTimeout > 5*time.Minute {
		v.AddError("server.read_timeout", "unusually long timeout (>5m)")
	}
	if cfg.WriteTimeout > 5*time.Minute {
		v.AddError("server.write_timeout", "unusually long timeout (>5m)")
	}
}

// validateStorage 验证存储配置。
func (v *Validator) validateStorage(cfg *StorageConfig) {
	// 验证存储类型
	validTypes := []string{StorageTypeLocal, StorageTypeRedis}
	if !contains(validTypes, cfg.Type) {
		v.AddError("storage.type", fmt.Sprintf("must be one of: %s", strings.Join(validTypes, ", ")))
	}

	// 验证本地存储配置
	if cfg.Type == StorageTypeLocal {
		v.validateLocalStorage(&cfg.Local)
	}

	// 验证Redis配置
	if cfg.Type == StorageTypeRedis {
		v.validateRedisStorage(&cfg.Redis)
	}
}

// validateLocalStorage 验证本地存储配置。
func (v *Validator) validateLocalStorage(cfg *LocalConfig) {
	if cfg.MaxSize != "" {
		_, err := ParseSize(cfg.MaxSize)
		if err != nil {
			v.AddError("storage.local.max_size", fmt.Sprintf("invalid size format: %s", cfg.MaxSize))
		}
	}

	if cfg.NumCounters < 0 {
		v.AddError("storage.local.num_counters", "must be non-negative")
	}

	if cfg.CleanupInterval < 0 {
		v.AddError("storage.local.cleanup_interval", "must be non-negative")
	}
}

// validateRedisStorage 验证Redis存储配置。
func (v *Validator) validateRedisStorage(cfg *RedisConfig) {
	if cfg.Addr == "" {
		v.AddError("storage.redis.addr", "address is required when using redis storage")
	} else if !isValidRedisAddress(cfg.Addr) {
		v.AddError("storage.redis.addr", "invalid address format (expected host:port)")
	}

	if cfg.DB < 0 || cfg.DB > 15 {
		v.AddError("storage.redis.db", "must be between 0 and 15")
	}

	if cfg.PoolSize < 0 {
		v.AddError("storage.redis.pool_size", "must be non-negative")
	}

	if cfg.DialTimeout < 0 {
		v.AddError("storage.redis.dial_timeout", "must be non-negative")
	}
	if cfg.ReadTimeout < 0 {
		v.AddError("storage.redis.read_timeout", "must be non-negative")
	}
	if cfg.WriteTimeout < 0 {
		v.AddError("storage.redis.write_timeout", "must be non-negative")
	}
}

// validateLogging 验证日志配置。
func (v *Validator) validateLogging(cfg *LoggingConfig) {
	validLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLevels, cfg.Level) {
		v.AddError("logging.level", fmt.Sprintf("must be one of: %s", strings.Join(validLevels, ", ")))
	}

	validFormats := []string{"console", "json"}
	if !contains(validFormats, cfg.Format) {
		v.AddError("logging.format", fmt.Sprintf("must be one of: %s", strings.Join(validFormats, ", ")))
	}

	// 验证文件日志配置
	if cfg.File.Enabled {
		v.validateFileLogging(&cfg.File)
	}
}

// validateFileLogging 验证文件日志配置。
func (v *Validator) validateFileLogging(cfg *FileConfig) {
	if cfg.Path == "" {
		v.AddError("logging.file.path", "path is required when file logging is enabled")
	}

	if cfg.MaxSize < 0 {
		v.AddError("logging.file.max_size", "must be non-negative")
	}

	if cfg.MaxBackups < 0 {
		v.AddError("logging.file.max_backups", "must be non-negative")
	}

	if cfg.MaxAge < 0 {
		v.AddError("logging.file.max_age", "must be non-negative")
	}
}

// validateMetrics 验证指标配置。
func (v *Validator) validateMetrics(cfg *MetricsConfig) {
	if cfg.Enabled && cfg.Path == "" {
		v.AddError("metrics.path", "path is required when metrics is enabled")
	}

	if cfg.Path != "" && !strings.HasPrefix(cfg.Path, "/") {
		v.AddError("metrics.path", "must start with /")
	}
}

// validateRulesRef 验证规则引用配置。
func (v *Validator) validateRulesRef(cfg *RulesRef) {
	if cfg.File != "" {
		// 检查文件是否存在（可选，仅作为警告）
		if _, err := os.Stat(cfg.File); os.IsNotExist(err) {
			// 不添加错误，仅注明文件将在运行时检查
		}
	}

	if cfg.ReloadInterval < 0 {
		v.AddError("rules.reload_interval", "must be non-negative")
	}

	if cfg.ReloadInterval > 0 && cfg.ReloadInterval < 1*time.Second {
		v.AddError("rules.reload_interval", "must be at least 1 second if enabled")
	}
}

// ValidateRules 对规则配置进行全面验证。
func ValidateRules(rules *RulesConfig) ValidationErrors {
	v := NewValidator()

	// 验证元数据
	v.validateMeta(&rules.Meta)

	// 验证结果
	v.validateResults(rules.Results)

	// 验证访问规则
	v.validateAccessRules(&rules.Access, rules.Results, rules.Dicts)

	// 验证限流规则
	v.validateRateRules(&rules.Rules, rules.Results)

	return v.GetErrors()
}

// validateMeta 验证规则元数据。
func (v *Validator) validateMeta(meta *Meta) {
	if meta.Version == "" {
		v.AddError("meta.version", "version is required")
	} else if !isValidVersion(meta.Version) {
		v.AddError("meta.version", "invalid version format (expected semver like 1.0.0)")
	}
}

// validateResults 验证结果模板。
func (v *Validator) validateResults(results map[string]Result) {
	if len(results) == 0 {
		v.AddError("results", "at least one result must be defined")
		return
	}

	for name, result := range results {
		if result.Code < 0 {
			v.AddError(fmt.Sprintf("results.%s.code", name), "must be non-negative")
		}
		if result.Message == "" {
			v.AddError(fmt.Sprintf("results.%s.message", name), "message is required")
		}
	}
}

// validateAccessRules 验证访问控制规则。
func (v *Validator) validateAccessRules(access *AccessRules, results map[string]Result, dicts map[string]string) {
	for i, rule := range access.Whitelist {
		v.validateSingleAccessRule(fmt.Sprintf("access.whitelist[%d]", i), &rule, results, dicts)
	}

	for i, rule := range access.Blacklist {
		v.validateSingleAccessRule(fmt.Sprintf("access.blacklist[%d]", i), &rule, results, dicts)
	}
}

// validateSingleAccessRule 验证单个访问规则。
func (v *Validator) validateSingleAccessRule(prefix string, rule *AccessRule, results map[string]Result, dicts map[string]string) {
	if rule.Name == "" {
		v.AddError(prefix+".name", "name is required")
	}

	if len(rule.Match) == 0 {
		v.AddError(prefix+".match", "match conditions are required")
	}

	// 验证匹配条件中的字典引用
	for key, value := range rule.Match {
		if IsDictReference(value) {
			dictName := GetDictName(value)
			if _, ok := dicts[dictName]; !ok {
				v.AddError(fmt.Sprintf("%s.match.%s", prefix, key), fmt.Sprintf("references undefined dict '%s'", dictName))
			}
		}
	}

	// 验证结果引用
	if rule.Result == "" {
		v.AddError(prefix+".result", "result is required")
	} else if _, ok := results[rule.Result]; !ok {
		v.AddError(prefix+".result", fmt.Sprintf("references undefined result '%s'", rule.Result))
	}
}

// validateRateRules 验证限流规则。
func (v *Validator) validateRateRules(rules *RateRules, results map[string]Result) {
	for i, rule := range rules.Business {
		v.validateSingleRateRule(fmt.Sprintf("rules.business[%d]", i), &rule, results)
	}

	for i, rule := range rules.Post {
		v.validateSingleRateRule(fmt.Sprintf("rules.post[%d]", i), &rule, results)
	}

	for i, rule := range rules.Advanced {
		v.validateSingleRateRule(fmt.Sprintf("rules.advanced[%d]", i), &rule, results)
	}

	for i, rule := range rules.Default {
		v.validateSingleRateRule(fmt.Sprintf("rules.default[%d]", i), &rule, results)
	}
}

// validateSingleRateRule 验证单个限流规则。
func (v *Validator) validateSingleRateRule(prefix string, rule *RateRule, results map[string]Result) {
	if rule.Name == "" {
		v.AddError(prefix+".name", "name is required")
	}

	validTypes := []string{RuleTypeCount, RuleTypeFreq, RuleTypeAccumulate}
	if !contains(validTypes, rule.Type) {
		v.AddError(prefix+".type", fmt.Sprintf("must be one of: %s", strings.Join(validTypes, ", ")))
	}

	if len(rule.Match) == 0 {
		v.AddError(prefix+".match", "match conditions are required")
	}

	if rule.Limit.Time <= 0 {
		v.AddError(prefix+".limit.time", "must be positive")
	}

	if rule.Limit.Count <= 0 {
		v.AddError(prefix+".limit.count", "must be positive")
	}

	if rule.Result == "" {
		v.AddError(prefix+".result", "result is required")
	} else if _, ok := results[rule.Result]; !ok {
		v.AddError(prefix+".result", fmt.Sprintf("references undefined result '%s'", rule.Result))
	}
}

// 辅助函数

// isValidListenAddress 检查监听地址是否有效。
func isValidListenAddress(addr string) bool {
	// 允许仅端口（如":8080"）
	if strings.HasPrefix(addr, ":") {
		port := addr[1:]
		return isValidPort(port)
	}

	// 允许host:port格式
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	// 主机可为空以进行通配符绑定
	if host != "" {
		// 验证IP或主机名
		if ip := net.ParseIP(host); ip == nil {
			// 不是IP，检查是否为有效主机名
			if !isValidHostname(host) {
				return false
			}
		}
	}

	return isValidPort(port)
}

// isValidPort 检查端口字符串是否有效。
func isValidPort(port string) bool {
	if port == "" {
		return false
	}
	var p int
	_, err := fmt.Sscanf(port, "%d", &p)
	if err != nil {
		return false
	}
	return p > 0 && p <= 65535
}

// isValidHostname 检查主机名是否有效。
func isValidHostname(hostname string) bool {
	// 简单的主机名验证
	re := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)
	return re.MatchString(hostname)
}

// isValidRedisAddress 检查Redis地址是否有效。
func isValidRedisAddress(addr string) bool {
	_, _, err := net.SplitHostPort(addr)
	return err == nil
}

// isValidVersion 检查版本字符串是否符合semver格式。
func isValidVersion(version string) bool {
	re := regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`)
	return re.MatchString(version)
}

// contains 检查切片中是否包含指定字符串。
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
