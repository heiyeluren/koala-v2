# 03 - 核心算法

## 3.1 算法概述

Koala V2 提供四种限流算法，满足不同的业务场景：

| 算法 | 配置类型 | 适用场景 | 复杂度 |
|------|---------|---------|-------|
| **Direct** | (访问控制) | 黑白名单 | O(1) |
| **Count** | count | 固定周期限制 | O(1) |
| **Base** | accumulate | 高频用户特殊限制 | O(1) |
| **Leak** | freq | 平滑流量控制 | O(1) |

> **注意**: 配置文件中使用的类型名称为 `count`、`freq`、`accumulate`，分别对应 Count、Leak、Base 算法。

## 3.2 Direct 算法

### 3.2.1 原理

直接匹配，无需计数，命中即返回结果。

```
请求参数 → 匹配检查 → 命中 → 返回配置的结果
                    → 未命中 → 继续下一条规则
```

### 3.2.2 使用场景

- 用户白名单：VIP 用户直接放行
- 用户黑名单：封禁用户直接拒绝
- IP 白名单：内部 IP 直接放行
- IP 黑名单：恶意 IP 直接拒绝

### 3.2.3 实现代码

```go
// internal/engine/algorithm/direct.go

package algorithm

import (
    "context"
)

type DirectAlgorithm struct{}

func NewDirect() *DirectAlgorithm {
    return &DirectAlgorithm{}
}

// Browse 检查是否命中
// 返回: hit=true 表示命中规则
func (a *DirectAlgorithm) Browse(ctx context.Context, key string, limit LimitConfig, storage Storage) (hit bool, err error) {
    // Direct 算法不需要查询存储
    // 命中与否由 Matcher 决定，到达这里说明已经匹配成功
    return true, nil
}

// Update Direct 算法无需更新
func (a *DirectAlgorithm) Update(ctx context.Context, key string, limit LimitConfig, storage Storage) error {
    return nil
}
```

### 3.2.4 配置示例

```toml
[[access.whitelist]]
name = "vip_users"
match = { uid = "@vip_list" }
result = "allow"

[[access.blacklist]]
name = "banned_users"
match = { uid = "@banned_list" }
result = "deny"
```

## 3.3 Count 算法

### 3.3.1 原理

在固定时间窗口内统计请求次数，超过阈值则拒绝。

```
┌────────────────────────────────────────┐
│          时间窗口 (e.g. 24h)            │
│                                        │
│  请求1  请求2  请求3  ...  请求N        │
│    ↓      ↓      ↓          ↓          │
│  count=1 count=2 count=3   count=N     │
│                                        │
│  if count >= limit → 拒绝              │
└────────────────────────────────────────┘
        窗口结束时计数器自动过期
```

### 3.3.2 特殊处理：自然天

当 `time = 24h` 时，过期时间计算到当天 23:59:59：

```go
if limit.Time == 24*time.Hour {
    // 计算到今天结束的剩余时间
    now := time.Now()
    endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
    ttl = endOfDay.Sub(now)
}
```

### 3.3.3 Redis 操作

| 操作 | 命令 | 说明 |
|------|------|------|
| 检查 | `GET key` | 获取当前计数 |
| 首次更新 | `SET key 1 EX ttl` | 设置初始值和过期时间 |
| 后续更新 | `INCR key` | 原子递增 |

### 3.3.4 实现代码

```go
// internal/engine/algorithm/count.go

package algorithm

import (
    "context"
    "time"
)

type CountAlgorithm struct{}

func NewCount() *CountAlgorithm {
    return &CountAlgorithm{}
}

// Browse 检查计数是否超限
func (a *CountAlgorithm) Browse(ctx context.Context, key string, limit LimitConfig, storage Storage) (hit bool, err error) {
    // 获取当前计数
    count, err := storage.GetInt(ctx, key)
    if err != nil {
        if err == ErrKeyNotFound {
            return false, nil  // 首次请求，未超限
        }
        return false, err
    }

    // 检查是否超限
    return count >= limit.Count, nil
}

// Update 增加计数
func (a *CountAlgorithm) Update(ctx context.Context, key string, limit LimitConfig, storage Storage) error {
    // 检查 key 是否存在
    exists, err := storage.Exists(ctx, key)
    if err != nil {
        return err
    }

    if !exists {
        // 首次请求，设置初始值和过期时间
        ttl := a.calculateTTL(limit.Time)
        return storage.SetInt(ctx, key, 1, ttl)
    }

    // 递增计数
    _, err = storage.Incr(ctx, key)
    return err
}

// calculateTTL 计算过期时间
func (a *CountAlgorithm) calculateTTL(duration time.Duration) time.Duration {
    // 24 小时特殊处理：计算到当天结束
    if duration == 24*time.Hour {
        now := time.Now()
        endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
        return endOfDay.Sub(now)
    }
    return duration
}
```

### 3.3.5 配置示例

```toml
[[rules.business]]
name = "daily_comment_limit"
type = "count"
match = { act = "comment", uid = "+" }
limit = { time = "24h", count = 10 }
result = "deny"
desc = "每天最多评论10次"

[[rules.business]]
name = "minute_post_limit"
type = "count"
match = { act = "post", uid = "+" }
limit = { time = "1m", count = 3 }
result = "deny"
desc = "每分钟最多发帖3次"
```

### 3.3.6 局限性

**窗口边界问题**：在窗口切换时刻可能出现突发流量。

```
窗口1结束 | 窗口2开始
    ↓    |    ↓
 count=9 | count=0
 请求10  | 请求1,2,3...
```

解决方案：使用 Leak 算法实现平滑限流。

## 3.4 Base 算法

### 3.4.1 原理

二级阈值计数：当主计数器达到基础阈值后，启用时间窗口限制。

```
┌─────────────────────────────────────────────────────────────┐
│  主计数器 (每天累计)                                         │
│                                                             │
│  请求1...请求10 (base=10)                                   │
│        ↓                                                    │
│  达到基础阈值，启用次级限制                                   │
│        ↓                                                    │
│  ┌─────────────────────────────────────┐                   │
│  │  次级计数器 (时间窗口内)              │                   │
│  │  每 5 秒最多 1 次                    │                   │
│  │  count >= 1 → 拒绝                  │                   │
│  └─────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

### 3.4.2 使用场景

- 正常用户快速操作，高频用户受限
- "每天超过 10 次后，每 5 秒限 1 次"

### 3.4.3 Redis 操作

| 操作 | 键 | 命令 |
|------|-----|------|
| 主计数器检查 | `{key}` | `GET` |
| 主计数器更新 | `{key}` | `INCR` |
| 次级计数器检查 | `{key}_B` | `GET` |
| 次级计数器更新 | `{key}_B` | `SETEX` / `INCR` |

### 3.4.4 实现代码

```go
// internal/engine/algorithm/base.go

package algorithm

import (
    "context"
    "time"
)

type BaseAlgorithm struct{}

func NewBase() *BaseAlgorithm {
    return &BaseAlgorithm{}
}

const baseSuffix = "_B"

// Browse 检查是否超限
func (a *BaseAlgorithm) Browse(ctx context.Context, key string, limit LimitConfig, storage Storage) (hit bool, err error) {
    // 1. 获取主计数器
    mainCount, err := storage.GetInt(ctx, key)
    if err != nil && err != ErrKeyNotFound {
        return false, err
    }

    // 未达到基础阈值，放行
    if mainCount < limit.Base {
        return false, nil
    }

    // 2. 达到基础阈值，检查次级计数器
    secondaryKey := key + baseSuffix
    secondaryCount, err := storage.GetInt(ctx, secondaryKey)
    if err != nil && err != ErrKeyNotFound {
        return false, err
    }

    // 次级计数器超限
    return secondaryCount >= limit.Count, nil
}

// Update 更新计数器
func (a *BaseAlgorithm) Update(ctx context.Context, key string, limit LimitConfig, storage Storage) error {
    // 1. 检查主计数器是否存在
    exists, err := storage.Exists(ctx, key)
    if err != nil {
        return err
    }

    if !exists {
        // 首次请求，初始化主计数器（24小时过期）
        ttl := a.calculateMainTTL()
        return storage.SetInt(ctx, key, 1, ttl)
    }

    // 2. 递增主计数器
    mainCount, err := storage.Incr(ctx, key)
    if err != nil {
        return err
    }

    // 3. 达到基础阈值后，更新次级计数器
    if mainCount >= limit.Base {
        secondaryKey := key + baseSuffix
        secondaryExists, err := storage.Exists(ctx, secondaryKey)
        if err != nil {
            return err
        }

        if !secondaryExists {
            // 首次触发，初始化次级计数器
            return storage.SetInt(ctx, secondaryKey, 1, limit.Time)
        }

        // 递增次级计数器
        _, err = storage.Incr(ctx, secondaryKey)
        return err
    }

    return nil
}

// calculateMainTTL 计算主计数器过期时间（到当天结束）
func (a *BaseAlgorithm) calculateMainTTL() time.Duration {
    now := time.Now()
    endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
    return endOfDay.Sub(now)
}
```

### 3.4.5 配置示例

```toml
[[rules.advanced]]
name = "ask_interval_control"
type = "accumulate"
match = { act = "ask", ip = "+" }
limit = { base = 10, time = "5s", count = 1 }
result = "deny"
desc = "同IP每天超过10次后，每5秒限1次"
```

## 3.5 Leak 算法

### 3.5.1 原理

漏桶算法：使用列表存储每次请求的时间戳，检查时间窗口内的请求数量。

```
┌─────────────────────────────────────────────────────────┐
│                    时间窗口 (60s)                        │
│                                                         │
│  ←───────────────── 60 秒 ──────────────────→          │
│                                                         │
│  [t1] [t2] [t3] [t4] [t5] ... [t100]                   │
│   ↑                            ↑                        │
│  最旧                         最新                       │
│                                                         │
│  检查: t_now - t[count] <= time_window ?               │
│  如果是 → 说明窗口内已有 count 个请求 → 拒绝            │
└─────────────────────────────────────────────────────────┘
```

### 3.5.2 优势

- **平滑限流**：没有窗口边界突发问题
- **精确控制**：任意 60 秒内最多 N 次

### 3.5.3 Redis 操作

| 操作 | 命令 | 说明 |
|------|------|------|
| 获取长度 | `LLEN key` | 列表当前长度 |
| 获取元素 | `LINDEX key index` | 获取指定位置的时间戳 |
| 插入元素 | `LPUSH key timestamp` | 在头部插入新时间戳 |
| 裁剪列表 | `LTRIM key 0 count` | 保留最新的 count+1 个元素 |
| 设置过期 | `EXPIRE key ttl` | 设置列表过期时间 |

### 3.5.4 实现代码

```go
// internal/engine/algorithm/leak.go

package algorithm

import (
    "context"
    "time"
)

type LeakAlgorithm struct{}

func NewLeak() *LeakAlgorithm {
    return &LeakAlgorithm{}
}

// Browse 检查是否超限
func (a *LeakAlgorithm) Browse(ctx context.Context, key string, limit LimitConfig, storage Storage) (hit bool, err error) {
    // 1. 获取列表长度
    length, err := storage.LLen(ctx, key)
    if err != nil {
        return false, err
    }

    // 列表长度不足，放行
    if length <= limit.Count {
        return false, nil
    }

    // 2. 获取第 count 个元素的时间戳
    timestamp, err := storage.LIndex(ctx, key, limit.Count)
    if err != nil {
        if err == ErrKeyNotFound || err == ErrIndexOutOfRange {
            return false, nil
        }
        return false, err
    }

    // 3. 检查时间差
    now := time.Now().Unix()
    if now-timestamp <= int64(limit.Time.Seconds()) {
        // 时间窗口内已有 count 个请求，拒绝
        return true, nil
    }

    return false, nil
}

// Update 记录请求时间戳
func (a *LeakAlgorithm) Update(ctx context.Context, key string, limit LimitConfig, storage Storage) error {
    now := time.Now().Unix()

    // 1. 在列表头部插入当前时间戳
    if err := storage.LPush(ctx, key, now); err != nil {
        return err
    }

    // 2. 设置过期时间（时间窗口的 2 倍，留有余量）
    ttl := limit.Time * 2
    if err := storage.Expire(ctx, key, ttl); err != nil {
        return err
    }

    // 3. 异步清理旧元素（保留 count+1 个）
    go a.cleanup(ctx, key, limit.Count, storage)

    return nil
}

// cleanup 清理过期元素
func (a *LeakAlgorithm) cleanup(ctx context.Context, key string, count int64, storage Storage) {
    // 裁剪列表，只保留前 count+1 个元素
    _ = storage.LTrim(ctx, key, 0, count)
}
```

### 3.5.5 配置示例

```toml
[[rules.advanced]]
name = "api_smooth_limit"
type = "freq"
match = { act = "api_call", uid = "+" }
limit = { time = "60s", count = 100 }
result = "deny"
desc = "API调用60秒内平滑限制100次"
```

## 3.6 匹配模式

### 3.6.1 匹配语法总览

| 语法 | 示例 | 说明 |
|------|------|------|
| 精确匹配 | `"post"` | 值等于 "post" |
| 任意值 | `"+"` | 匹配任何非空值 |
| 取反匹配 | `"!post"` | 值不等于 "post" |
| 多值匹配 | `"post,comment"` | 值为 "post" 或 "comment" |
| 数值范围 | `"1-100"` | 数值在 1 到 100 之间 |
| 大于 | `">1000"` | 数值大于 1000 |
| 小于 | `"<100"` | 数值小于 100 |
| IP 通配 | `"192.168.*"` | IP 匹配通配模式 |
| IP 范围 | `"192.168.0.1-192.168.0.255"` | IP 在范围内 |
| 字典引用 | `"@whitelist"` | 值在字典 whitelist 中 |
| 字典取反 | `"!@blacklist"` | 值不在字典 blacklist 中 |

### 3.6.2 匹配器实现

```go
// internal/engine/matcher.go

package engine

import (
    "net"
    "strconv"
    "strings"
)

type Matcher interface {
    Match(value string) bool
}

// ExactMatcher 精确匹配
type ExactMatcher struct {
    expected string
}

func (m *ExactMatcher) Match(value string) bool {
    return value == m.expected
}

// AnyMatcher 任意值匹配
type AnyMatcher struct{}

func (m *AnyMatcher) Match(value string) bool {
    return value != ""
}

// NotMatcher 取反匹配
type NotMatcher struct {
    inner Matcher
}

func (m *NotMatcher) Match(value string) bool {
    return !m.inner.Match(value)
}

// MultiMatcher 多值匹配
type MultiMatcher struct {
    values map[string]struct{}
}

func (m *MultiMatcher) Match(value string) bool {
    _, ok := m.values[value]
    return ok
}

// RangeMatcher 范围匹配
type RangeMatcher struct {
    min, max int64
}

func (m *RangeMatcher) Match(value string) bool {
    v, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return false
    }
    return v >= m.min && v <= m.max
}

// GreaterMatcher 大于匹配
type GreaterMatcher struct {
    threshold int64
}

func (m *GreaterMatcher) Match(value string) bool {
    v, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return false
    }
    return v > m.threshold
}

// LessMatcher 小于匹配
type LessMatcher struct {
    threshold int64
}

func (m *LessMatcher) Match(value string) bool {
    v, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return false
    }
    return v < m.threshold
}

// IPWildcardMatcher IP 通配匹配
type IPWildcardMatcher struct {
    pattern string
}

func (m *IPWildcardMatcher) Match(value string) bool {
    parts := strings.Split(m.pattern, ".")
    valueParts := strings.Split(value, ".")

    if len(parts) != 4 || len(valueParts) != 4 {
        return false
    }

    for i := 0; i < 4; i++ {
        if parts[i] != "*" && parts[i] != valueParts[i] {
            return false
        }
    }
    return true
}

// IPRangeMatcher IP 范围匹配
type IPRangeMatcher struct {
    startIP, endIP net.IP
}

func (m *IPRangeMatcher) Match(value string) bool {
    ip := net.ParseIP(value)
    if ip == nil {
        return false
    }
    return bytes.Compare(ip, m.startIP) >= 0 && bytes.Compare(ip, m.endIP) <= 0
}

// DictMatcher 字典匹配
type DictMatcher struct {
    dict map[string]struct{}
}

func (m *DictMatcher) Match(value string) bool {
    _, ok := m.dict[value]
    return ok
}

// ParseMatcher 解析匹配模式
func ParseMatcher(pattern string, dicts map[string]map[string]struct{}) (Matcher, error) {
    // 任意值
    if pattern == "+" {
        return &AnyMatcher{}, nil
    }

    // 取反
    if strings.HasPrefix(pattern, "!") {
        inner, err := ParseMatcher(pattern[1:], dicts)
        if err != nil {
            return nil, err
        }
        return &NotMatcher{inner: inner}, nil
    }

    // 字典引用
    if strings.HasPrefix(pattern, "@") {
        dictName := pattern[1:]
        dict, ok := dicts[dictName]
        if !ok {
            return nil, fmt.Errorf("dict not found: %s", dictName)
        }
        return &DictMatcher{dict: dict}, nil
    }

    // 范围匹配
    if strings.Contains(pattern, "-") && !strings.Contains(pattern, ".") {
        parts := strings.Split(pattern, "-")
        if len(parts) == 2 {
            min, err1 := strconv.ParseInt(parts[0], 10, 64)
            max, err2 := strconv.ParseInt(parts[1], 10, 64)
            if err1 == nil && err2 == nil {
                return &RangeMatcher{min: min, max: max}, nil
            }
        }
    }

    // 大于
    if strings.HasPrefix(pattern, ">") {
        threshold, err := strconv.ParseInt(pattern[1:], 10, 64)
        if err != nil {
            return nil, err
        }
        return &GreaterMatcher{threshold: threshold}, nil
    }

    // 小于
    if strings.HasPrefix(pattern, "<") {
        threshold, err := strconv.ParseInt(pattern[1:], 10, 64)
        if err != nil {
            return nil, err
        }
        return &LessMatcher{threshold: threshold}, nil
    }

    // IP 通配
    if strings.Contains(pattern, "*") {
        return &IPWildcardMatcher{pattern: pattern}, nil
    }

    // IP 范围
    if strings.Count(pattern, ".") == 6 && strings.Contains(pattern, "-") {
        parts := strings.Split(pattern, "-")
        if len(parts) == 2 {
            startIP := net.ParseIP(parts[0])
            endIP := net.ParseIP(parts[1])
            if startIP != nil && endIP != nil {
                return &IPRangeMatcher{startIP: startIP, endIP: endIP}, nil
            }
        }
    }

    // 多值匹配
    if strings.Contains(pattern, ",") {
        values := make(map[string]struct{})
        for _, v := range strings.Split(pattern, ",") {
            values[strings.TrimSpace(v)] = struct{}{}
        }
        return &MultiMatcher{values: values}, nil
    }

    // 精确匹配
    return &ExactMatcher{expected: pattern}, nil
}
```

## 3.7 缓存键生成

### 3.7.1 键格式

```
koala:{rule_name}:{sorted_params}
```

示例：
```
koala:daily_comment:act=comment|uid=12345
koala:api_limit:act=api_call|ip=192.168.1.1|uid=12345
```

### 3.7.2 生成规则

1. 参数按键名字母顺序排序
2. 每个参数格式：`key=value`
3. 参数间用 `|` 分隔
4. 前缀 `koala:{rule_name}:`

### 3.7.3 实现代码

```go
// internal/engine/cache_key.go

package engine

import (
    "sort"
    "strings"
)

func GenerateCacheKey(ruleName string, params map[string]string, matchKeys []string) string {
    var parts []string

    // 只使用规则中定义的匹配键
    for _, key := range matchKeys {
        if value, ok := params[key]; ok {
            parts = append(parts, key+"="+value)
        }
    }

    // 排序确保一致性
    sort.Strings(parts)

    return "koala:" + ruleName + ":" + strings.Join(parts, "|")
}
```

## 3.8 算法选择指南

| 场景 | 推荐算法 | 配置类型 | 理由 |
|------|---------|---------|------|
| 白名单/黑名单 | Direct | (访问控制规则) | 无需计数，直接决策 |
| 每天/每小时固定限制 | Count | count | 简单高效 |
| 高频用户特殊限制 | Base | accumulate | 正常用户不受影响 |
| 需要平滑流量 | Leak | freq | 无边界突发问题 |
| API 限流 | Leak | freq | 流量更稳定 |
| 登录失败限制 | Count | count | 简单场景足够 |
