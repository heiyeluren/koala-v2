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
# 监听地址
listen = ":9981"

# 读取超时
read_timeout = "5s"

# 写入超时
write_timeout = "5s"

# 优雅关闭超时
shutdown_timeout = "30s"

# ============================================================
# 规则配置
# ============================================================
[rules]
# 规则配置文件路径
file = "conf/rules.toml"

# 热重载检查间隔
reload_interval = "30s"

# ============================================================
# 存储配置
# ============================================================
[storage]
# 存储类型: redis | local
type = "redis"

# Redis 配置
[storage.redis]
addr = "127.0.0.1:6379"
password = ""
db = 0
pool_size = 100
min_idle_conns = 10
dial_timeout = "5s"
read_timeout = "3s"
write_timeout = "3s"

# 本地存储配置（用于纯本地模式或降级）
[storage.local]
# 最大内存（支持单位: KB, MB, GB）
max_size = "1GB"

# Ristretto 计数器数量（推荐为预期 key 数量的 10 倍）
num_counters = 10000000

# 清理间隔
cleanup_interval = "1m"

# 降级策略配置
[storage.fallback]
# 是否启用降级
enabled = true

# 健康检查间隔
health_check_interval = "5s"

# 连续失败次数阈值（达到后切换到本地存储）
failure_threshold = 3

# 恢复检查间隔（切换到本地后，多久尝试恢复）
recovery_interval = "30s"

# ============================================================
# 日志配置
# ============================================================
[logging]
# 日志级别: debug | info | warn | error
level = "info"

# 日志格式: json | console
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

# 指标监听地址（留空则复用主服务端口）
listen = ""

# 指标路径
path = "/metrics"

# ============================================================
# 性能调优
# ============================================================
[performance]
# 最大并发数（0 = 不限制）
max_concurrent = 0

# 请求队列大小
queue_size = 10000

# 工作协程数（0 = 自动）
workers = 0
```

### 5.2.2 配置字段详解

#### [server] 服务器配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| listen | string | ":9981" | 监听地址 |
| read_timeout | duration | "5s" | 读取超时 |
| write_timeout | duration | "5s" | 写入超时 |
| shutdown_timeout | duration | "30s" | 优雅关闭超时 |

#### [storage] 存储配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| type | string | "redis" | 存储类型 |

#### [storage.redis] Redis 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| addr | string | "127.0.0.1:6379" | Redis 地址 |
| password | string | "" | 密码 |
| db | int | 0 | 数据库编号 |
| pool_size | int | 100 | 连接池大小 |
| min_idle_conns | int | 10 | 最小空闲连接 |
| dial_timeout | duration | "5s" | 连接超时 |
| read_timeout | duration | "3s" | 读取超时 |
| write_timeout | duration | "3s" | 写入超时 |

#### [storage.local] 本地存储配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| max_size | string | "1GB" | 最大内存 |
| num_counters | int | 10000000 | 计数器数量 |
| cleanup_interval | duration | "1m" | 清理间隔 |

#### [storage.fallback] 降级配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| enabled | bool | true | 是否启用降级 |
| health_check_interval | duration | "5s" | 健康检查间隔 |
| failure_threshold | int | 3 | 失败阈值 |
| recovery_interval | duration | "30s" | 恢复检查间隔 |

#### [logging] 日志配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| level | string | "info" | 日志级别 |
| format | string | "json" | 日志格式 |
| console | bool | true | 是否输出到控制台 |

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
# 结果模板
# ============================================================
[results]
# 默认允许
allow = { code = 0, message = "Allow" }

# 拒绝
deny = { code = 10, message = "Deny" }

# 需要滑块验证
auth_slider = { code = 20, message = "Auth", auth_type = 1 }

# 需要短信验证
auth_sms = { code = 21, message = "Auth", auth_type = 3 }

# 需要图形验证码
auth_captcha = { code = 22, message = "Auth", auth_type = 5 }

# ============================================================
# 访问控制规则（Phase 1: 最先执行）
# ============================================================

# 白名单规则（命中则立即放行）
[[access.whitelist]]
name = "vip_users"
match = { uid = "@vip_users" }
result = "allow"
desc = "VIP用户直接放行"

[[access.whitelist]]
name = "uid_whitelist"
match = { uid = "@uid_whitelist" }
result = "allow"
desc = "白名单用户直接放行"

[[access.whitelist]]
name = "ip_whitelist"
match = { ip = "@ip_whitelist" }
result = "allow"
desc = "白名单IP直接放行"

# 黑名单规则（命中则立即拒绝）
[[access.blacklist]]
name = "uid_blacklist"
match = { uid = "@uid_blacklist" }
result = "deny"
desc = "黑名单用户直接拒绝"

[[access.blacklist]]
name = "ip_blacklist"
match = { ip = "@ip_blacklist" }
result = "deny"
desc = "黑名单IP直接拒绝"

# ============================================================
# 业务规则（Phase 2: 优先级 1）
# ============================================================

[[rules.business]]
name = "ai_question_normal"
type = "count"
match = { act = "ai_question", uid = "+", vip = "!true" }
limit = { time = "24h", count = 50 }
result = "deny"
desc = "普通用户每天AI问答50次"

[[rules.business]]
name = "ai_question_vip"
type = "count"
match = { act = "ai_question", uid = "+", vip = "true" }
limit = { time = "24h", count = 200 }
result = "deny"
desc = "VIP用户每天AI问答200次"

[[rules.business]]
name = "comment_daily"
type = "count"
match = { act = "comment", uid = "+" }
limit = { time = "24h", count = 20 }
result = "deny"
desc = "每天最多评论20次"

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
desc = "每天最多发帖10次"

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
desc = "同IP每天超过20次后，每10秒限1次"

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
| desc | string | 否 | 规则描述 |

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
| base | int | accumulate | 基础阈值 |

### 5.3.3 匹配语法

| 语法 | 示例 | 说明 |
|------|------|------|
| 精确匹配 | `act = "post"` | 值等于 "post" |
| 任意值 | `uid = "+"` | 匹配任何非空值 |
| 取反 | `vip = "!true"` | 值不等于 "true" |
| 多值 | `act = "post,comment"` | 值为列表中任意一个 |
| 数值范围 | `level = "1-10"` | 数值在范围内 |
| 大于 | `level = ">5"` | 数值大于 5 |
| 小于 | `level = "<10"` | 数值小于 10 |
| IP 通配 | `ip = "192.168.*"` | IP 匹配通配 |
| 字典引用 | `uid = "@whitelist"` | 值在字典中 |
| 字典取反 | `uid = "!@blacklist"` | 值不在字典中 |

### 5.3.4 时间格式

支持的时间单位:

| 单位 | 示例 | 说明 |
|------|------|------|
| s | "30s" | 秒 |
| m | "5m" | 分钟 |
| h | "1h" | 小时 |
| 组合 | "1h30m" | 1小时30分钟 |
| 特殊 | "24h" | 自然天（到当天 23:59:59） |

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
1. 读取 koala.toml
   │
2. 读取 rules.toml
   │
3. 解析字典文件
   │
4. 验证配置有效性
   │   ├─→ 失败: 启动失败，输出错误
   │   └─→ 成功: 继续
   │
5. 构建 Policy 对象
   │
6. 存入 atomic.Pointer
   │
7. 启动文件监听（热重载）
```

## 5.6 热重载机制

### 5.6.1 触发条件

- rules.toml 文件变化
- 字典文件变化

### 5.6.2 重载流程

```
1. fsnotify 检测到文件变化
   │
2. 等待 100ms（防抖）
   │
3. 读取并解析新配置
   │
4. 验证配置有效性
   │   ├─→ 失败: 记录错误，保留旧配置
   │   └─→ 成功: 继续
   │
5. 原子替换 Policy
   │
6. 记录重载日志
```

### 5.6.3 注意事项

- koala.toml 变更需要重启服务
- rules.toml 和字典文件支持热重载
- 配置解析失败不会影响当前运行
