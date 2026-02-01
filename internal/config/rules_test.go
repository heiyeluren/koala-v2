// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadRules tests loading a valid rules configuration file.
func TestLoadRules(t *testing.T) {
	tmpDir := t.TempDir()
	rulesPath := filepath.Join(tmpDir, "rules.toml")

	rulesContent := `
# Koala V2 规则配置

[meta]
version = "1.0.0"
description = "Test rules configuration"

[dicts]
uid_whitelist = "conf/uid_whitelist.txt"
uid_blacklist = "conf/uid_blacklist.txt"
ip_whitelist = "conf/ip_whitelist.txt"
ip_blacklist = "conf/ip_blacklist.txt"

[results]
allow = { code = 0, message = "Allow" }
deny = { code = 10, message = "Deny" }
auth_slider = { code = 20, message = "Auth Required", auth_type = 1 }
auth_captcha = { code = 21, message = "Captcha Required", auth_type = 2 }

[[access.whitelist]]
name = "uid_whitelist"
match = { uid = "@uid_whitelist" }
result = "allow"

[[access.whitelist]]
name = "ip_whitelist"
match = { ip = "@ip_whitelist" }
result = "allow"

[[access.blacklist]]
name = "uid_blacklist"
match = { uid = "@uid_blacklist" }
result = "deny"

[[access.blacklist]]
name = "ip_blacklist"
match = { ip = "@ip_blacklist" }
result = "deny"

[[rules.business]]
name = "login_rate_limit"
type = "count"
match = { act = "login", uid = "+" }
limit = { time = "1m", count = 10 }
result = "auth_slider"
desc = "Login rate limit 10 per minute"

[[rules.business]]
name = "submit_rate_limit"
type = "count"
match = { act = "submit", uid = "+" }
limit = { time = "1h", count = 100 }
result = "deny"
desc = "Submit rate limit 100 per hour"

[[rules.post]]
name = "comment_rate_limit"
type = "count"
match = { act = "comment", uid = "+" }
limit = { time = "10s", count = 5 }
result = "deny"
desc = "Comment rate limit"

[[rules.advanced]]
name = "ip_rate_limit"
type = "count"
match = { ip = "+" }
limit = { time = "1m", count = 1000 }
result = "deny"
desc = "IP rate limit"

[[rules.default]]
name = "global_default"
type = "count"
match = { act = "+", uid = "+" }
limit = { time = "1m", count = 60 }
result = "deny"
desc = "Default rate limit"
`

	err := os.WriteFile(rulesPath, []byte(rulesContent), 0644)
	require.NoError(t, err)

	rules, err := LoadRules(rulesPath)
	require.NoError(t, err)
	require.NotNil(t, rules)

	// Verify meta
	assert.Equal(t, "1.0.0", rules.Meta.Version)
	assert.Equal(t, "Test rules configuration", rules.Meta.Description)

	// Verify dicts
	assert.Equal(t, "conf/uid_whitelist.txt", rules.Dicts["uid_whitelist"])
	assert.Equal(t, "conf/uid_blacklist.txt", rules.Dicts["uid_blacklist"])
	assert.Equal(t, "conf/ip_whitelist.txt", rules.Dicts["ip_whitelist"])
	assert.Equal(t, "conf/ip_blacklist.txt", rules.Dicts["ip_blacklist"])

	// Verify results
	require.Contains(t, rules.Results, "allow")
	assert.Equal(t, 0, rules.Results["allow"].Code)
	assert.Equal(t, "Allow", rules.Results["allow"].Message)

	require.Contains(t, rules.Results, "deny")
	assert.Equal(t, 10, rules.Results["deny"].Code)
	assert.Equal(t, "Deny", rules.Results["deny"].Message)

	require.Contains(t, rules.Results, "auth_slider")
	assert.Equal(t, 20, rules.Results["auth_slider"].Code)
	assert.Equal(t, 1, rules.Results["auth_slider"].AuthType)

	// Verify access whitelist
	require.Len(t, rules.Access.Whitelist, 2)
	assert.Equal(t, "uid_whitelist", rules.Access.Whitelist[0].Name)
	assert.Equal(t, "@uid_whitelist", rules.Access.Whitelist[0].Match["uid"])
	assert.Equal(t, "allow", rules.Access.Whitelist[0].Result)

	// Verify access blacklist
	require.Len(t, rules.Access.Blacklist, 2)
	assert.Equal(t, "uid_blacklist", rules.Access.Blacklist[0].Name)
	assert.Equal(t, "deny", rules.Access.Blacklist[0].Result)

	// Verify business rules
	require.Len(t, rules.Rules.Business, 2)
	assert.Equal(t, "login_rate_limit", rules.Rules.Business[0].Name)
	assert.Equal(t, RuleTypeCount, rules.Rules.Business[0].Type)
	assert.Equal(t, "login", rules.Rules.Business[0].Match["act"])
	assert.Equal(t, "+", rules.Rules.Business[0].Match["uid"])
	assert.Equal(t, 1*time.Minute, rules.Rules.Business[0].Limit.Time)
	assert.Equal(t, 10, rules.Rules.Business[0].Limit.Count)
	assert.Equal(t, "auth_slider", rules.Rules.Business[0].Result)

	// Verify post rules
	require.Len(t, rules.Rules.Post, 1)
	assert.Equal(t, "comment_rate_limit", rules.Rules.Post[0].Name)

	// Verify advanced rules
	require.Len(t, rules.Rules.Advanced, 1)
	assert.Equal(t, "ip_rate_limit", rules.Rules.Advanced[0].Name)

	// Verify default rules
	require.Len(t, rules.Rules.Default, 1)
	assert.Equal(t, "global_default", rules.Rules.Default[0].Name)
}

// TestLoadRulesNotFound tests loading a non-existent rules file.
func TestLoadRulesNotFound(t *testing.T) {
	rules, err := LoadRules("/non/existent/path/rules.toml")
	assert.Error(t, err)
	assert.Nil(t, rules)
}

// TestLoadRulesInvalid tests loading an invalid TOML rules file.
func TestLoadRulesInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	rulesPath := filepath.Join(tmpDir, "invalid.toml")

	err := os.WriteFile(rulesPath, []byte("this is not valid toml [[["), 0644)
	require.NoError(t, err)

	rules, err := LoadRules(rulesPath)
	assert.Error(t, err)
	assert.Nil(t, rules)
}

// TestLoadRulesMinimal tests loading a minimal rules configuration.
func TestLoadRulesMinimal(t *testing.T) {
	tmpDir := t.TempDir()
	rulesPath := filepath.Join(tmpDir, "minimal.toml")

	rulesContent := `
[meta]
version = "1.0.0"

[results]
allow = { code = 0, message = "Allow" }
deny = { code = 10, message = "Deny" }
`

	err := os.WriteFile(rulesPath, []byte(rulesContent), 0644)
	require.NoError(t, err)

	rules, err := LoadRules(rulesPath)
	require.NoError(t, err)
	require.NotNil(t, rules)

	assert.Equal(t, "1.0.0", rules.Meta.Version)
	assert.Len(t, rules.Access.Whitelist, 0)
	assert.Len(t, rules.Access.Blacklist, 0)
	assert.Len(t, rules.Rules.Business, 0)
}

// TestRulesValidation tests rules validation.
func TestRulesValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rules",
			content: `
[meta]
version = "1.0.0"

[results]
allow = { code = 0, message = "Allow" }
deny = { code = 10, message = "Deny" }
`,
			wantErr: false,
		},
		{
			name: "missing version",
			content: `
[meta]
description = "test"

[results]
allow = { code = 0, message = "Allow" }
`,
			wantErr: true,
			errMsg:  "version",
		},
		{
			name: "invalid rule type",
			content: `
[meta]
version = "1.0.0"

[results]
allow = { code = 0, message = "Allow" }

[[rules.business]]
name = "test"
type = "invalid_type"
match = { act = "+" }
limit = { time = "1m", count = 10 }
result = "allow"
`,
			wantErr: true,
			errMsg:  "rule type",
		},
		{
			name: "rule references undefined result",
			content: `
[meta]
version = "1.0.0"

[results]
allow = { code = 0, message = "Allow" }

[[rules.business]]
name = "test"
type = "count"
match = { act = "+" }
limit = { time = "1m", count = 10 }
result = "undefined_result"
`,
			wantErr: true,
			errMsg:  "result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			rulesPath := filepath.Join(tmpDir, "test.toml")

			err := os.WriteFile(rulesPath, []byte(tt.content), 0644)
			require.NoError(t, err)

			rules, err := LoadRules(rulesPath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rules)
			}
		})
	}
}

// TestParseLimitTime tests parsing limit time strings.
func TestParseLimitTime(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1s", 1 * time.Second, false},
		{"10s", 10 * time.Second, false},
		{"1m", 1 * time.Minute, false},
		{"30m", 30 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseLimitTime(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestRuleTypeConstants tests rule type constants.
func TestRuleTypeConstants(t *testing.T) {
	assert.Equal(t, "count", RuleTypeCount)
	assert.Equal(t, "freq", RuleTypeFreq)
	assert.Equal(t, "accumulate", RuleTypeAccumulate)
}

// TestAccessRuleMatch tests access rule match patterns.
func TestAccessRuleMatch(t *testing.T) {
	rule := AccessRule{
		Name:   "test",
		Match:  map[string]string{"uid": "@uid_whitelist", "ip": "+"},
		Result: "allow",
	}

	// Verify dict reference detection
	assert.True(t, IsDictReference(rule.Match["uid"]))
	assert.False(t, IsDictReference(rule.Match["ip"]))

	// Get dict name from reference
	dictName := GetDictName(rule.Match["uid"])
	assert.Equal(t, "uid_whitelist", dictName)
}

// TestRateRuleGetLimit tests rate rule limit retrieval.
func TestRateRuleGetLimit(t *testing.T) {
	rule := RateRule{
		Name: "test",
		Type: RuleTypeCount,
		Limit: Limit{
			Time:  1 * time.Minute,
			Count: 100,
		},
	}

	assert.Equal(t, 1*time.Minute, rule.Limit.Time)
	assert.Equal(t, 100, rule.Limit.Count)
}
