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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"koala/internal/config"
	"koala/internal/engine/matcher"
	"koala/internal/storage"
)

// mockStorage 实现storage.Storage接口用于测试。
// 使用sync.RWMutex确保并发安全。
type mockStorage struct {
	mu      sync.RWMutex
	data    map[string]int64
	strings map[string]string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:    make(map[string]int64),
		strings: make(map[string]string),
	}
}

func (m *mockStorage) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.strings[key]; ok {
		return v, nil
	}
	return "", storage.ErrKeyNotFound
}

func (m *mockStorage) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strings[key] = value
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.strings, key)
	delete(m.data, key)
	return nil
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok1 := m.strings[key]
	_, ok2 := m.data[key]
	return ok1 || ok2, nil
}

func (m *mockStorage) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (m *mockStorage) GetInt(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return 0, storage.ErrKeyNotFound
}

func (m *mockStorage) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockStorage) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key]++
	return m.data[key], nil
}

func (m *mockStorage) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] += delta
	return m.data[key], nil
}

func (m *mockStorage) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key]++
	return m.data[key], nil
}

func (m *mockStorage) LPush(ctx context.Context, key string, values ...int64) error {
	return nil
}

func (m *mockStorage) LLen(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockStorage) LIndex(ctx context.Context, key string, index int64) (int64, error) {
	return 0, nil
}

func (m *mockStorage) LTrim(ctx context.Context, key string, start, stop int64) error {
	return nil
}

func (m *mockStorage) LRange(ctx context.Context, key string, start, stop int64) ([]int64, error) {
	return nil, nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func (m *mockStorage) Type() string {
	return "mock"
}

// ===========================================================================
// 测试 Request 类型
// ===========================================================================

func TestRequest_GetField(t *testing.T) {
	t.Run("标准字段", func(t *testing.T) {
		req := &Request{
			Act: "login",
			UID: "user123",
			IP:  "192.168.1.1",
			DID: "device456",
		}

		assert.Equal(t, "login", req.GetField("act"))
		assert.Equal(t, "user123", req.GetField("uid"))
		assert.Equal(t, "192.168.1.1", req.GetField("ip"))
		assert.Equal(t, "device456", req.GetField("did"))
	})

	t.Run("扩展字段", func(t *testing.T) {
		req := &Request{
			Act: "post",
			Ext: map[string]string{
				"channel": "web",
				"version": "2.0",
			},
		}

		assert.Equal(t, "web", req.GetField("channel"))
		assert.Equal(t, "2.0", req.GetField("version"))
	})

	t.Run("不存在的字段返回空字符串", func(t *testing.T) {
		req := &Request{Act: "test"}
		assert.Equal(t, "", req.GetField("unknown"))
	})

	t.Run("空扩展字段", func(t *testing.T) {
		req := &Request{Act: "test"}
		assert.Equal(t, "", req.GetField("channel"))
	})
}

// ===========================================================================
// 测试 Response 类型
// ===========================================================================

func TestNewAllowedResponse(t *testing.T) {
	resp := NewAllowedResponse()

	assert.True(t, resp.Allowed)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Message)
	assert.Equal(t, "", resp.RuleName)
	assert.Equal(t, 0, resp.AuthType)
}

func TestNewDeniedResponse(t *testing.T) {
	resp := NewDeniedResponse(1001, "请求过于频繁", "rate_limit_login", 1)

	assert.False(t, resp.Allowed)
	assert.Equal(t, 1001, resp.Code)
	assert.Equal(t, "请求过于频繁", resp.Message)
	assert.Equal(t, "rate_limit_login", resp.RuleName)
	assert.Equal(t, 1, resp.AuthType)
}

// ===========================================================================
// 测试 Rule 类型
// ===========================================================================

func TestRule_Matches(t *testing.T) {
	t.Run("精确匹配", func(t *testing.T) {
		rule := &Rule{
			Name: "test_rule",
			Match: map[string]string{
				"act": "login",
				"uid": "user123",
			},
			Matchers: make(map[string]matcher.Matcher),
		}

		// 匹配的请求
		req1 := &Request{Act: "login", UID: "user123"}
		assert.True(t, rule.Matches(req1))

		// 不匹配的请求
		req2 := &Request{Act: "login", UID: "user456"}
		assert.False(t, rule.Matches(req2))

		req3 := &Request{Act: "register", UID: "user123"}
		assert.False(t, rule.Matches(req3))
	})

	t.Run("通配符匹配", func(t *testing.T) {
		rule := &Rule{
			Name: "test_wildcard",
			Match: map[string]string{
				"act": "login",
				"uid": "+",
			},
			Matchers: make(map[string]matcher.Matcher),
		}

		req1 := &Request{Act: "login", UID: "any_user"}
		assert.True(t, rule.Matches(req1))

		req2 := &Request{Act: "login", UID: ""}
		assert.False(t, rule.Matches(req2)) // 通配符不匹配空值
	})

	t.Run("多值匹配", func(t *testing.T) {
		rule := &Rule{
			Name: "test_multi",
			Match: map[string]string{
				"act": "login,register,reset",
			},
			Matchers: make(map[string]matcher.Matcher),
		}

		assert.True(t, rule.Matches(&Request{Act: "login"}))
		assert.True(t, rule.Matches(&Request{Act: "register"}))
		assert.True(t, rule.Matches(&Request{Act: "reset"}))
		assert.False(t, rule.Matches(&Request{Act: "logout"}))
	})
}

func TestRule_GenerateKey(t *testing.T) {
	rule := &Rule{
		Name: "login_limit",
		Match: map[string]string{
			"act": "login",
			"uid": "+",
		},
	}

	req := &Request{Act: "login", UID: "user123"}
	key := rule.GenerateKey(req)

	// 键应包含规则名和匹配字段的值
	assert.Contains(t, key, "koala:")
	assert.Contains(t, key, "login_limit")
	assert.Contains(t, key, "act=login")
	assert.Contains(t, key, "uid=user123")
}

// ===========================================================================
// 测试 RuleSet 构建
// ===========================================================================

func TestBuildRuleSet(t *testing.T) {
	t.Run("构建完整规则集", func(t *testing.T) {
		rulesConfig := &config.RulesConfig{
			Meta: config.Meta{
				Version:     "1.0",
				Description: "Test rules",
			},
			Results: map[string]config.Result{
				"allow":      {Code: 0, Message: "ok", AuthType: 0},
				"deny":       {Code: 1001, Message: "denied", AuthType: 0},
				"rate_limit": {Code: 1002, Message: "rate limited", AuthType: 1},
			},
			Access: config.AccessRules{
				Whitelist: []config.AccessRule{
					{
						Name:   "vip_whitelist",
						Match:  map[string]string{"uid": "vip_user"},
						Result: "allow",
					},
				},
				Blacklist: []config.AccessRule{
					{
						Name:   "bad_ip_blacklist",
						Match:  map[string]string{"ip": "10.0.0.1"},
						Result: "deny",
					},
				},
			},
			Rules: config.RateRules{
				Business: []config.RateRule{
					{
						Name:   "login_limit",
						Type:   config.RuleTypeCount,
						Match:  map[string]string{"act": "login", "uid": "+"},
						Limit:  config.Limit{Time: time.Minute, Count: 5},
						Result: "rate_limit",
					},
				},
				Default: []config.RateRule{
					{
						Name:   "default_limit",
						Type:   config.RuleTypeCount,
						Match:  map[string]string{"act": "+"},
						Limit:  config.Limit{Time: time.Minute, Count: 100},
						Result: "rate_limit",
					},
				},
			},
		}

		ruleSet, err := BuildRuleSet(rulesConfig)
		require.NoError(t, err)
		require.NotNil(t, ruleSet)

		assert.Len(t, ruleSet.Whitelist, 1)
		assert.Len(t, ruleSet.Blacklist, 1)
		assert.Len(t, ruleSet.Business, 1)
		assert.Len(t, ruleSet.Default, 1)

		// 验证白名单规则
		whiteRule := ruleSet.Whitelist[0]
		assert.Equal(t, "vip_whitelist", whiteRule.Name)
		assert.Equal(t, RuleTypeAccess, whiteRule.Type)
		assert.Equal(t, PhaseWhitelist, whiteRule.Phase)
	})

	t.Run("结果不存在时返回错误", func(t *testing.T) {
		rulesConfig := &config.RulesConfig{
			Meta:    config.Meta{Version: "1.0"},
			Results: map[string]config.Result{},
			Access: config.AccessRules{
				Whitelist: []config.AccessRule{
					{
						Name:   "test",
						Match:  map[string]string{"uid": "123"},
						Result: "nonexistent",
					},
				},
			},
		}

		_, err := BuildRuleSet(rulesConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// ===========================================================================
// 测试 Engine
// ===========================================================================

func TestEngine_New(t *testing.T) {
	t.Run("创建空引擎", func(t *testing.T) {
		engine := New()
		require.NotNil(t, engine)

		ruleSet := engine.GetRuleSet()
		require.NotNil(t, ruleSet)
		assert.Empty(t, ruleSet.Whitelist)
	})

	t.Run("使用选项创建引擎", func(t *testing.T) {
		store := newMockStorage()
		dicts := config.NewDictManager()

		engine := New(
			WithStorage(store),
			WithDicts(dicts),
		)

		require.NotNil(t, engine)
		assert.NotNil(t, engine.storage)
		assert.NotNil(t, engine.dicts)
	})
}

func TestEngine_LoadRules(t *testing.T) {
	engine := New()

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "ok"},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{Name: "test", Match: map[string]string{"uid": "123"}, Result: "allow"},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ruleSet := engine.GetRuleSet()
	assert.Len(t, ruleSet.Whitelist, 1)
}

func TestEngine_Check_Whitelist(t *testing.T) {
	engine := New()

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "VIP用户允许", AuthType: 0},
			"deny":  {Code: 1001, Message: "拒绝访问", AuthType: 0},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{
					Name:   "vip_user",
					Match:  map[string]string{"uid": "vip123"},
					Result: "allow",
				},
			},
			Blacklist: []config.AccessRule{
				{
					Name:   "bad_user",
					Match:  map[string]string{"uid": "bad123"},
					Result: "deny",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// VIP用户应该直接允许
	resp, err := engine.Check(ctx, &Request{UID: "vip123"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.Equal(t, "vip_user", resp.RuleName)
}

func TestEngine_Check_Blacklist(t *testing.T) {
	engine := New()

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"deny": {Code: 1001, Message: "拒绝访问", AuthType: 0},
		},
		Access: config.AccessRules{
			Blacklist: []config.AccessRule{
				{
					Name:   "bad_ip",
					Match:  map[string]string{"ip": "10.0.0.1"},
					Result: "deny",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// 黑名单IP应该直接拒绝
	resp, err := engine.Check(ctx, &Request{IP: "10.0.0.1"})
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, "bad_ip", resp.RuleName)
	assert.Equal(t, 1001, resp.Code)
}

func TestEngine_Check_WhitelistPriority(t *testing.T) {
	engine := New()

	// 同时在白名单和黑名单中的用户，白名单优先
	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "允许"},
			"deny":  {Code: 1001, Message: "拒绝"},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{Name: "vip", Match: map[string]string{"uid": "user123"}, Result: "allow"},
			},
			Blacklist: []config.AccessRule{
				{Name: "bad", Match: map[string]string{"uid": "user123"}, Result: "deny"},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	resp, err := engine.Check(ctx, &Request{UID: "user123"})
	require.NoError(t, err)

	// 白名单应该优先于黑名单
	assert.True(t, resp.Allowed)
	assert.Equal(t, "vip", resp.RuleName)
}

func TestEngine_Check_RateLimit(t *testing.T) {
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"rate_limit": {Code: 1002, Message: "请求过于频繁", AuthType: 1},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "login_limit",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "login", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 3},
					Result: "rate_limit",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	req := &Request{Act: "login", UID: "user123"}

	// 前3次请求应该允许
	for i := 0; i < 3; i++ {
		resp, err := engine.Check(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Allowed, "第%d次请求应该允许", i+1)
	}

	// 第4次请求应该被限流
	resp, err := engine.Check(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, "login_limit", resp.RuleName)
	assert.Equal(t, 1002, resp.Code)
	assert.Equal(t, 1, resp.AuthType)
}

func TestEngine_Check_NoMatchingRule(t *testing.T) {
	engine := New()

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"rate_limit": {Code: 1002, Message: "限流"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "login_limit",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "login"},
					Limit:  config.Limit{Time: time.Minute, Count: 5},
					Result: "rate_limit",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// 不匹配任何规则的请求应该允许
	resp, err := engine.Check(ctx, &Request{Act: "logout"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
}

func TestEngine_Check_NilRequest(t *testing.T) {
	engine := New()

	ctx := context.Background()
	_, err := engine.Check(ctx, nil)
	assert.Error(t, err)
}

func TestEngine_Check_EmptyRuleSet(t *testing.T) {
	engine := New()

	ctx := context.Background()
	resp, err := engine.Check(ctx, &Request{Act: "test"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
}

func TestEngine_Check_RulePhasePriority(t *testing.T) {
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"business": {Code: 1001, Message: "业务限流"},
			"default":  {Code: 1002, Message: "默认限流"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "business_limit",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "post"},
					Limit:  config.Limit{Time: time.Minute, Count: 2},
					Result: "business",
				},
			},
			Default: []config.RateRule{
				{
					Name:   "default_limit",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 10},
					Result: "default",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	req := &Request{Act: "post", UID: "user1"}

	// 前2次请求应该允许（业务规则限制为2）
	for i := 0; i < 2; i++ {
		resp, err := engine.Check(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Allowed)
	}

	// 第3次应该被业务规则限流（不是默认规则）
	resp, err := engine.Check(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, "business_limit", resp.RuleName)
	assert.Equal(t, 1001, resp.Code)
}

func TestEngine_Browse(t *testing.T) {
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"rate_limit": {Code: 1002, Message: "限流"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "test_limit",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "test"},
					Limit:  config.Limit{Time: time.Minute, Count: 3},
					Result: "rate_limit",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	req := &Request{Act: "test", UID: "user1"}

	// Browse不应该更新计数器
	for i := 0; i < 5; i++ {
		resp, err := engine.Browse(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Allowed, "Browse不应该触发限流，因为不更新计数器")
	}

	// 但Check应该更新计数器
	for i := 0; i < 3; i++ {
		_, err := engine.Check(ctx, req)
		require.NoError(t, err)
	}

	// 现在Browse应该显示已达到限制
	resp, err := engine.Browse(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
}

func TestEngine_HotReload(t *testing.T) {
	engine := New()

	// 初始规则
	rulesV1 := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "v1允许"},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{Name: "v1_rule", Match: map[string]string{"uid": "user1"}, Result: "allow"},
			},
		},
	}

	err := engine.LoadRules(rulesV1)
	require.NoError(t, err)

	ctx := context.Background()

	// 验证v1规则生效
	resp, err := engine.Check(ctx, &Request{UID: "user1"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.Equal(t, "v1_rule", resp.RuleName)

	// 热加载新规则
	rulesV2 := &config.RulesConfig{
		Meta: config.Meta{Version: "2.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "v2允许"},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{Name: "v2_rule", Match: map[string]string{"uid": "user2"}, Result: "allow"},
			},
		},
	}

	err = engine.LoadRules(rulesV2)
	require.NoError(t, err)

	// 验证v2规则生效
	resp, err = engine.Check(ctx, &Request{UID: "user2"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.Equal(t, "v2_rule", resp.RuleName)

	// 旧规则不再生效
	resp, err = engine.Check(ctx, &Request{UID: "user1"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed) // 允许（因为没有规则匹配）
	assert.Equal(t, "", resp.RuleName)
}

// ===========================================================================
// 测试不同的匹配模式
// ===========================================================================

func TestEngine_Check_MultiValueMatch(t *testing.T) {
	engine := New()

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"deny": {Code: 1001, Message: "禁止访问"},
		},
		Access: config.AccessRules{
			Blacklist: []config.AccessRule{
				{
					Name:   "multi_act_blacklist",
					Match:  map[string]string{"act": "hack,attack,exploit"},
					Result: "deny",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// 这些行为应该被拒绝
	for _, act := range []string{"hack", "attack", "exploit"} {
		resp, err := engine.Check(ctx, &Request{Act: act})
		require.NoError(t, err)
		assert.False(t, resp.Allowed, "行为 %s 应该被拒绝", act)
	}

	// 正常行为应该允许
	resp, err := engine.Check(ctx, &Request{Act: "login"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)
}

func TestEngine_Check_NotMatch(t *testing.T) {
	engine := New()

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"deny": {Code: 1001, Message: "非VIP禁止"},
		},
		Access: config.AccessRules{
			Blacklist: []config.AccessRule{
				{
					Name:   "non_vip_block",
					Match:  map[string]string{"act": "premium", "uid": "!vip_user"},
					Result: "deny",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// VIP用户可以访问premium
	resp, err := engine.Check(ctx, &Request{Act: "premium", UID: "vip_user"})
	require.NoError(t, err)
	assert.True(t, resp.Allowed)

	// 非VIP用户不能访问premium
	resp, err = engine.Check(ctx, &Request{Act: "premium", UID: "normal_user"})
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
}

// ===========================================================================
// 测试不同的算法类型
// ===========================================================================

func TestEngine_Check_FreqAlgorithm(t *testing.T) {
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"rate_limit": {Code: 1002, Message: "频率限制"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "freq_limit",
					Type:   config.RuleTypeFreq,
					Match:  map[string]string{"act": "api"},
					Limit:  config.Limit{Time: time.Minute, Count: 5},
					Result: "rate_limit",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	// 验证规则被正确加载
	ruleSet := engine.GetRuleSet()
	require.Len(t, ruleSet.Business, 1)
	assert.Equal(t, "leak", ruleSet.Business[0].Algorithm.Name())
}

func TestEngine_Check_AccumulateAlgorithm(t *testing.T) {
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"rate_limit": {Code: 1002, Message: "累积限制"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "accumulate_limit",
					Type:   config.RuleTypeAccumulate,
					Match:  map[string]string{"act": "download"},
					Limit:  config.Limit{Time: time.Hour, Count: 10},
					Result: "rate_limit",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	// 验证规则被正确加载
	ruleSet := engine.GetRuleSet()
	require.Len(t, ruleSet.Business, 1)
	assert.Equal(t, "base", ruleSet.Business[0].Algorithm.Name())
}

// ===========================================================================
// 并发测试
// ===========================================================================

func TestEngine_ConcurrentCheck(t *testing.T) {
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"rate_limit": {Code: 1002, Message: "限流"},
		},
		Rules: config.RateRules{
			Default: []config.RateRule{
				{
					Name:   "concurrent_limit",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 1000},
					Result: "rate_limit",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()

	// 并发执行100个请求
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			req := &Request{Act: "test", UID: "user" + string(rune(id))}
			_, err := engine.Check(ctx, req)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestEngine_ConcurrentHotReload(t *testing.T) {
	engine := New()

	rulesV1 := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "v1"},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{Name: "v1", Match: map[string]string{"uid": "+"}, Result: "allow"},
			},
		},
	}

	rulesV2 := &config.RulesConfig{
		Meta: config.Meta{Version: "2.0"},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "v2"},
		},
		Access: config.AccessRules{
			Whitelist: []config.AccessRule{
				{Name: "v2", Match: map[string]string{"uid": "+"}, Result: "allow"},
			},
		},
	}

	err := engine.LoadRules(rulesV1)
	require.NoError(t, err)

	ctx := context.Background()

	// 并发执行检查和热加载
	done := make(chan bool, 110)

	// 100个goroutine执行检查
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				resp, err := engine.Check(ctx, &Request{UID: "test"})
				assert.NoError(t, err)
				assert.True(t, resp.Allowed)
			}
			done <- true
		}()
	}

	// 10个goroutine执行热加载
	for i := 0; i < 10; i++ {
		go func(id int) {
			if id%2 == 0 {
				engine.LoadRules(rulesV1)
			} else {
				engine.LoadRules(rulesV2)
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 110; i++ {
		<-done
	}
}

// ===========================================================================
// 测试规则优先级
// ===========================================================================

func TestEngine_Check_AllPhasePriority(t *testing.T) {
	// 测试所有四个阶段的规则优先级: Business > Post > Advanced > Default
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"business_result": {Code: 1001, Message: "业务规则限流"},
			"post_result":     {Code: 1002, Message: "发帖规则限流"},
			"advanced_result": {Code: 1003, Message: "高级规则限流"},
			"default_result":  {Code: 1004, Message: "默认规则限流"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "business_rule",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "test_priority", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 1},
					Result: "business_result",
				},
			},
			Post: []config.RateRule{
				{
					Name:   "post_rule",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "test_priority", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 2},
					Result: "post_result",
				},
			},
			Advanced: []config.RateRule{
				{
					Name:   "advanced_rule",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "test_priority", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 3},
					Result: "advanced_result",
				},
			},
			Default: []config.RateRule{
				{
					Name:   "default_rule",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "+", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 10},
					Result: "default_result",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	req := &Request{Act: "test_priority", UID: "user_priority_test"}

	// 第1次请求应该允许
	resp, err := engine.Check(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Allowed, "第1次请求应该允许")

	// 第2次请求应该被业务规则限流（count=1）
	resp, err = engine.Check(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed, "第2次请求应该被限流")
	assert.Equal(t, "business_rule", resp.RuleName, "应该被业务规则限流，而不是其他规则")
	assert.Equal(t, 1001, resp.Code)
}

func TestEngine_Check_PostVsAdvancedPriority(t *testing.T) {
	// 测试 Post 规则优先于 Advanced 规则
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"post_result":     {Code: 2001, Message: "发帖规则限流"},
			"advanced_result": {Code: 2002, Message: "高级规则限流"},
		},
		Rules: config.RateRules{
			Post: []config.RateRule{
				{
					Name:   "post_priority_rule",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "post_test", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 1},
					Result: "post_result",
				},
			},
			Advanced: []config.RateRule{
				{
					Name:   "advanced_priority_rule",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "post_test", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 2},
					Result: "advanced_result",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	req := &Request{Act: "post_test", UID: "user_post_priority"}

	// 第1次请求允许
	resp, err := engine.Check(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Allowed)

	// 第2次请求应该被 Post 规则限流（不是 Advanced）
	resp, err = engine.Check(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, "post_priority_rule", resp.RuleName)
	assert.Equal(t, 2001, resp.Code)
}

func TestEngine_Check_AdvancedVsDefaultPriority(t *testing.T) {
	// 测试 Advanced 规则优先于 Default 规则
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"advanced_result": {Code: 3001, Message: "高级规则限流"},
			"default_result":  {Code: 3002, Message: "默认规则限流"},
		},
		Rules: config.RateRules{
			Advanced: []config.RateRule{
				{
					Name:   "advanced_vs_default",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "adv_test", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 1},
					Result: "advanced_result",
				},
			},
			Default: []config.RateRule{
				{
					Name:   "default_catch_all",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "+", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 5},
					Result: "default_result",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	req := &Request{Act: "adv_test", UID: "user_adv_priority"}

	// 第1次请求允许
	resp, err := engine.Check(ctx, req)
	require.NoError(t, err)
	assert.True(t, resp.Allowed)

	// 第2次请求应该被 Advanced 规则限流（不是 Default）
	resp, err = engine.Check(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, "advanced_vs_default", resp.RuleName)
	assert.Equal(t, 3001, resp.Code)
}

func TestEngine_Check_DefaultFallback(t *testing.T) {
	// 测试当没有其他规则匹配时，Default 规则生效
	store := newMockStorage()
	engine := New(WithStorage(store))

	rulesConfig := &config.RulesConfig{
		Meta: config.Meta{Version: "1.0"},
		Results: map[string]config.Result{
			"business_result": {Code: 4001, Message: "业务规则限流"},
			"default_result":  {Code: 4002, Message: "默认规则限流"},
		},
		Rules: config.RateRules{
			Business: []config.RateRule{
				{
					Name:   "business_specific",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "specific_action"},
					Limit:  config.Limit{Time: time.Minute, Count: 1},
					Result: "business_result",
				},
			},
			Default: []config.RateRule{
				{
					Name:   "default_fallback",
					Type:   config.RuleTypeCount,
					Match:  map[string]string{"act": "+", "uid": "+"},
					Limit:  config.Limit{Time: time.Minute, Count: 2},
					Result: "default_result",
				},
			},
		},
	}

	err := engine.LoadRules(rulesConfig)
	require.NoError(t, err)

	ctx := context.Background()
	// 使用一个没有被 Business 规则匹配的 action
	req := &Request{Act: "other_action", UID: "user_default_test"}

	// 前2次请求允许
	for i := 0; i < 2; i++ {
		resp, err := engine.Check(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Allowed, "第%d次请求应该允许", i+1)
	}

	// 第3次请求应该被 Default 规则限流
	resp, err := engine.Check(ctx, req)
	require.NoError(t, err)
	assert.False(t, resp.Allowed)
	assert.Equal(t, "default_fallback", resp.RuleName)
	assert.Equal(t, 4002, resp.Code)
}

