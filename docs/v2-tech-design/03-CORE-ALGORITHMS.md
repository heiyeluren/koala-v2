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

### 算法接口定义

```go
// internal/engine/algorithm/interface.go

type Algorithm interface {
    // Browse 检查是否达到限流阈值。
    // 返回 true 表示达到限制（应拒绝），返回 false 表示未达到限制（应允许）。
    Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (hit bool, err error)

    // Update 递增指定键的计数器。
    Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error

    // Name 返回算法名称。
    Name() string
}

type LimitConfig struct {
    Time  time.Duration // 时间窗口
    Count int64         // 窗口内的计数限制
    Base  int64         // 基础阈值（用于Base算法）
}
```

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

    "koala/internal/storage"
)

type Direct struct{}

func NewDirect() *Direct {
    return &Direct{}
}

// Browse 对于Direct算法始终返回 hit=true。
func (d *Direct) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (hit bool, err error) {
    // Direct 算法不需要查询存储
    // 命中与否由 Matcher 决定，到达这里说明已经匹配成功
    return true, nil
}

// Update 对于Direct算法是空操作。
func (d *Direct) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
    return nil
}

// Name 返回算法名称。
func (d *Direct) Name() string {
    return "direct"
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

### 3.3.2 Redis 操作

| 操作 | 命令 | 说明 |
|------|------|------|
| 检查 | `GET key` | 获取当前计数 |
| 更新 | `INCRWITHTTL key ttl` | 原子递增并设置过期时间 |

### 3.3.3 实现代码

```go
// internal/engine/algorithm/count.go

package algorithm

import (
    "context"

    "koala/internal/storage"
)

type Count struct{}

func NewCount() *Count {
    return &Count{}
}

// Browse 检查计数器是否已达到限制。
func (c *Count) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
    count, err := store.GetInt(ctx, key)
    if err != nil {
        if err == storage.ErrKeyNotFound {
            return false, nil // 尚无计数，未达到限制
        }
        return false, err
    }

    return count >= limit.Count, nil
}

// Update 递增计数器并设置TTL过期时间。
func (c *Count) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
    _, err := store.IncrWithTTL(ctx, key, limit.Time)
    return err
}

// Name 返回算法名称。
func (c *Count) Name() string {
    return "count"
}
```

> **设计说明**: Update 使用 `IncrWithTTL` 原子操作，避免 `Exists + SetInt / Incr` 的竞态条件。
> 首次调用时 `IncrWithTTL` 会创建 key 并设置 TTL；后续调用只递增计数，TTL 不变。

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

解决方案：使用 Freq 算法实现平滑限流。

## 3.4 Base 算法（配置类型：`accumulate`）

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
| 主计数器检查 | `{key}:total` | `GET` |
| 主计数器更新 | `{key}:total` | `INCRWITHTTL`（带 clampTTL） |
| 次级计数器检查 | `{key}:secondary` | `GET` |
| 次级计数器更新 | `{key}:secondary` | `INCRWITHTTL` |

### 3.4.4 实现代码

```go
// internal/engine/algorithm/base.go

package algorithm

import (
    "context"
    "time"

    "koala/internal/storage"
)

type Base struct{}

func NewBase() *Base {
    return &Base{}
}

// Browse 检查是否超限
// 未达到/刚好达到基础阈值时：始终返回 false（未命中）。
// 超过基础阈值时：检查二级限制。
func (b *Base) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
    // 获取总计数
    totalKey := key + ":total"
    total, err := store.GetInt(ctx, totalKey)
    if err != nil && err != storage.ErrKeyNotFound {
        return false, err
    }

    // 未达到或刚好达到基础阈值，未命中
    if total <= limit.Base {
        return false, nil
    }

    // 超过基础阈值，检查二级限制
    secondaryKey := key + ":secondary"
    count, err := store.GetInt(ctx, secondaryKey)
    if err != nil && err != storage.ErrKeyNotFound {
        return false, err
    }

    return count >= limit.Count, nil
}

// Update 递增总计数器和二级计数器
func (b *Base) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
    // 递增总计数（带 TTL，防止永不过期导致内存泄漏）
    totalKey := key + ":total"
    total, err := store.IncrWithTTL(ctx, totalKey, clampTTL(limit.Time))
    if err != nil {
        return err
    }

    // 如果超过基础阈值（严格大于），同时递增二级计数器
    if total > limit.Base {
        secondaryKey := key + ":secondary"
        _, err = store.IncrWithTTL(ctx, secondaryKey, limit.Time)
        if err != nil {
            return err
        }
    }

    return nil
}

// clampTTL 计算 totalKey 的 TTL。
// TTL = clamp(limit.Time * 10, 1小时, 7天)
func clampTTL(windowTime time.Duration) time.Duration {
    ttl := windowTime * 10
    const minTTL = time.Hour
    const maxTTL = 7 * 24 * time.Hour
    if ttl < minTTL {
        ttl = minTTL
    }
    if ttl > maxTTL {
        ttl = maxTTL
    }
    return ttl
}

// Name 返回算法名称。
func (b *Base) Name() string {
    return "base"
}
```

> **设计说明**:
> - 阈值判断使用 `total <= limit.Base`（小于等于），即刚好达到 Base 时仍然放行，
>   超过 Base 后才启用二级限制。
> - 主计数器 TTL 使用 `clampTTL(windowTime)` —— 函数内部执行 `windowTime * 10` 并
>   clamp 到 [1h, 7d] 范围，而非固定的"到当天结束"，以适应不同时间窗口场景，同时防止内存泄漏。
> - 使用 `IncrWithTTL` 原子操作，避免 `Exists + SetInt / Incr` 的竞态条件。

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

## 3.5 Leak 算法（配置类型：`freq`）

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
| 获取长度 | `LLEN key` | 列表当前长度（清理后） |
| 获取所有元素 | `LRANGE key 0 -1` | 获取全部时间戳（用于过期清理） |
| 插入元素 | `LPUSH key timestamp` | 在头部插入新时间戳 |
| 裁剪列表 | `LTRIM key start end` | 移除过期时间戳 |

### 3.5.4 并发安全：按 Key 加锁

Leak 算法的 `LRange + LTrim` 操作不是原子的，可能产生竞态条件。
实现中使用 `sync.Map` 为每个 key 维护独立的 `sync.Mutex`：

```go
type Leak struct {
    keyLocks sync.Map // 按 key 加锁，避免 LRange+LTrim 竞态
}

// getLock 获取指定 key 的互斥锁。
func (l *Leak) getLock(key string) *sync.Mutex {
    val, _ := l.keyLocks.LoadOrStore(key, &sync.Mutex{})
    return val.(*sync.Mutex)
}

// Browse 和 Update 方法都需要先获取 key 锁
func (l *Leak) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
    mu := l.getLock(key)
    mu.Lock()
    defer mu.Unlock()
    // ...清理过期条目后检查桶大小
}
```

> **设计说明**: 不同 key 之间不互相阻塞，同一 key 的操作串行化，
> 确保 `LRange` 读取和 `LTrim` 裁剪之间不会被其他请求插入新数据。

### 3.5.5 实现代码

```go
// internal/engine/algorithm/leak.go

package algorithm

import (
    "context"
    "sync"
    "time"

    "koala/internal/storage"
)

type Leak struct {
    keyLocks sync.Map // 按 key 加锁，避免 LRange+LTrim 竞态
}

func NewLeak() *Leak {
    return &Leak{}
}

func (l *Leak) getLock(key string) *sync.Mutex {
    val, _ := l.keyLocks.LoadOrStore(key, &sync.Mutex{})
    return val.(*sync.Mutex)
}

// Browse 检查桶是否已满
func (l *Leak) Browse(ctx context.Context, key string, limit LimitConfig, store storage.Storage) (bool, error) {
    mu := l.getLock(key)
    mu.Lock()
    defer mu.Unlock()

    // 首先，清理过期条目
    err := l.cleanExpired(ctx, key, limit, store)
    if err != nil {
        return false, err
    }

    // 检查桶大小
    length, err := store.LLen(ctx, key)
    if err != nil {
        return false, err
    }

    return length >= limit.Count, nil
}

// Update 向桶中添加新的时间戳
func (l *Leak) Update(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
    mu := l.getLock(key)
    mu.Lock()
    defer mu.Unlock()

    // 先清理过期条目
    err := l.cleanExpired(ctx, key, limit, store)
    if err != nil {
        return err
    }

    // 添加当前时间戳（毫秒精度）
    now := time.Now().UnixMilli()
    return store.LPush(ctx, key, now)
}

// cleanExpired 移除超出时间窗口的旧时间戳
func (l *Leak) cleanExpired(ctx context.Context, key string, limit LimitConfig, store storage.Storage) error {
    now := time.Now().UnixMilli()
    windowStart := now - limit.Time.Milliseconds()

    // 获取所有条目
    values, err := store.LRange(ctx, key, 0, -1)
    if err != nil {
        return err
    }

    if len(values) == 0 {
        return nil
    }

    // 找到窗口内第一个有效条目的索引
    validStart := -1
    for i, ts := range values {
        if ts >= windowStart {
            validStart = i
            break
        }
    }

    if validStart == -1 {
        // 所有条目都已过期，清空列表
        return store.LTrim(ctx, key, 1, 0)
    }

    if validStart > 0 {
        // 修剪过期条目
        return store.LTrim(ctx, key, 0, int64(len(values)-validStart-1))
    }

    return nil
}

// Name 返回算法名称。
func (l *Leak) Name() string {
    return "leak"
}
```

### 3.5.6 配置示例

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
| IP 通配 | `"192.168.*.*"` | IP 匹配通配模式 |
| 字典引用 | `"@whitelist"` | 值在字典 whitelist 中 |
| 字典取反 | `"!@blacklist"` | 值不在字典 blacklist 中 |

### 3.6.2 匹配器实现

匹配器位于独立子包 `internal/engine/matcher/`，采用**无状态设计**。
所有 Matcher 都是空结构体，不持有任何字段；pattern 在每次 `Match` 调用时解析。

文件结构：

```
internal/engine/matcher/
├── matcher.go      // 接口定义、Parse 函数、辅助函数
├── exact.go        // ExactMatcher
├── any.go          // AnyMatcher
├── not.go          // NotMatcher
├── multi.go        // MultiMatcher
├── range.go        // RangeMatcher
├── greater.go      // GreaterMatcher
├── less.go         // LessMatcher
├── ip.go           // IPMatcher（IP 通配符匹配）
├── dict.go         // DictMatcher（唯一有状态的 matcher，单例模式）
└── matcher_test.go // 测试
```

#### Matcher 接口

```go
// internal/engine/matcher/matcher.go

package matcher

// Matcher 定义模式匹配的接口。
type Matcher interface {
    // Match 检查值是否与模式匹配。
    // pattern 是规则配置中的匹配模式，value 是请求中的实际值。
    Match(pattern string, value string) bool

    // Type 返回匹配器的类型名称。
    Type() string
}
```

> **设计说明**: 接口采用 2 参数 `Match(pattern, value)` 设计，而非预编译模式。
> 这使得所有 Matcher（除 DictMatcher 外）都是无状态空结构体，
> 可以安全地在多个规则之间共享，无需同步。

#### Parse 函数

```go
// internal/engine/matcher/matcher.go

// Parse 分析模式并返回适当的匹配器。
// 模式前缀决定匹配器类型：
//   - "+" -> AnyMatcher（匹配任何非空值）
//   - "!" -> NotMatcher（当值不等于模式时匹配）
//   - "@" -> DictMatcher（当值在字典中时匹配，返回单例 defaultDictMatcher）
//   - ">" -> GreaterMatcher（当值大于阈值时匹配）
//   - "<" -> LessMatcher（当值小于阈值时匹配）
//   - 包含"," -> MultiMatcher（当值在列表中时匹配）
//   - 包含"-"且两侧为数字 -> RangeMatcher
//   - 包含"*"的IP格式 -> IPMatcher
//   - 默认 -> ExactMatcher
func Parse(pattern string) Matcher {
    if len(pattern) == 0 {
        return &ExactMatcher{}
    }

    switch pattern[0] {
    case '+':
        return &AnyMatcher{}
    case '!':
        return &NotMatcher{}
    case '@':
        return defaultDictMatcher
    case '>':
        return &GreaterMatcher{}
    case '<':
        return &LessMatcher{}
    }

    if strings.Contains(pattern, ",") {
        return &MultiMatcher{}
    }

    if isRangePattern(pattern) {
        return &RangeMatcher{}
    }

    if isIPWildcardPattern(pattern) {
        return &IPMatcher{}
    }

    return &ExactMatcher{}
}
```

> **注意**: `Parse` 返回 `Matcher`，无 error 返回值。DictMatcher 使用包级单例 `defaultDictMatcher`，
> 字典数据通过 `RegisterDict` / `RegisterDictSlice` 包级函数预先注册。

#### 各 Matcher 实现

**ExactMatcher** — 精确匹配（`exact.go`）

```go
// internal/engine/matcher/exact.go

package matcher

// ExactMatcher 当值与模式完全相等时匹配。
type ExactMatcher struct{}

func (m *ExactMatcher) Match(pattern string, value string) bool {
    return pattern == value
}

func (m *ExactMatcher) Type() string {
    return "exact"
}
```

**AnyMatcher** — 任意非空值匹配（`any.go`）

```go
// internal/engine/matcher/any.go

package matcher

// AnyMatcher 匹配任何非空值。模式应为 "+"。
type AnyMatcher struct{}

func (m *AnyMatcher) Match(pattern string, value string) bool {
    return len(value) > 0
}

func (m *AnyMatcher) Type() string {
    return "any"
}
```

**NotMatcher** — 取反匹配（`not.go`）

```go
// internal/engine/matcher/not.go

package matcher

// NotMatcher 当值不等于模式时匹配。
// 模式格式："!value"，Match 时去除 "!" 前缀后直接与 value 比较。
// 注意：这是无状态设计，不内嵌其他 Matcher，直接字符串比较。
type NotMatcher struct{}

func (m *NotMatcher) Match(pattern string, value string) bool {
    if len(pattern) == 0 {
        return true
    }
    negatedValue := pattern[1:] // 移除 "!" 前缀
    return value != negatedValue
}

func (m *NotMatcher) Type() string {
    return "not"
}
```

**MultiMatcher** — 多值匹配（`multi.go`）

```go
// internal/engine/matcher/multi.go

package matcher

import "strings"

// MultiMatcher 当值是模式中逗号分隔值之一时匹配。
// 模式格式："a,b,c"，每次 Match 时解析逗号分隔列表。
type MultiMatcher struct{}

func (m *MultiMatcher) Match(pattern string, value string) bool {
    parts := strings.Split(pattern, ",")
    for _, part := range parts {
        trimmed := strings.TrimSpace(part)
        if trimmed == value {
            return true
        }
    }
    return false
}

func (m *MultiMatcher) Type() string {
    return "multi"
}
```

**RangeMatcher** — 数值范围匹配（`range.go`）

```go
// internal/engine/matcher/range.go

package matcher

import "strconv"

// RangeMatcher 当数值在指定范围内时匹配。
// 模式格式："min-max"，支持负数如 "-10-10"。
type RangeMatcher struct{}

func (m *RangeMatcher) Match(pattern string, value string) bool {
    val, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return false
    }
    min, max, ok := parseRange(pattern)
    if !ok {
        return false
    }
    return val >= min && val <= max
}

func (m *RangeMatcher) Type() string {
    return "range"
}
```

**GreaterMatcher** — 大于匹配（`greater.go`）

```go
// internal/engine/matcher/greater.go

package matcher

import "strconv"

// GreaterMatcher 当数值大于阈值时匹配。
// 模式格式：">10"，Match 时从 pattern 中解析阈值。
type GreaterMatcher struct{}

func (m *GreaterMatcher) Match(pattern string, value string) bool {
    if len(pattern) < 2 {
        return false
    }
    threshold, err := strconv.ParseInt(pattern[1:], 10, 64)
    if err != nil {
        return false
    }
    val, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return false
    }
    return val > threshold
}

func (m *GreaterMatcher) Type() string {
    return "greater"
}
```

**LessMatcher** — 小于匹配（`less.go`）

```go
// internal/engine/matcher/less.go

package matcher

import "strconv"

// LessMatcher 当数值小于阈值时匹配。
// 模式格式："<10"，Match 时从 pattern 中解析阈值。
type LessMatcher struct{}

func (m *LessMatcher) Match(pattern string, value string) bool {
    if len(pattern) < 2 {
        return false
    }
    threshold, err := strconv.ParseInt(pattern[1:], 10, 64)
    if err != nil {
        return false
    }
    val, err := strconv.ParseInt(value, 10, 64)
    if err != nil {
        return false
    }
    return val < threshold
}

func (m *LessMatcher) Type() string {
    return "less"
}
```

**IPMatcher** — IP 通配符匹配（`ip.go`）

```go
// internal/engine/matcher/ip.go

package matcher

import "strings"

// IPMatcher 使用通配符模式匹配 IP 地址。
// 模式格式："192.168.*.*"，通配符 "*" 可用于任何八位组位置。
// 注意：代码中不存在 IPRangeMatcher，仅支持通配符匹配。
type IPMatcher struct{}

func (m *IPMatcher) Match(pattern string, value string) bool {
    patternParts := strings.Split(pattern, ".")
    valueParts := strings.Split(value, ".")

    if len(patternParts) != 4 || len(valueParts) != 4 {
        return false
    }

    for i := 0; i < 4; i++ {
        if patternParts[i] == "*" {
            continue
        }
        if patternParts[i] != valueParts[i] {
            return false
        }
    }
    return true
}

func (m *IPMatcher) Type() string {
    return "ip"
}
```

**DictMatcher** — 字典匹配（`dict.go`）

```go
// internal/engine/matcher/dict.go

package matcher

import "sync"

// DictMatcher 根据命名字典匹配值。
// 模式格式："@dict_name"，当值存在于指定字典中时匹配。
// 这是唯一有状态的 Matcher：内部持有多个字典 + sync.RWMutex。
// 通过包级单例 defaultDictMatcher 使用，Parse("@xxx") 返回该单例。
type DictMatcher struct {
    mu    sync.RWMutex
    dicts map[string]map[string]bool  // 字典名 -> {值 -> bool}
}

func NewDictMatcher() *DictMatcher {
    return &DictMatcher{
        dicts: make(map[string]map[string]bool),
    }
}

func (m *DictMatcher) Match(pattern string, value string) bool {
    if len(pattern) < 2 || pattern[0] != '@' {
        return false
    }
    dictName := pattern[1:]
    if dictName == "" {
        return false
    }

    m.mu.RLock()
    defer m.mu.RUnlock()

    dict, exists := m.dicts[dictName]
    if !exists {
        return false
    }
    return dict[value]
}

func (m *DictMatcher) Type() string {
    return "dict"
}

// RegisterDict 注册字典，values 的 key 是要匹配的值。
func (m *DictMatcher) RegisterDict(name string, values map[string]bool) { ... }

// RegisterDictSlice 从切片注册字典。
func (m *DictMatcher) RegisterDictSlice(name string, values []string) { ... }
```

> **DictMatcher 设计说明**:
> - 内部存储为 `map[string]map[string]bool`（多字典），使用 `sync.RWMutex` 保护并发读写。
> - 包级变量 `defaultDictMatcher = NewDictMatcher()` 作为全局单例。
> - 包级函数 `RegisterDict()` / `RegisterDictSlice()` 委托给 `defaultDictMatcher` 方法。
> - `Parse("@xxx")` 返回该单例，不创建新实例。

#### 与 Rule 的集成

`Rule` 结构体同时持有 `Match`（原始模式字符串）和 `Matchers`（预编译的匹配器）：

```go
// internal/engine/rule.go

type Rule struct {
    Name      string
    Type      RuleType
    Phase     RulePhase
    Match     map[string]string           // 匹配条件（字段名 -> 模式字符串）
    Matchers  map[string]matcher.Matcher  // 预编译的匹配器（字段名 -> Matcher）
    Limit     LimitConfig
    Algorithm algorithm.Algorithm
    Result    ResultConfig
}
```

> **注意**: `Match` 存储原始配置模式（如 `"+"`, `"!post"`, `"@vip_list"`），
> `Matchers` 存储由 `matcher.Parse()` 预编译的 Matcher 实例。
> 匹配时使用 `Matchers[field].Match(Match[field], requestValue)` 调用。

## 3.7 缓存键生成

### 3.7.1 键格式

```
koala:{rule_name}:{sorted_params}
```

示例：
```
koala:daily_comment:act=comment:uid=12345
koala:api_limit:act=api_call:ip=192.168.1.1:uid=12345
```

### 3.7.2 生成规则

1. 参数按键名字母顺序排序
2. 每个参数格式：`key=value`
3. 参数间用 `:` 分隔
4. 前缀 `koala:{rule_name}:`

### 3.7.3 实现代码

```go
// internal/engine/rule.go - GenerateKey 方法

package engine

import (
    "fmt"
    "sort"
)

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
