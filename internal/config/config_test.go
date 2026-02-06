// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package config provides configuration loading for Koala rate limiter.
package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig tests loading a valid configuration file.
func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "koala.toml")

	configContent := `
# Koala V2 服务配置

[server]
listen = ":9981"
read_timeout = "5s"
write_timeout = "10s"
shutdown_timeout = "30s"

[rules]
file = "conf/rules.toml"
reload_interval = "30s"

[storage]
type = "local"

[storage.local]
max_size = "512MB"
num_counters = 1000000
cleanup_interval = "1m"

[storage.redis]
addr = "localhost:6379"
password = ""
db = 0
pool_size = 100
dial_timeout = "5s"
read_timeout = "3s"
write_timeout = "3s"

[storage.fallback]
enabled = true

[logging]
level = "info"
format = "json"
console = true

[logging.file]
enabled = true
path = "logs/koala.log"
max_size = 100
max_backups = 5
max_age = 7
compress = true

[metrics]
enabled = true
path = "/metrics"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load configuration
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify server configuration
	assert.Equal(t, ":9981", cfg.Server.Listen)
	assert.Equal(t, 5*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.ShutdownTimeout)

	// Verify rules configuration
	assert.Equal(t, "conf/rules.toml", cfg.Rules.File)
	assert.Equal(t, 30*time.Second, cfg.Rules.ReloadInterval)

	// Verify storage configuration
	assert.Equal(t, "local", cfg.Storage.Type)
	assert.Equal(t, "512MB", cfg.Storage.Local.MaxSize)
	assert.Equal(t, 1000000, cfg.Storage.Local.NumCounters)
	assert.Equal(t, 1*time.Minute, cfg.Storage.Local.CleanupInterval)

	// Verify Redis configuration
	assert.Equal(t, "localhost:6379", cfg.Storage.Redis.Addr)
	assert.Equal(t, "", cfg.Storage.Redis.Password)
	assert.Equal(t, 0, cfg.Storage.Redis.DB)
	assert.Equal(t, 100, cfg.Storage.Redis.PoolSize)
	assert.Equal(t, 5*time.Second, cfg.Storage.Redis.DialTimeout)
	assert.Equal(t, 3*time.Second, cfg.Storage.Redis.ReadTimeout)
	assert.Equal(t, 3*time.Second, cfg.Storage.Redis.WriteTimeout)

	// Verify fallback configuration
	assert.True(t, cfg.Storage.Fallback.Enabled)

	// Verify logging configuration
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.True(t, cfg.Logging.Console)
	assert.True(t, cfg.Logging.File.Enabled)
	assert.Equal(t, "logs/koala.log", cfg.Logging.File.Path)
	assert.Equal(t, 100, cfg.Logging.File.MaxSize)
	assert.Equal(t, 5, cfg.Logging.File.MaxBackups)
	assert.Equal(t, 7, cfg.Logging.File.MaxAge)
	assert.True(t, cfg.Logging.File.Compress)

	// Verify metrics configuration
	assert.True(t, cfg.Metrics.Enabled)
	assert.Equal(t, "/metrics", cfg.Metrics.Path)
}

// TestLoadConfigNotFound tests loading a non-existent configuration file.
func TestLoadConfigNotFound(t *testing.T) {
	cfg, err := LoadConfig("/non/existent/path/koala.toml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TestLoadConfigInvalid tests loading an invalid TOML configuration file.
func TestLoadConfigInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	// Write invalid TOML content
	err := os.WriteFile(configPath, []byte("this is not valid toml [[["), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TestLoadConfigDefaults tests that default values are applied.
func TestLoadConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "minimal.toml")

	// Minimal configuration
	configContent := `
[server]
listen = ":8080"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify defaults are applied
	assert.Equal(t, ":8080", cfg.Server.Listen)
	assert.Equal(t, DefaultReadTimeout, cfg.Server.ReadTimeout)
	assert.Equal(t, DefaultWriteTimeout, cfg.Server.WriteTimeout)
	assert.Equal(t, DefaultShutdownTimeout, cfg.Server.ShutdownTimeout)
	assert.Equal(t, DefaultStorageType, cfg.Storage.Type)
	assert.Equal(t, DefaultLogLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultLogFormat, cfg.Logging.Format)
}

// TestConfigValidation tests configuration validation.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: `
[server]
listen = ":9981"
`,
			wantErr: false,
		},
		{
			name: "empty listen address",
			config: `
[server]
listen = ""
`,
			wantErr: true,
			errMsg:  "listen address",
		},
		{
			name: "invalid storage type",
			config: `
[server]
listen = ":9981"
[storage]
type = "invalid"
`,
			wantErr: true,
			errMsg:  "storage type",
		},
		{
			name: "invalid log level",
			config: `
[server]
listen = ":9981"
[logging]
level = "invalid"
`,
			wantErr: true,
			errMsg:  "log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test.toml")

			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			cfg, err := LoadConfig(configPath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

// TestParseSize tests parsing size strings like "512MB".
func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"512MB", 512 * 1024 * 1024, false},
		{"1GB", 1 * 1024 * 1024 * 1024, false},
		{"100KB", 100 * 1024, false},
		{"1024", 1024, false},
		{"invalid", 0, true},
		{"-100MB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestStorageTypeConstants tests storage type constants.
func TestStorageTypeConstants(t *testing.T) {
	assert.Equal(t, "local", StorageTypeLocal)
	assert.Equal(t, "redis", StorageTypeRedis)
}

// TestValidateConfigComprehensive tests comprehensive config validation.
func TestValidateConfigComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		wantErrors bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Listen:          ":8080",
					ReadTimeout:     5 * time.Second,
					WriteTimeout:    5 * time.Second,
					ShutdownTimeout: 30 * time.Second,
				},
				Storage: StorageConfig{
					Type: "local",
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErrors: false,
		},
		{
			name: "invalid listen address",
			config: &Config{
				Server: ServerConfig{
					Listen: "invalid",
				},
				Storage: StorageConfig{Type: "local"},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantErrors: true,
		},
		{
			name: "negative timeout",
			config: &Config{
				Server: ServerConfig{
					Listen:      ":8080",
					ReadTimeout: -1 * time.Second,
				},
				Storage: StorageConfig{Type: "local"},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantErrors: true,
		},
		{
			name: "invalid redis config",
			config: &Config{
				Server: ServerConfig{Listen: ":8080"},
				Storage: StorageConfig{
					Type: "redis",
					Redis: RedisConfig{
						Addr: "", // Empty address
						DB:   20, // Invalid DB
					},
				},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateConfig(tt.config)
			if tt.wantErrors {
				assert.True(t, errors.HasErrors())
			} else {
				assert.False(t, errors.HasErrors())
			}
		})
	}
}

// TestValidationError tests the ValidationError type.
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "server.listen",
		Message: "is required",
	}

	assert.Contains(t, err.Error(), "server.listen")
	assert.Contains(t, err.Error(), "is required")
}

// TestValidationErrors tests the ValidationErrors collection.
func TestValidationErrors(t *testing.T) {
	var errors ValidationErrors

	// Empty errors
	assert.False(t, errors.HasErrors())
	assert.Empty(t, errors.Error())

	// Add errors
	errors = append(errors, ValidationError{Field: "field1", Message: "error1"})
	errors = append(errors, ValidationError{Field: "field2", Message: "error2"})

	assert.True(t, errors.HasErrors())
	assert.Contains(t, errors.Error(), "field1")
	assert.Contains(t, errors.Error(), "field2")
}

// TestWatcher tests the file watcher.
func TestWatcher(t *testing.T) {
	watcher, err := NewWatcher(DefaultWatcherConfig())
	require.NoError(t, err)
	require.NotNil(t, watcher)

	defer watcher.Stop()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	err = os.WriteFile(testFile, []byte("initial"), 0644)
	require.NoError(t, err)

	// Track callback invocations
	callbackCalled := make(chan string, 1)
	callback := func(path string) error {
		callbackCalled <- path
		return nil
	}

	// Watch the file
	err = watcher.Watch(testFile, callback)
	require.NoError(t, err)

	// Start the watcher
	watcher.Start()

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified"), 0644)
	require.NoError(t, err)

	// Wait for callback
	select {
	case path := <-callbackCalled:
		absPath, _ := filepath.Abs(testFile)
		assert.Equal(t, absPath, path)
	case <-time.After(2 * time.Second):
		t.Fatal("callback not called within timeout")
	}
}

// TestWatcherUnwatch tests removing a watch.
func TestWatcherUnwatch(t *testing.T) {
	watcher, err := NewWatcher(DefaultWatcherConfig())
	require.NoError(t, err)
	defer watcher.Stop()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err = os.WriteFile(testFile, []byte("initial"), 0644)
	require.NoError(t, err)

	callback := func(path string) error { return nil }

	err = watcher.Watch(testFile, callback)
	require.NoError(t, err)

	files := watcher.WatchedFiles()
	assert.Len(t, files, 1)

	err = watcher.Unwatch(testFile)
	require.NoError(t, err)

	files = watcher.WatchedFiles()
	assert.Len(t, files, 0)
}

// TestDefaultWatcherConfig tests default watcher configuration.
func TestDefaultWatcherConfig(t *testing.T) {
	cfg := DefaultWatcherConfig()
	assert.Equal(t, 100*time.Millisecond, cfg.Debounce)
}

// TestConfigChangeType tests the ConfigChangeType string method.
func TestConfigChangeType(t *testing.T) {
	assert.Equal(t, "config", ConfigChangeTypeConfig.String())
	assert.Equal(t, "rules", ConfigChangeTypeRules.String())
	assert.Equal(t, "dict", ConfigChangeTypeDict.String())
	assert.Equal(t, "unknown", ConfigChangeType(99).String())
}

// TestParseSize_Concurrent 并发调用 ParseSize，验证包级别正则表达式在多 goroutine 下无竞态条件。
func TestParseSize_Concurrent(t *testing.T) {
	// 测试用例：覆盖多种合法大小格式
	inputs := []struct {
		input    string
		expected int64
	}{
		{"512MB", 512 * 1024 * 1024},
		{"1GB", 1 * 1024 * 1024 * 1024},
		{"100KB", 100 * 1024},
		{"1024", 1024},
		{"2TB", 2 * 1024 * 1024 * 1024 * 1024},
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 启动 100 个 goroutine 并发调用 ParseSize
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			tc := inputs[idx%len(inputs)]
			result, err := ParseSize(tc.input)
			assert.NoError(t, err, "goroutine %d: ParseSize(%q) 返回错误", idx, tc.input)
			assert.Equal(t, tc.expected, result, "goroutine %d: ParseSize(%q) 返回值不匹配", idx, tc.input)
		}(i)
	}

	wg.Wait()
}
