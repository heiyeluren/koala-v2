// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package integration

import (
	"testing"

	"koala/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadRealConfig tests loading the actual configuration files.
func TestLoadRealConfig(t *testing.T) {
	// Test loading real config
	cfg, err := config.LoadConfig("../../conf/koala.toml")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, ":9981", cfg.Server.Listen)
	assert.Equal(t, "local", cfg.Storage.Type)
	assert.Equal(t, "info", cfg.Logging.Level)
}

// TestLoadRealRules tests loading the actual rules files.
func TestLoadRealRules(t *testing.T) {
	rules, err := config.LoadRules("../../conf/rules.toml")
	require.NoError(t, err)
	require.NotNil(t, rules)

	assert.Equal(t, "1.0.0", rules.Meta.Version)
	assert.Contains(t, rules.Results, "allow")
	assert.Contains(t, rules.Results, "deny")
}

// TestLoadRealDicts tests loading the actual dictionary files.
func TestLoadRealDicts(t *testing.T) {
	rules, err := config.LoadRules("../../conf/rules.toml")
	require.NoError(t, err)

	// Adjust paths for test directory
	adjustedDicts := make(map[string]string)
	for name, path := range rules.Dicts {
		adjustedDicts[name] = "../../" + path
	}

	dicts, err := config.LoadDicts(adjustedDicts)
	require.NoError(t, err)
	require.NotNil(t, dicts)

	// Check that dictionaries were loaded
	ipWhitelist, ok := dicts.Get("ip_whitelist")
	require.True(t, ok)
	assert.True(t, ipWhitelist.Contains("127.0.0.1"))
}
