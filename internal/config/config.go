// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package config 提供Koala限流器的配置加载功能。
//
// Koala是一个反作弊频率控制系统，提供限流和访问控制能力。
package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// 存储类型常量。
const (
	StorageTypeLocal = "local"
	StorageTypeRedis = "redis"
)

// sizeRegexp 用于匹配大小字符串的预编译正则表达式（避免每次调用重复编译）。
var sizeRegexp = regexp.MustCompile(`^(\d+)\s*(KB|MB|GB|TB|K|M|G|T)?$`)

// 默认配置值。
const (
	DefaultReadTimeout     = 5 * time.Second
	DefaultWriteTimeout    = 5 * time.Second
	DefaultShutdownTimeout = 30 * time.Second
	DefaultStorageType     = StorageTypeLocal
	DefaultLogLevel        = "info"
	DefaultLogFormat       = "console"
	DefaultMetricsPath     = "/metrics"
	DefaultReloadInterval  = 30 * time.Second
)

// Config 表示从koala.toml加载的主服务配置。
type Config struct {
	Server  ServerConfig  `toml:"server"`
	Rules   RulesRef      `toml:"rules"`
	Storage StorageConfig `toml:"storage"`
	Logging LoggingConfig `toml:"logging"`
	Metrics MetricsConfig `toml:"metrics"`
}

// ServerConfig 包含HTTP服务器设置。
type ServerConfig struct {
	Listen          string        `toml:"listen"`
	ReadTimeout     time.Duration `toml:"read_timeout"`
	WriteTimeout    time.Duration `toml:"write_timeout"`
	ShutdownTimeout time.Duration `toml:"shutdown_timeout"`
}

// RulesRef 包含规则配置文件的引用。
type RulesRef struct {
	File           string        `toml:"file"`
	ReloadInterval time.Duration `toml:"reload_interval"`
}

// StorageConfig 包含存储后端设置。
type StorageConfig struct {
	Type     string         `toml:"type"`
	Local    LocalConfig    `toml:"local"`
	Redis    RedisConfig    `toml:"redis"`
	Fallback FallbackConfig `toml:"fallback"`
}

// LocalConfig 包含本地（内存）存储设置。
type LocalConfig struct {
	MaxSize         string        `toml:"max_size"`
	NumCounters     int           `toml:"num_counters"`
	CleanupInterval time.Duration `toml:"cleanup_interval"`
}

// RedisConfig 包含Redis存储设置。
type RedisConfig struct {
	Addr         string        `toml:"addr"`
	Password     string        `toml:"password"`
	DB           int           `toml:"db"`
	PoolSize     int           `toml:"pool_size"`
	DialTimeout  time.Duration `toml:"dial_timeout"`
	ReadTimeout  time.Duration `toml:"read_timeout"`
	WriteTimeout time.Duration `toml:"write_timeout"`
}

// FallbackConfig 包含备用存储设置。
type FallbackConfig struct {
	Enabled bool `toml:"enabled"`
}

// LoggingConfig 包含日志设置。
type LoggingConfig struct {
	Level   string     `toml:"level"`
	Format  string     `toml:"format"`
	Console bool       `toml:"console"`
	File    FileConfig `toml:"file"`
}

// FileConfig 包含文件日志设置。
type FileConfig struct {
	Enabled    bool   `toml:"enabled"`
	Path       string `toml:"path"`
	MaxSize    int    `toml:"max_size"`
	MaxBackups int    `toml:"max_backups"`
	MaxAge     int    `toml:"max_age"`
	Compress   bool   `toml:"compress"`
}

// MetricsConfig 包含指标/监控设置。
type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

// LoadConfig 从TOML文件加载并验证服务配置。
func LoadConfig(path string) (*Config, error) {
	var cfg Config

	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file %s: %w", path, err)
	}

	// 应用默认值
	applyDefaults(&cfg)

	// 验证配置
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaults 为配置应用默认值。
func applyDefaults(cfg *Config) {
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = DefaultReadTimeout
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = DefaultWriteTimeout
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = DefaultShutdownTimeout
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = DefaultStorageType
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = DefaultLogLevel
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = DefaultLogFormat
	}
	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = DefaultMetricsPath
	}
	if cfg.Rules.ReloadInterval == 0 {
		cfg.Rules.ReloadInterval = DefaultReloadInterval
	}
}

// validateConfig 验证配置。
func validateConfig(cfg *Config) error {
	// 验证服务器配置
	if cfg.Server.Listen == "" {
		return fmt.Errorf("server listen address is required")
	}

	// 验证存储类型
	if cfg.Storage.Type != StorageTypeLocal && cfg.Storage.Type != StorageTypeRedis {
		return fmt.Errorf("invalid storage type: %s (must be 'local' or 'redis')", cfg.Storage.Type)
	}

	// 验证日志级别
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", cfg.Logging.Level)
	}

	// 验证日志格式
	validFormats := map[string]bool{
		"console": true,
		"json":    true,
	}
	if !validFormats[cfg.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (must be console or json)", cfg.Logging.Format)
	}

	return nil
}

// ParseSize 将大小字符串（如"512MB"、"1GB"）解析为字节数。
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// 使用预编译的包级别正则表达式匹配大小字符串
	matches := sizeRegexp.FindStringSubmatch(strings.ToUpper(s))

	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s", s)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value: %s", s)
	}

	if value < 0 {
		return 0, fmt.Errorf("negative size not allowed: %s", s)
	}

	unit := matches[2]
	multiplier := int64(1)

	switch unit {
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	return value * multiplier, nil
}

// GetMaxSizeBytes 返回本地存储的max_size字节数。
func (c *LocalConfig) GetMaxSizeBytes() (int64, error) {
	if c.MaxSize == "" {
		return 512 * 1024 * 1024, nil // 默认512MB
	}
	return ParseSize(c.MaxSize)
}
