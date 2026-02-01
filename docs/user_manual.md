# Koala 频率控制系统用户手册

## 目录

1. [系统简介](#1-系统简介)
2. [快速开始](#2-快速开始)
3. [服务配置](#3-服务配置)
4. [策略配置](#4-策略配置)
5. [API 接口](#5-api-接口)
6. [日志与问题排查](#6-日志与问题排查)
7. [常见问题](#7-常见问题)

---

## 1. 系统简介

Koala 是一个高性能的频率控制系统，用于防止恶意请求、限制用户操作频率、保护后端服务。

### 核心功能

| 功能 | 说明 |
|------|------|
| **白名单** | VIP用户、内部IP等直接放行 |
| **黑名单** | 恶意用户、封禁IP等直接拦截 |
| **频率限制** | 限制用户在时间窗口内的操作次数 |
| **批量检查** | 一次请求检查多个用户/行为 |

### 性能指标

- QPS: 50,000+ (Redis) / 75,000+ (Local)
- 延迟: P99 < 20ms
- 并发: 支持 500+ 并发连接

---

## 2. 快速开始

### 2.1 目录结构

```
koala/
├── koala              # 可执行文件
├── conf/
│   ├── koala.toml     # 服务配置
│   ├── rules.toml     # 策略配置
│   ├── whitelist_users.txt   # 白名单用户
│   └── blacklist_ips.txt     # 黑名单IP
└── logs/              # 日志目录
```

### 2.2 启动服务

```bash
# 前台启动（开发调试）
./koala -config conf/koala.toml

# 后台启动（生产环境）
nohup ./koala -config conf/koala.toml > logs/koala.log 2>&1 &

# 查看版本
./koala -version
```

### 2.3 验证服务

```bash
# 健康检查
curl http://localhost:18080/health

# 预期输出
{"status":"ok","timestamp":"2026-02-01T12:00:00Z"}
```

### 2.4 停止服务

```bash
# 方式1: 通过进程名
pkill -f koala

# 方式2: 通过PID
kill $(cat /var/run/koala.pid)

# 方式3: 优雅停止（发送 SIGTERM）
kill -15 <PID>
```

---

## 3. 服务配置

配置文件: `conf/koala.toml`

### 3.1 完整配置示例

```toml
# =============================================================================
# 服务器配置
# =============================================================================
[server]
listen = ":18080"              # 监听地址和端口
read_timeout = "5s"            # 读取超时
write_timeout = "5s"           # 写入超时
shutdown_timeout = "10s"       # 优雅关闭超时

# =============================================================================
# 规则配置
# =============================================================================
[rules]
file = "conf/rules.toml"       # 规则文件路径
reload_interval = "60s"        # 热加载间隔，0表示不自动加载

# =============================================================================
# 存储配置
# =============================================================================
[storage]
type = "local"                 # 存储类型: local 或 redis

# 本地存储配置（单机部署推荐）
[storage.local]
max_size = "64MB"              # 最大内存
num_counters = 100000          # 计数器数量

# Redis存储配置（多实例部署推荐）
[storage.redis]
addr = "127.0.0.1:6379"        # Redis地址
password = ""                  # Redis密码
db = 0                         # 数据库编号
pool_size = 100                # 连接池大小
dial_timeout = "5s"            # 连接超时
read_timeout = "3s"            # 读取超时
write_timeout = "3s"           # 写入超时

# 降级存储配置（Redis故障时自动切换到本地存储）
[storage.fallback]
enabled = true                 # 是否启用降级
health_check_interval = "5s"   # 健康检查间隔
failure_threshold = 3          # 连续失败次数阈值
recovery_interval = "30s"      # 恢复检查间隔

# =============================================================================
# 日志配置
# =============================================================================
[logging]
level = "info"                 # 日志级别: debug, info, warn, error
format = "json"                # 日志格式: json 或 console
console = true                 # 是否输出到控制台

# 日志文件配置
[logging.file]
enabled = true                 # 是否启用文件日志
path = "logs/koala.log"        # 日志文件路径
max_size = 100                 # 单个文件最大大小（MB）
max_backups = 10               # 保留的旧文件数量
max_age = 30                   # 保留天数
compress = true                # 是否压缩旧文件

# =============================================================================
# 监控配置
# =============================================================================
[metrics]
enabled = true                 # 是否启用监控
path = "/metrics"              # 监控端点路径
```

### 3.2 存储类型选择

| 类型 | 优点 | 缺点 | 适用场景 |
|------|------|------|----------|
| **local** | 性能高、无依赖 | 不支持多实例 | 单机部署 |
| **redis** | 支持多实例、可持久化 | 需要Redis | 分布式部署 |

### 3.3 降级存储说明

当使用 Redis 存储时，可以配置 `[storage.fallback]` 实现自动降级：

- **降级触发**: 当 Redis 连续失败达到 `failure_threshold` 次时，自动切换到本地存储
- **自动恢复**: 切换后，系统定期（`recovery_interval`）检查 Redis 是否恢复
- **数据一致性**: 降级期间的计数数据存储在本地，Redis 恢复后新请求使用 Redis

### 3.4 日志文件配置说明

`[logging.file]` 配置支持日志文件轮转：

| 字段 | 说明 | 示例 |
|------|------|------|
| `enabled` | 是否启用文件日志 | `true` |
| `path` | 日志文件路径 | `"logs/koala.log"` |
| `max_size` | 单个文件最大大小（MB） | `100` |
| `max_backups` | 保留的旧文件数量 | `10` |
| `max_age` | 保留天数 | `30` |
| `compress` | 是否压缩旧文件 | `true` |

---

## 4. 策略配置

配置文件: `conf/rules.toml`

### 4.1 策略概念说明

Koala 的策略分为三类：

| 类型 | 优先级 | 说明 |
|------|--------|------|
| **白名单** | 最高 | 匹配即放行，不检查其他规则 |
| **黑名单** | 次高 | 匹配即拦截，不检查其他规则 |
| **限流规则** | 最低 | 基于计数器判断是否超限 |

### 4.2 完整策略配置示例

```toml
# =============================================================================
# 元数据
# =============================================================================
[meta]
version = "1.0.0"
description = "生产环境规则配置"

# =============================================================================
# 字典文件（外部文件引用）
# =============================================================================
[dicts]
whitelist_users = "conf/whitelist_users.txt"    # 白名单用户文件
blacklist_ips = "conf/blacklist_ips.txt"        # 黑名单IP文件

# =============================================================================
# 结果模板（预定义返回值）
# =============================================================================
[results]

[results.allow]
code = 0
message = "ok"
auth_type = 0

[results.vip_allow]
code = 0
message = "VIP用户放行"
auth_type = 0

[results.internal_allow]
code = 0
message = "内网IP放行"
auth_type = 0

[results.ip_blocked]
code = 4003
message = "IP已被封禁"
auth_type = 0

[results.device_blocked]
code = 4004
message = "设备已被封禁"
auth_type = 0

[results.login_limit]
code = 4001
message = "登录过于频繁，请稍后重试"
auth_type = 1                  # 1表示需要验证码

[results.post_limit]
code = 4101
message = "发帖数量已达上限"
auth_type = 1

[results.comment_limit]
code = 4102
message = "评论过于频繁"
auth_type = 0

[results.global_limit]
code = 4999
message = "操作过于频繁"
auth_type = 0

# =============================================================================
# 访问控制规则
# =============================================================================
[access]

# ---------- 白名单规则 ----------

# VIP用户白名单（从文件读取）
[[access.whitelist]]
name = "vip_user_whitelist"
match = { uid = "@whitelist_users" }    # @开头表示引用字典文件
result = "vip_allow"

# 内网IP白名单（通配符匹配）
[[access.whitelist]]
name = "internal_ip_whitelist"
match = { ip = "10.*.*.*" }             # 支持通配符
result = "internal_allow"

# 内网IP白名单（另一网段）
[[access.whitelist]]
name = "internal_ip_whitelist_192"
match = { ip = "192.168.1.*" }
result = "internal_allow"

# ---------- 黑名单规则 ----------

# 封禁IP黑名单（从文件读取）
[[access.blacklist]]
name = "blocked_ip_blacklist"
match = { ip = "@blacklist_ips" }
result = "ip_blocked"

# 封禁设备黑名单（精确匹配）
[[access.blacklist]]
name = "blocked_device_blacklist"
match = { did = "blocked_device_001" }
result = "device_blocked"

# =============================================================================
# 限流规则
# =============================================================================
[rules]

# ---------- 业务规则（优先级高）----------

# 登录频率限制（按用户）
[[rules.business]]
name = "login_rate_limit"
type = "count"                          # count=计数器, freq=漏桶
match = { act = "login", uid = "+" }    # +表示任意非空值
limit = { time = "60s", count = 5 }     # 60秒内最多5次
result = "login_limit"

# 登录频率限制（按IP）
[[rules.business]]
name = "login_ip_rate_limit"
type = "count"
match = { act = "login_ip_test", ip = "+" }
limit = { time = "60s", count = 10 }
result = "login_limit"

# ---------- 发帖规则 ----------

# 发帖频率限制
[[rules.post]]
name = "post_user_limit"
type = "count"
match = { act = "post", uid = "+" }
limit = { time = "3600s", count = 20 }  # 1小时最多20篇
result = "post_limit"

# 评论频率限制
[[rules.post]]
name = "comment_limit"
type = "count"
match = { act = "comment", uid = "+" }
limit = { time = "60s", count = 10 }
result = "comment_limit"

# ---------- 默认规则（优先级最低）----------

# 全局默认限制
[[rules.default]]
name = "global_default_limit"
type = "count"
match = { act = "+", uid = "+" }
limit = { time = "60s", count = 100 }
result = "global_limit"
```

### 4.3 匹配模式说明

| 模式 | 示例 | 说明 |
|------|------|------|
| **精确匹配** | `"login"` | 完全相等 |
| **通配符** | `"10.*.*.*"` | 支持 * 通配 |
| **任意非空** | `"+"` | 匹配任意非空字符串 |
| **字典引用** | `"@whitelist"` | 从文件读取列表 |

### 4.4 字典文件格式

白名单用户文件 `whitelist_users.txt`:
```
vip_user_001
vip_user_002
admin_user
test_whitelist_user
```

黑名单IP文件 `blacklist_ips.txt`:
```
192.168.100.1
192.168.100.2
10.0.0.99
```

### 4.5 规则执行顺序

```
请求进入
    │
    ▼
┌─────────────┐
│  白名单检查  │ ──匹配──▶ 直接放行
└─────────────┘
    │不匹配
    ▼
┌─────────────┐
│  黑名单检查  │ ──匹配──▶ 直接拦截
└─────────────┘
    │不匹配
    ▼
┌─────────────┐
│  业务规则   │ ──超限──▶ 返回限流
└─────────────┘
    │未超限
    ▼
┌─────────────┐
│  发帖规则   │ ──超限──▶ 返回限流
└─────────────┘
    │未超限
    ▼
┌─────────────┐
│  默认规则   │ ──超限──▶ 返回限流
└─────────────┘
    │未超限
    ▼
  允许通过
```

---

## 5. API 接口

### 5.1 接口总览

| 接口 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/ready` | GET | 就绪检查 |
| `/metrics` | GET | Prometheus 监控指标 |
| `/api/v1/browse` | POST | 查询限流状态（不更新计数器） |
| `/api/v1/update` | POST | 更新计数器 |
| `/api/v1/batch` | POST | 批量查询 |

### 5.2 请求参数说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `act` | string | ✓ | 行为/动作，如 login, post, comment |
| `uid` | string | | 用户ID |
| `ip` | string | | 客户端IP |
| `did` | string | | 设备ID |
| `ext` | object | | 扩展字段 |
| `update` | bool | | Browse时是否自动更新计数器 |

### 5.3 响应参数说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `allowed` | bool | 是否允许通过 |
| `code` | int | 结果码，0表示正常 |
| `message` | string | 结果描述 |
| `rule_name` | string | 命中的规则名称 |
| `auth_type` | int | 验证类型，0=无需验证，1=需要验证码 |

---

### 5.4 接口使用示例

#### 5.4.1 健康检查

```bash
curl http://localhost:18080/health
```

**成功响应:**
```json
{
  "status": "ok",
  "timestamp": "2026-02-01T12:00:00Z"
}
```

#### 5.4.2 就绪检查

```bash
curl http://localhost:18080/ready
```

**成功响应:**
```json
{
  "ready": true,
  "timestamp": "2026-02-01T12:00:00Z"
}
```

---

#### 5.4.3 Browse API - 查询限流状态

##### 场景1: 正常请求（允许通过）

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "login",
    "uid": "user123",
    "ip": "192.168.1.100"
  }'
```

**响应:**
```json
{
  "allowed": true,
  "code": 0,
  "message": "ok"
}
```

##### 场景2: VIP用户（白名单放行）

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "login",
    "uid": "vip_user_001"
  }'
```

**响应:**
```json
{
  "allowed": true,
  "code": 0,
  "message": "VIP用户放行",
  "rule_name": "vip_user_whitelist"
}
```

##### 场景3: 内网IP（白名单放行）

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "login",
    "uid": "user123",
    "ip": "10.0.0.50"
  }'
```

**响应:**
```json
{
  "allowed": true,
  "code": 0,
  "message": "内网IP放行",
  "rule_name": "internal_ip_whitelist"
}
```

##### 场景4: 封禁IP（黑名单拦截）

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "login",
    "uid": "user123",
    "ip": "192.168.100.1"
  }'
```

**响应:**
```json
{
  "allowed": false,
  "code": 4003,
  "message": "IP已被封禁",
  "rule_name": "blocked_ip_blacklist"
}
```

##### 场景5: 封禁设备（黑名单拦截）

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "login",
    "uid": "user123",
    "did": "blocked_device_001"
  }'
```

**响应:**
```json
{
  "allowed": false,
  "code": 4004,
  "message": "设备已被封禁",
  "rule_name": "blocked_device_blacklist"
}
```

##### 场景6: 频率超限（触发限流）

```bash
# 连续请求6次触发限流（限制5次/分钟）
for i in {1..6}; do
  curl -X POST http://localhost:18080/api/v1/browse \
    -H "Content-Type: application/json" \
    -d '{"act":"login","uid":"test_user","update":true}'
  echo ""
done
```

**前5次响应:**
```json
{"allowed":true,"code":0,"message":"ok"}
```

**第6次响应（触发限流）:**
```json
{
  "allowed": false,
  "code": 4001,
  "message": "登录过于频繁，请稍后重试",
  "rule_name": "login_rate_limit",
  "auth_type": 1
}
```

##### 场景7: 带扩展字段

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "purchase",
    "uid": "user123",
    "ip": "192.168.1.100",
    "did": "device_abc",
    "ext": {
      "channel": "app",
      "platform": "ios",
      "version": "2.0.0"
    }
  }'
```

**响应:**
```json
{
  "allowed": true,
  "code": 0,
  "message": "ok"
}
```

---

#### 5.4.4 Update API - 更新计数器

##### 场景1: 正常更新

```bash
curl -X POST http://localhost:18080/api/v1/update \
  -H "Content-Type: application/json" \
  -d '{
    "act": "login",
    "uid": "user123"
  }'
```

**响应:**
```json
{
  "allowed": true,
  "code": 0,
  "message": "ok"
}
```

##### 场景2: 批量更新（循环调用）

```bash
# 模拟用户连续操作
for i in {1..10}; do
  curl -s -X POST http://localhost:18080/api/v1/update \
    -H "Content-Type: application/json" \
    -d "{\"act\":\"comment\",\"uid\":\"user_$i\"}"
done
```

---

#### 5.4.5 Batch API - 批量查询

##### 场景1: 批量检查多个用户

```bash
curl -X POST http://localhost:18080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"id": "1", "act": "login", "uid": "user001"},
      {"id": "2", "act": "login", "uid": "user002"},
      {"id": "3", "act": "login", "uid": "vip_user_001"},
      {"id": "4", "act": "login", "uid": "user004", "ip": "192.168.100.1"}
    ]
  }'
```

**响应:**
```json
{
  "results": [
    {"id": "1", "allowed": true, "code": 0, "message": "ok"},
    {"id": "2", "allowed": true, "code": 0, "message": "ok"},
    {"id": "3", "allowed": true, "code": 0, "message": "VIP用户放行", "rule_name": "vip_user_whitelist"},
    {"id": "4", "allowed": false, "code": 4003, "message": "IP已被封禁", "rule_name": "blocked_ip_blacklist"}
  ]
}
```

##### 场景2: 批量检查多种行为

```bash
curl -X POST http://localhost:18080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"id": "login_check", "act": "login", "uid": "user123"},
      {"id": "post_check", "act": "post", "uid": "user123"},
      {"id": "comment_check", "act": "comment", "uid": "user123"}
    ]
  }'
```

**响应:**
```json
{
  "results": [
    {"id": "login_check", "allowed": true, "code": 0, "message": "ok"},
    {"id": "post_check", "allowed": true, "code": 0, "message": "ok"},
    {"id": "comment_check", "allowed": true, "code": 0, "message": "ok"}
  ]
}
```

---

#### 5.4.6 错误场景

##### 错误1: 缺少必填字段 act

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"uid": "user123"}'
```

**响应 (HTTP 400):**
```json
{
  "allowed": false,
  "code": -1,
  "message": "act is required"
}
```

##### 错误2: 无效的JSON格式

```bash
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d 'invalid json'
```

**响应 (HTTP 400):**
```json
{
  "allowed": false,
  "code": -1,
  "message": "invalid request body"
}
```

##### 错误3: 错误的HTTP方法

```bash
curl http://localhost:18080/api/v1/browse
```

**响应 (HTTP 405):**
```json
{
  "allowed": false,
  "code": -1,
  "message": "method not allowed"
}
```

##### 错误4: 批量请求超过限制（最多100条）

```bash
# 生成超过100条的请求
curl -X POST http://localhost:18080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{"requests": [... 超过100条 ...]}'
```

**响应 (HTTP 400):**
```json
{
  "allowed": false,
  "code": -1,
  "message": "too many requests, max 100"
}
```

---

## 6. 日志与问题排查

### 6.1 日志级别

| 级别 | 说明 | 使用场景 |
|------|------|----------|
| `debug` | 详细调试信息 | 开发调试 |
| `info` | 一般信息 | 生产环境 |
| `warn` | 警告信息 | 需要关注 |
| `error` | 错误信息 | 需要处理 |

### 6.2 查看日志

```bash
# 实时查看日志
tail -f logs/koala.log

# 查看最近100行
tail -100 logs/koala.log

# 搜索错误日志
grep "error" logs/koala.log

# 搜索特定用户
grep "user123" logs/koala.log

# 搜索特定时间段
grep "2026-02-01T12:" logs/koala.log
```

### 6.3 常见问题排查

#### 问题1: 服务无法启动

**检查步骤:**
```bash
# 1. 检查端口是否被占用
lsof -i :18080

# 2. 检查配置文件是否存在
ls -la conf/koala.toml

# 3. 检查配置文件格式
cat conf/koala.toml

# 4. 检查规则文件
cat conf/rules.toml

# 5. 手动启动查看错误
./koala -config conf/koala.toml
```

#### 问题2: 规则不生效

**检查步骤:**
```bash
# 1. 确认规则文件路径正确
grep "file" conf/koala.toml

# 2. 检查规则语法
cat conf/rules.toml

# 3. 检查字典文件存在
ls -la conf/*.txt

# 4. 重启服务加载新规则
pkill -f koala && ./koala -config conf/koala.toml
```

#### 问题3: 连接Redis失败

**检查步骤:**
```bash
# 1. 检查Redis服务
redis-cli ping

# 2. 检查Redis配置
grep -A5 "redis" conf/koala.toml

# 3. 测试Redis连接
redis-cli -h 127.0.0.1 -p 6379 ping
```

#### 问题4: 限流不符合预期

**检查步骤:**
```bash
# 1. 查看当前计数器（Redis存储）
redis-cli KEYS "koala:*"

# 2. 查看具体计数器值
redis-cli GET "koala:login_rate_limit:act=login:uid=user123"

# 3. 清空计数器
redis-cli FLUSHALL
```

### 6.4 性能监控

```bash
# 查看Prometheus指标
curl http://localhost:18080/metrics

# 关键指标说明
# koala_requests_total - 总请求数
# koala_requests_duration_seconds - 请求延迟
# koala_rate_limit_hits_total - 限流触发次数
```

---

## 7. 常见问题

### Q1: 如何添加新的白名单用户？

**A:** 编辑 `conf/whitelist_users.txt` 文件，每行一个用户ID，然后重启服务或等待热加载。

### Q2: 如何临时封禁一个IP？

**A:** 编辑 `conf/blacklist_ips.txt` 文件，添加IP地址，然后重启服务。

### Q3: Browse 和 Update 的区别？

**A:**
- `Browse`: 只查询当前限流状态，不增加计数器
- `Update`: 增加计数器，用于确认用户真正执行了操作

**推荐用法:**
1. 用户请求时先调用 `Browse` 检查是否允许
2. 如果允许，执行业务逻辑
3. 业务执行成功后调用 `Update` 更新计数器

### Q4: 如何实现不同用户不同限制？

**A:** 通过白名单实现。VIP用户加入白名单后直接放行，不受限流规则影响。

### Q5: 计数器什么时候重置？

**A:** 计数器在时间窗口结束后自动过期。例如配置 `time = "60s"` 表示60秒后计数器自动清零。

### Q6: 如何查看当前被限流的用户？

**A:** 查看日志中的限流记录：
```bash
grep "rate_limit" logs/koala.log | grep "allowed\":false"
```

### Q7: 服务支持热加载配置吗？

**A:** 支持。配置 `reload_interval` 为非零值即可自动重新加载规则文件。
```toml
[rules]
file = "conf/rules.toml"
reload_interval = "60s"  # 每60秒检查一次
```

---

## 附录

### A. 错误码对照表

| 错误码 | 说明 | 触发条件 | 处理建议 |
|--------|------|----------|----------|
| 0 | 正常 | 请求通过 | 允许操作 |
| -1 | 请求错误 | 参数缺失或格式错误 | 检查请求参数 |
| 4001 | 登录限流（用户级） | 同一用户登录次数超限 | 等待或验证码 |
| 4002 | 登录限流（IP级） | 同一IP登录次数超限（需配置 `login_ip_test` 规则） | 更换网络或等待 |
| 4003 | IP封禁 | IP在黑名单中 | 联系管理员 |
| 4004 | 设备封禁 | 设备ID在黑名单中 | 联系管理员 |
| 4101 | 发帖限流 | 发帖次数超限 | 等待 |
| 4102 | 评论限流 | 等待 |
| 4999 | 全局限流 | 等待 |

### B. auth_type 说明

| 值 | 说明 |
|----|------|
| 0 | 无需额外验证 |
| 1 | 需要图形验证码 |
| 2 | 需要短信验证码 |

### C. 联系方式

如有问题，请联系系统管理员。
