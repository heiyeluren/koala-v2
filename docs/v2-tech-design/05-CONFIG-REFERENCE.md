# 05 - 配置参考

## 5.1 配置文件概述

| 文件 | 格式 | 说明 |
|------|------|------|
| koala.toml | TOML | 服务配置（端口、存储、日志等） |
| rules.toml | TOML | 规则配置（限流规则、结果模板等） |
| *.txt | 文本 | 字典文件（每行一个值） |

## 5.2 服务配置 (koala.toml)

### 5.2.1 完整配置示例

```toml
# Koala V2 服务配置

# ============================================================
# 服务器配置
# ============================================================
[server]
# 监听地址（必填，无默认值）
listen = ":9981"

# 读取超时
read_timeout = "5s"

# 写入超时
write_timeout = "5s"

# 优雅关闭超时
shutdown_timeout = "10s"

# ============================================================
# 规则配置
# ============================================================
[rules]
# 规则配置文件路径
file = "conf/rules.toml"

# 热重载检查间隔（当前仅用于配置记录，实际热重载通过 fsnotify 事件驱动）
reload_interval = "30s"

# ============================================================
# 存储配置
# ============================================================
[storage]
# 存储类型: local | redis（默认 "local"）
type = "local"

# Redis 配置（仅当 type = "redis" 时生效）
[storage.redis]
addr = "127.0.0.1:6379"
password = ""
db = 0
pool_size = 100
dial_timeout = "5s"
read_timeout = "3s"
write_timeout = "3s"

# 本地存储配置（用于纯本地模式或降级）
[storage.local]
# 最大内存（支持单位: KB, MB, GB；未设置时默认 512MB）
max_size = "64MB"

# Ristretto 计数器数量（推荐为预期 key 数量的 10 倍；未设置时为 Go int 零值 0）
num_counters = 10000

# 清理间隔（未设置时为 Go Duration 零值 0，即不主动清理）
# cleanup_interval = "1m"

# 降级策略配置
[storage.fallback]
# 是否启用降级
enabled = true

# ============================================================
# 日志配置
# ============================================================
[logging]
# 日志级别: debug | info | warn | error
level = "info"

# 日志格式: console | json（默认 "console"）
format = "json"

# 是否输出到控制台
console = true

# 日志文件配置
[logging.file]
# 是否启用文件日志
enabled = true

# 日志文件路径
path = "logs/koala.log"

# 单个文件最大大小（MB）
max_size = 100

# 保留的旧文件数量
max_backups = 10

# 保留天数
max_age = 30

# 是否压缩旧文件
compress = true

# ============================================================
# 指标配置
# ============================================================
[metrics]
# 是否启用 Prometheus 指标
enabled = true

# 指标路径
path = "/metrics"
```

### 5.2.2 配置字段详解

#### [server] 服务器配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| listen | string | **必填，无默认值** | 监听地址，validateConfig 检查非空 |
| read_timeout | duration | "5s" | 读取超时 |
| write_timeout | duration | "5s" | 写入超时 |
| shutdown_timeout | duration | "30s" | 优雅关闭超时 |

#### [rules] 规则配置引用

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| file | string | - | 规则配置文件路径 |
| reload_interval | duration | "30s" | 热重载检查间隔（当前仅用于配置记录，实际通过 fsnotify 事件驱动） |

#### [storage] 存储配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| type | string | "local" | 存储类型（local 或 redis） |

#### [storage.redis] Redis 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| addr | string | - | Redis 地址 |
| password | string | "" | 密码 |
| db | int | 0 | 数据库编号 |
| pool_size | int | 0 | 连接池大小 |
| dial_timeout | duration | 0 | 连接超时 |
| read_timeout | duration | 0 | 读取超时 |
| write_timeout | duration | 0 | 写入超时 |

#### [storage.local] 本地存储配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| max_size | string | 未设置时等效 "512MB" | 最大内存（GetMaxSizeBytes 在空值时返回 512MB） |
| num_counters | int | 0（Go 零值，未由 applyDefaults 设置） | 计数器数量（推荐为预期 key 数量的 10 倍） |
| cleanup_interval | duration | 0（Go 零值，未由 applyDefaults 设置） | 清理间隔 |

#### [storage.fallback] 降级配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | false | 是否启用降级 |

#### [logging] 日志配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| level | string | "info" | 日志级别 |
| format | string | "console" | 日志格式（console 或 json） |
| console | bool | false | 是否输出到控制台 |

#### [logging.file] 文件日志配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | false | 是否启用文件日志 |
| path | string | "" | 日志文件路径 |
| max_size | int | 0 | 单个文件最大大小（MB） |
| max_backups | int | 0 | 保留的旧文件数量 |
| max_age | int | 0 | 保留天数 |
| compress | bool | false | 是否压缩旧文件 |

#### [metrics] 指标配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | false | 是否启用 Prometheus 指标 |
| path | string | "/metrics" | 指标暴露路径 |

## 5.3 规则配置 (rules.toml)

### 5.3.1 完整配置示例

```toml
# Koala V2 规则配置

# ============================================================
# 元信息
# ============================================================
[meta]
# 配置版本
version = "1.0.0"

# 配置描述
description = "GIL 教育平台频率控制规则"

# ============================================================
# 字典定义
# ============================================================
[dicts]
# 用户白名单
uid_whitelist = "conf/uid_whitelist.txt"

# 用户黑名单
uid_blacklist = "conf/uid_blacklist.txt"

# IP 白名单
ip_whitelist = "conf/ip_whitelist.txt"

# IP 黑名单
ip_blacklist = "conf/ip_blacklist.txt"

# VIP 用户列表
vip_users = "conf/vip_users.txt"

# ============================================================
# 结果模板（支持 inline table 或 section table 两种写法）
# ============================================================

# 写法一：section table（推荐，便于添加注释）
[results.allow]
code = 0
message = "ok"
auth_type = 0

[results.deny]
code = 10
message = "操作过于频繁"
auth_type = 0

[results.auth_slider]
code = 20
message = "需要滑块验证"
auth_type = 1

[results.auth_sms]
code = 21
message = "需要短信验证"
auth_type = 3

[results.auth_captcha]
code = 22
message = "需要图形验证码"
auth_type = 5

# ============================================================
# 访问控制规则（Phase 1: 最先执行）
# ============================================================

# 白名单规则（命中则立即放行）
[[access.whitelist]]
name = "vip_users"
match = { uid = "@vip_users" }
result = "allow"

[[access.whitelist]]
name = "uid_whitelist"
match = { uid = "@uid_whitelist" }
result = "allow"

[[access.whitelist]]
name = "ip_whitelist"
match = { ip = "@ip_whitelist" }
result = "allow"

# 黑名单规则（命中则立即拒绝）
[[access.blacklist]]
name = "uid_blacklist"
match = { uid = "@uid_blacklist" }
result = "deny"

[[access.blacklist]]
name = "ip_blacklist"
match = { ip = "@ip_blacklist" }
result = "deny"

# ============================================================
# 业务规则（Phase 2: 优先级 1）
# ============================================================

[[rules.business]]
name = "ai_question_normal"
type = "count"
match = { act = "ai_question", uid = "+", vip = "!true" }
limit = { time = "24h", count = 50 }
result = "deny"
desc = "普通用户每24小时AI问答50次"

[[rules.business]]
name = "ai_question_vip"
type = "count"
match = { act = "ai_question", uid = "+", vip = "true" }
limit = { time = "24h", count = 200 }
result = "deny"
desc = "VIP用户每24小时AI问答200次"

[[rules.business]]
name = "comment_daily"
type = "count"
match = { act = "comment", uid = "+" }
limit = { time = "24h", count = 20 }
result = "deny"
desc = "每24小时最多评论20次"

[[rules.business]]
name = "video_watch_rate"
type = "count"
match = { act = "video_start", uid = "+" }
limit = { time = "1m", count = 10 }
result = "deny"
desc = "每分钟最多开始观看10个视频"

# ============================================================
# 发帖规则（Phase 2: 优先级 2）
# ============================================================

[[rules.post]]
name = "post_daily"
type = "count"
match = { act = "post", uid = "+" }
limit = { time = "24h", count = 10 }
result = "deny"
desc = "每24小时最多发帖10次"

[[rules.post]]
name = "post_minute"
type = "count"
match = { act = "post", uid = "+" }
limit = { time = "1m", count = 2 }
result = "deny"
desc = "每分钟最多发帖2次"

[[rules.post]]
name = "reply_minute"
type = "count"
match = { act = "reply", uid = "+" }
limit = { time = "1m", count = 5 }
result = "deny"
desc = "每分钟最多回复5次"

# ============================================================
# 高级规则（Phase 2: 优先级 3）
# ============================================================

[[rules.advanced]]
name = "login_fail_protect"
type = "count"
match = { act = "login_fail", uid = "+" }
limit = { time = "10m", count = 5 }
result = "auth_slider"
desc = "10分钟内登录失败5次需要滑块验证"

[[rules.advanced]]
name = "ask_base_control"
type = "accumulate"
match = { act = "ask", ip = "+" }
limit = { base = 20, time = "10s", count = 1 }
result = "deny"
desc = "同IP超过20次后，每10秒限1次"

[[rules.advanced]]
name = "api_freq_control"
type = "freq"
match = { act = "api_call", uid = "+" }
limit = { time = "60s", count = 100 }
result = "deny"
desc = "API调用60秒内平滑限制100次"

[[rules.advanced]]
name = "search_freq_control"
type = "freq"
match = { act = "search", ip = "+" }
limit = { time = "10s", count = 20 }
result = "auth_captcha"
desc = "搜索10秒内超过20次需要验证码"

# ============================================================
# 默认规则（Phase 2: 优先级 4）
# ============================================================

[[rules.default]]
name = "global_ip_rate"
type = "count"
match = { ip = "+" }
limit = { time = "1s", count = 100 }
result = "deny"
desc = "单IP每秒最多100次请求"
```

### 5.3.2 配置字段详解

#### [meta] 元信息

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| version | string | 是 | 配置版本号 |
| description | string | 否 | 配置描述 |

#### [dicts] 字典定义

格式: `字典名 = "文件路径"`

字典文件格式：每行一个值

```
# uid_whitelist.txt
12345
67890
11111
```

#### [results] 结果模板

支持两种 TOML 写法：

**写法一：section table（推荐）**
```toml
[results.allow]
code = 0
message = "ok"
auth_type = 0
```

**写法二：inline table**
```toml
[results]
allow = { code = 0, message = "ok", auth_type = 0 }
```

结果字段说明：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | int | 是 | 结果码 |
| message | string | 是 | 结果消息 |
| auth_type | int | 否 | 验证类型（默认 0） |

验证类型说明:

| auth_type | 说明 |
|-----------|------|
| 0 | 无需验证 |
| 1 | 滑块验证 |
| 2 | 密码验证 |
| 3 | 短信验证 |
| 4 | 邮箱验证 |
| 5 | 图形验证码 |
| 6 | 其他 |

#### [[access.whitelist]] / [[access.blacklist]] 访问控制规则

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 规则名称（唯一） |
| match | object | 是 | 匹配条件 |
| result | string | 是 | 结果模板引用 |

#### [[rules.*]] 频率控制规则

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 规则名称（唯一） |
| type | string | 是 | 算法类型: count/freq/accumulate |
| match | object | 是 | 匹配条件 |
| limit | object | 是 | 限制参数 |
| result | string | 是 | 结果模板引用 |
| desc | string | 否 | 规则描述 |

limit 字段说明:

| 字段 | 类型 | 适用算法 | 说明 |
|------|------|---------|------|
| time | duration | count/freq/accumulate | 时间窗口 |
| count | int | count/freq/accumulate | 计数限制 |
| base | int | accumulate | 基础阈值（达到后启用二级限制） |

> **accumulate 类型说明**: `base` 为主计数器的阈值。当主计数器 <= base 时放行；
> 超过 base 后，在 `time` 窗口内允许最多 `count` 次请求。
> 主计数器的 TTL = clamp(time * 10, 1小时, 7天)。

### 5.3.3 匹配语法

| 语法 | 示例 | 说明 |
|------|------|------|
| 精确匹配 | `act = "post"` | 值等于 "post" |
| 任意值 | `uid = "+"` | 匹配任何非空值 |
| 取反 | `vip = "!true"` | 值不等于 "true"（简单字符串比较） |
| 多值 | `act = "post,comment"` | 值为列表中任意一个 |
| 数值范围 | `level = "1-10"` | 数值在范围内 |
| 大于 | `level = ">5"` | 数值大于 5 |
| 小于 | `level = "<10"` | 数值小于 10 |
| IP 通配 | `ip = "192.168.*.*"` | IP 通配匹配（必须为完整 4 段格式） |
| 字典引用 | `uid = "@whitelist"` | 值在字典中 |

> **注意**：`!` 取反前缀仅支持简单字符串比较（NotMatcher 实现为 `value != pattern[1:]`）。
> 不支持 `!@dict` 字典取反语法——`"!@blacklist"` 会被解析为 NotMatcher，
> 执行 `value != "@blacklist"` 的字符串比较，而非"值不在字典中"的语义。
> 如需实现"值不在黑名单中"的逻辑，请改用白名单规则反向实现。

### 5.3.4 时间格式

支持的时间单位:

| 单位 | 示例 | 说明 |
|------|------|------|
| s | "30s" | 秒 |
| m | "5m" | 分钟 |
| h | "1h" | 小时 |
| d | "1d" | 天（ParseLimitTime 转换为 24*N 小时） |
| 组合 | "1h30m" | 1小时30分钟（Go time.ParseDuration 语法） |

> **关于 "24h"**：`"24h"` 通过 Go 标准库 `time.ParseDuration` 解析，
> 表示精确的 24 小时时间窗口，**不是**自然日对齐（不会自动截止到当天 23:59:59）。
> 等价写法：`"1d"`（通过 ParseLimitTime 的天格式支持，转换为 `24 * time.Hour`）。

## 5.4 字典文件

### 5.4.1 格式

每行一个值，支持 `#` 注释：

```
# 用户白名单
# 格式: 每行一个用户ID

12345
67890

# VIP 用户
11111
22222
```

### 5.4.2 示例文件

**uid_whitelist.txt:**
```
# 管理员账号
admin_001
admin_002

# 测试账号
test_001
```

**ip_whitelist.txt:**
```
# 内网IP
192.168.1.1
192.168.1.2

# 办公室IP
10.0.0.100
```

## 5.5 配置加载流程

```
1. 读取 koala.toml（ConfigWatcher.Load）
   │
2. applyDefaults 应用默认值
   │
3. validateConfig 验证配置
   │   ├─→ 失败: 启动失败，输出错误
   │   └─→ 成功: 继续
   │
4. 读取 rules.toml（LoadRules）
   │
5. 解析字典文件（LoadDicts）
   │
6. 构建 RuleSet 对象
   │
7. Engine 通过 atomic.Pointer[RuleSet] 存储规则集
   │   （ConfigWatcher 内部用 sync.RWMutex 保护 config/rules/dicts 字段）
   │
8. 启动 fsnotify 文件监听（热重载）
```

## 5.6 热重载机制

### 5.6.1 触发方式

热重载通过 **fsnotify 事件驱动**实现，监听文件的 Write 和 Create 事件。
`reload_interval` 字段存在于配置结构中，但当前实际重载逻辑完全由 fsnotify 事件触发，
不使用轮询机制。

### 5.6.2 触发条件

- rules.toml 文件变化
- 字典文件变化
- koala.toml 文件变化（ConfigWatcher 也会监听，但一般建议重启服务）

### 5.6.3 重载流程

```
1. fsnotify 检测到文件 Write/Create 事件
   │
2. 防抖检查（同一文件 100ms 内的重复事件被忽略）
   │
3. 读取并解析新配置
   │
4. 验证配置有效性
   │   ├─→ 失败: 记录错误日志，保留旧配置
   │   └─→ 成功: 继续
   │
5. 更新对应资源：
   │   ├─→ rules.toml 变化: 通过 ConfigWatcher.mu (RWMutex) 更新 rules
   │   ├─→ 字典文件变化: 通过 ConfigWatcher.mu (RWMutex) 更新 dicts
   │   └─→ 上层 Engine 通过 atomic.Pointer 原子替换 RuleSet
   │
6. 触发 onChange 回调，记录变更事件
```

### 5.6.4 注意事项

- koala.toml 虽然也被 ConfigWatcher 监听，但存储类型等变更通常需要重启服务才能完全生效
- rules.toml 和字典文件支持热重载
- 配置解析失败不会影响当前运行的规则
- 防抖机制确保编辑器保存（可能产生多次写事件）只触发一次重载
