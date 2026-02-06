# 02 - 架构设计

## 2.1 系统架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Client                                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           HTTP Layer (Gin)                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   Logger    │  │  Recovery   │  │   Metrics   │  │   Router    │    │
│  │ Middleware  │  │ Middleware  │  │ Middleware  │  │             │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Handler Layer                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   Browse    │  │   Update    │  │    Batch    │  │   Health    │    │
│  │  Handler    │  │  Handler    │  │   Handler   │  │   Handler   │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Rule Engine                                    │
│  ┌───────────────────────────────────────────────────────────────┐      │
│  │                      RuleSet (atomic.Pointer)                  │      │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │      │
│  │  │   Access    │  │    Rate     │  │   Results   │           │      │
│  │  │   Rules     │  │   Rules     │  │  Templates  │           │      │
│  │  └─────────────┘  └─────────────┘  └─────────────┘           │      │
│  └───────────────────────────────────────────────────────────────┘      │
│  ┌───────────────────────────────────────────────────────────────┐      │
│  │                        Algorithms                              │      │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐              │      │
│  │  │ Direct │  │ Count  │  │  Base  │  │  Leak  │              │      │
│  │  └────────┘  └────────┘  └────────┘  └────────┘              │      │
│  └───────────────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Storage Manager                                   │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │              Failover (atomic.Pointer[storage.Storage])          │    │
│  │         ┌──────────────────────────────────────┐                │    │
│  │         ▼                                      ▼                │    │
│  │  ┌─────────────┐                       ┌─────────────┐          │    │
│  │  │    Redis    │                       │    Local    │          │    │
│  │  │   Single    │                       │   Storage   │          │    │
│  │  └─────────────┘                       └─────────────┘          │    │
│  │                                               │                 │    │
│  │                            ┌──────────────────┼─────────────┐   │    │
│  │                            ▼                  ▼             ▼   │    │
│  │                     ┌───────────┐      ┌───────────┐ ┌────────┐│    │
│  │                     │ Ristretto │      │  Counter  │ │  List  ││    │
│  │                     │  (String) │      │   Store   │ │ Store  ││    │
│  │                     └───────────┘      └───────────┘ └────────┘│    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Config Watcher                                   │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  fsnotify → Parse → Validate → atomic.Pointer.Store(newRuleSet) │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## 2.2 模块职责

### 2.2.1 HTTP Layer

| 模块 | 职责 |
|------|------|
| Router | 路由注册与分发，通过 `RouterConfig` 配置（含 `CORSAllowOrigins []string` 字段） |
| Logger Middleware | 请求日志记录 |
| Recovery Middleware | panic 恢复 |
| Metrics Middleware | Prometheus 指标采集（使用 `c.FullPath()` 路由模式作为 key，避免动态路径参数导致 map 无限增长） |
| CORS Middleware | 跨域请求处理。`CORSMiddlewareWithConfig(CORSConfig)` 支持可配置的允许来源列表（`AllowOrigins []string`）；`CORSMiddleware()` 为向后兼容包装，允许所有来源 |

### 2.2.2 Handler Layer

| Handler | 路由 | 职责 |
|---------|------|------|
| BrowseHandler | POST /api/v1/browse | 检查请求是否允许 |
| UpdateHandler | POST /api/v1/update | 记录请求，增加计数 |
| BatchHandler | POST /api/v1/batch | 批量检查 |
| HealthHandler | GET /health | 健康检查 |

### 2.2.3 Rule Engine

| 组件 | 职责 |
|------|------|
| RuleSet | 存储当前生效的规则集（原子指针） |
| AccessRules | 白名单/黑名单规则 |
| RateRules | 频率控制规则（按优先级分组） |
| Algorithms | 四种限流算法实现 |
| Matcher | 规则匹配器 |

### 2.2.4 Storage Manager

| 组件 | 职责 |
|------|------|
| StorageManager | 存储统一入口，管理主备切换与自动故障转移（内置健康检查与降级恢复逻辑） |
| RedisStorage | Redis 单机存储实现 |
| LocalStorage | 本地内存存储实现（Ristretto 缓存 + 自定义计数器/列表） |

### 2.2.5 Config Watcher

| 组件 | 职责 |
|------|------|
| Watcher | 监听配置文件变化 |
| Parser | 解析 TOML 配置 |
| Validator | 验证配置有效性 |
| Loader | 加载字典文件 |

## 2.3 数据流

### 2.3.1 Browse 请求流程

```
1. Client 发送 POST /api/v1/browse
   │
2. Gin Router 路由到 BrowseHandler
   │
3. Handler 解析请求参数
   │
4. 调用 RuleEngine.Evaluate(request)
   │
   ├─→ Phase 1: 访问控制检查
   │   ├─→ 检查 IP/UID 白名单 → 命中则立即返回 Allow
   │   └─→ 检查 IP/UID 黑名单 → 命中则立即返回 Deny
   │
   ├─→ Phase 2: 频率控制检查（按优先级）
   │   ├─→ rules.business (优先级 1)
   │   ├─→ rules.post     (优先级 2)
   │   ├─→ rules.advanced (优先级 3)
   │   └─→ 首次匹配即返回结果
   │
   └─→ Phase 3: 无匹配，返回默认 Allow
   │
5. 返回 JSON 响应
```

### 2.3.2 Update 请求流程

```
1. Client 发送 POST /api/v1/update
   │
2. Gin Router 路由到 UpdateHandler
   │
3. Handler 解析请求参数
   │
4. 调用 RuleEngine.Update(request)
   │
   ├─→ 遍历所有匹配的规则
   │   └─→ 调用对应算法的 Update 方法
   │       ├─→ Count: INCR 计数器
   │       ├─→ Base: INCR 主计数器 + 条件更新次级计数器
   │       └─→ Leak: LPUSH 时间戳
   │
5. 返回成功响应（异步更新，不阻塞）
```

### 2.3.3 热重载流程

```
1. fsnotify 检测到配置文件变化
   │
2. 读取并解析新配置
   │
3. 验证配置有效性
   │   ├─→ 失败: 记录错误日志，保留旧配置
   │   └─→ 成功: 继续
   │
4. 加载字典文件
   │
5. 构建新的 RuleSet 对象
   │
6. atomic.Pointer.Store(newRuleSet)
   │
7. 旧 RuleSet 等待 GC 回收
```

## 2.4 目录结构

```
koala-v2/
├── cmd/
│   └── koala/
│       └── main.go                 # 程序入口
│
├── internal/                        # 内部包（不对外暴露）
│   ├── api/
│   │   ├── handler.go              # HTTP Handler（Browse/Update/Batch）
│   │   ├── health.go               # 健康检查（HealthChecker/HealthManager/ReadinessManager）
│   │   ├── middleware.go           # 中间件（日志/恢复/指标/CORS/超时/限流）
│   │   ├── router.go              # 路由定义与 Server 封装
│   │   └── types.go               # 请求/响应类型定义与 Engine 接口
│   │
│   ├── engine/
│   │   ├── engine.go              # 规则引擎（Check/Browse/Update）
│   │   ├── rule.go                # 规则定义、匹配、规则集与构建
│   │   ├── request.go             # 请求/响应类型定义
│   │   ├── dict_bridge.go         # 字典桥接（SyncDictsToMatcher）
│   │   ├── matcher/               # 规则匹配器
│   │   │   ├── matcher.go         # 匹配器接口、Parse 分发与字典注册
│   │   │   ├── any.go             # 通配符匹配（AnyMatcher）
│   │   │   ├── dict.go            # 字典匹配（DictMatcher）
│   │   │   ├── exact.go           # 精确匹配（ExactMatcher）
│   │   │   ├── greater.go         # 大于比较（GreaterMatcher）
│   │   │   ├── ip.go              # IP/CIDR 匹配（IPMatcher）
│   │   │   ├── less.go            # 小于比较（LessMatcher）
│   │   │   ├── multi.go           # 多值匹配（MultiMatcher）
│   │   │   ├── not.go             # 取反匹配（NotMatcher）
│   │   │   └── range.go           # 范围匹配（RangeMatcher）
│   │   └── algorithm/
│   │       ├── interface.go       # 算法接口与 LimitConfig
│   │       ├── direct.go          # Direct 算法（访问控制）
│   │       ├── count.go           # Count 算法（type="count"）
│   │       ├── base.go            # Base 算法（type="accumulate"）
│   │       └── leak.go            # Leak 算法（type="freq"）
│   │
│   ├── storage/
│   │   ├── interface.go            # 存储接口定义
│   │   ├── manager/
│   │   │   └── manager.go          # 存储管理器（含故障转移与健康检查）
│   │   ├── redis/
│   │   │   └── redis.go            # Redis 单机实现
│   │   └── local/
│   │       └── local.go            # 本地内存存储（Ristretto + 计数器 + 列表）
│   │
│   └── config/
│       ├── config.go               # 服务配置结构定义与加载（LoadConfig）
│       ├── rules.go                # 规则配置结构定义与加载（LoadRules）、TOML 解析
│       ├── watcher.go              # 文件监听器（Watcher/ConfigWatcher）
│       ├── validate.go             # 配置验证器（Validator/ValidateConfig/ValidateRules）
│       └── dict.go                 # 字典加载器
│
├── pkg/                             # 可复用的公共包
│   ├── logger/
│   │   └── logger.go               # log/slog 封装
│   └── errors/
│       └── errors.go               # InternalError 错误类型（区分内部/外部消息）
│
├── conf/                            # 配置文件
│   ├── koala.toml                  # 服务配置
│   ├── koala-redis.toml            # Redis 模式服务配置
│   ├── rules.toml                  # 规则配置
│   ├── uid_whitelist.txt           # 用户白名单
│   ├── uid_blacklist.txt           # 用户黑名单
│   ├── ip_whitelist.txt            # IP 白名单
│   ├── ip_blacklist.txt            # IP 黑名单
│   ├── whitelist_users.txt         # 用户白名单（别名）
│   └── blacklist_ips.txt           # IP 黑名单（别名）
│
├── docs/                            # 文档
│   └── v2-tech-design/
│       ├── 00-INDEX.md
│       ├── 01-PRODUCT-OVERVIEW.md
│       ├── 02-ARCHITECTURE.md
│       ├── 03-CORE-ALGORITHMS.md
│       ├── 04-API-REFERENCE.md
│       ├── 05-CONFIG-REFERENCE.md
│       ├── 06-IMPLEMENTATION-GUIDE.md
│       └── conf/
│           ├── koala.toml
│           └── rules.toml
│
├── scripts/                         # 脚本
│   └── fetch_deps.sh               # 第三方依赖拉取脚本
│
├── deployments/                     # 部署配置
│   ├── Dockerfile
│   └── docker-compose.yml
│
├── test/                            # 测试
│   ├── api/                        # API 集成测试
│   │   ├── browse_test.go          # Browse 接口测试
│   │   ├── update_test.go          # Update 接口测试
│   │   ├── batch_test.go           # Batch 接口测试
│   │   ├── health_test.go          # Health 接口测试
│   │   ├── stress_test.go          # 压力测试
│   │   ├── testutil.go             # 测试工具函数
│   │   └── testdata/               # 测试数据
│   │       ├── koala.toml
│   │       ├── rules.toml
│   │       ├── whitelist_users.txt
│   │       └── blacklist_ips.txt
│   ├── integration/                # 集成测试
│   │   └── config_test.go
│   └── benchmark/                  # 性能测试
│       └── benchmark.go
│
├── go.mod                           # Go 模块定义
├── go.sum                           # 依赖校验
├── Makefile                         # 构建命令
└── README.md                        # 项目说明
```

## 2.5 依赖管理

### 2.5.1 标准 Go Module 依赖管理

项目使用标准的 Go Modules 进行依赖管理，所有依赖通过 `go.mod` 和 `go.sum` 管理，
无 `third_party/` 目录，无 `replace` 指令。

### 2.5.2 go.mod 配置

```go
module koala

go 1.24.0

require (
    github.com/BurntSushi/toml v1.6.0
    github.com/dgraph-io/ristretto/v2 v2.4.0
    github.com/fsnotify/fsnotify v1.9.0
    github.com/gin-gonic/gin v1.11.0
    github.com/redis/go-redis/v9 v9.17.3
    github.com/stretchr/testify v1.11.1
)
```

### 2.5.3 核心直接依赖

| 依赖 | 版本 | 说明 |
|------|------|------|
| github.com/BurntSushi/toml | v1.6.0 | TOML 配置文件解析 |
| github.com/dgraph-io/ristretto/v2 | v2.4.0 | 高性能本地缓存（LocalStorage 使用） |
| github.com/fsnotify/fsnotify | v1.9.0 | 文件系统事件监听（配置热重载） |
| github.com/gin-gonic/gin | v1.11.0 | HTTP Web 框架 |
| github.com/redis/go-redis/v9 | v9.17.3 | Redis 客户端 |
| github.com/stretchr/testify | v1.11.1 | 测试断言库 |

> **说明**:
> - 日志使用标准库 `log/slog`，不依赖 zap 或其他第三方日志库。
> - 指标采集在 `middleware.go` 中内联实现，不依赖 prometheus/client_golang。
> - 故障转移逻辑内置于 `internal/storage/manager/manager.go`，不依赖独立的熔断器库。
> - `scripts/fetch_deps.sh` 是一个可选的依赖拉取脚本，用于离线环境将依赖源码拉取到本地参考，
>   但项目实际构建不依赖 `third_party/` 目录。

## 2.6 核心数据结构

### 2.6.1 RuleSet（规则集）

```go
// internal/engine/rule.go

// RuleSet 规则集合，按阶段组织规则。
type RuleSet struct {
    Whitelist []*Rule // 白名单规则
    Blacklist []*Rule // 黑名单规则
    Business  []*Rule // 业务规则（优先级 1）
    Post      []*Rule // 发帖规则（优先级 2）
    Advanced  []*Rule // 高级规则（优先级 3）
    Default   []*Rule // 默认规则（优先级 4）
}
```

### 2.6.2 Rule（规则）

config 层（`internal/config/rules.go`）的规则结构：

```go
// config.RateRule — 从 TOML 文件直接解码的规则
type RateRule struct {
    Name   string            `toml:"name"`
    Type   string            `toml:"type"`   // 算法类型: count/freq/accumulate
    Match  map[string]string `toml:"match"`
    Limit  Limit             `toml:"limit"`
    Result string            `toml:"result"` // 结果模板引用
    Desc   string            `toml:"desc"`
}

// config.Limit — TOML 解码的限流参数（Count/Base 为 int）
type Limit struct {
    Time  time.Duration `toml:"time"`
    Count int           `toml:"count"`
    Base  int           `toml:"base"`  // 累积阈值（仅 accumulate 类型使用）
}
```

engine 层（`internal/engine/rule.go`）的规则结构：

```go
// engine.LimitConfig — 引擎内部使用的限流配置（Count/Base 为 int64）
type LimitConfig struct {
    Time  time.Duration // 时间窗口
    Count int64         // 窗口内最大请求数
    Base  int64         // 累积阈值（仅 Base 算法使用）
}
```

> **说明**: `config.Limit`（`int`）→ `engine.LimitConfig`（`int64`）的类型转换
> 在 `buildRateRule()` 函数中完成：`Count: int64(rr.Limit.Count), Base: int64(rr.Limit.Base)`。

### 2.6.3 Result（结果）

```go
// internal/config/rules.go
type Result struct {
    Code     int    `toml:"code"`      // 结果码
    Message  string `toml:"message"`   // 结果消息
    AuthType int    `toml:"auth_type"` // 认证类型（0=无，1=滑块...）
}
```

### 2.6.4 Request/Response

```go
// 引擎请求（internal/engine/request.go）
type Request struct {
    Act string            // 行为类型（如 login, register, post）
    UID string            // 用户ID
    IP  string            // 客户端IP地址
    DID string            // 设备ID
    Ext map[string]string // 扩展参数（用于自定义匹配条件）
}

// 引擎响应
type Response struct {
    Allowed  bool   // 是否允许
    Code     int    // 响应码（0=允许，其他=拒绝原因）
    Message  string // 响应消息
    RuleName string // 命中的规则名称（如果有）
    AuthType int    // 验证类型（0=无验证，1=滑块验证，2=短信验证等）
}
```

## 2.7 字典桥接

### 2.7.1 SyncDictsToMatcher

`SyncDictsToMatcher` 是连接 `config.DictManager` 和 `matcher.DictMatcher` 的桥梁函数，
定义在 `internal/engine/dict_bridge.go` 中：

```go
// SyncDictsToMatcher 将 DictManager 中的字典同步到 matcher 的 DictMatcher。
func SyncDictsToMatcher(dicts *config.DictManager) {
    if dicts == nil {
        return
    }
    for _, name := range dicts.List() {
        dict, ok := dicts.Get(name)
        if ok {
            entries := dict.List()
            dictMap := make(map[string]bool, len(entries))
            for _, entry := range entries {
                dictMap[entry] = true
            }
            matcher.RegisterDict(name, dictMap)
        }
    }
}
```

在以下场景中调用：
- 服务启动时加载字典后
- 配置热重载检测到字典文件变更时

## 2.8 优雅关闭设计

### 2.8.1 Server.Shutdown

`api.Server` 封装了 `http.Server`，提供 `Shutdown` 方法：

```go
type Server struct {
    router     *gin.Engine
    addr       string
    httpServer *http.Server
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx)
}
```

### 2.8.2 信号处理与超时

`cmd/koala/main.go` 中通过信号监听实现优雅关闭：

```go
// 等待中断信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// 使用配置的 ShutdownTimeout 优雅关闭
ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    logger.Error("服务器关闭错误", "error", err)
}
```

> **说明**: `ShutdownTimeout` 在 `koala.toml` 的 `[server]` 中配置，默认 30 秒。
> 在超时时间内，服务器会等待正在处理的请求完成。

## 2.9 并发安全设计

### 2.9.1 规则热重载

使用 `atomic.Pointer` 实现无锁读取：

```go
type Engine struct {
    rules   atomic.Pointer[RuleSet]     // 原子指针，支持热加载
    storage storage.Storage             // 存储后端
    dicts   *config.DictManager         // 字典管理器
    mu      sync.RWMutex               // 保护 storage 和 dicts 字段
}

// Check/Browse 是对 matchRules 的轻量包装，仅通过 updateOnMatch 参数区分行为。
// Check 在匹配到限流规则且未触发限制时更新计数器；Browse 仅查询状态，不更新。
func (e *Engine) Check(ctx context.Context, req *Request) (*Response, error) {
    ruleSet := e.rules.Load()  // 原子读取
    return e.matchRules(ctx, ruleSet, req, true)
}

func (e *Engine) Browse(ctx context.Context, req *Request) (*Response, error) {
    ruleSet := e.rules.Load()  // 原子读取
    return e.matchRules(ctx, ruleSet, req, false)
}

// matchRules 是核心规则匹配方法，Check 和 Browse 均委托给它。
// updateOnMatch 控制匹配到限流规则且未触发限制时是否更新计数器。
// 执行流程：阶段1（白名单→黑名单），阶段2（业务→发帖→高级→默认），
// 首次匹配即返回结果。
func (e *Engine) matchRules(ctx context.Context, ruleSet *RuleSet, req *Request,
    updateOnMatch bool) (*Response, error) {
    // ...
}

// 重载（后台协程）
func (e *Engine) LoadRules(rulesConfig *config.RulesConfig) error {
    ruleSet, err := BuildRuleSet(rulesConfig)
    if err != nil {
        return err
    }
    e.rules.Store(ruleSet)  // 原子写入
    return nil
}
```

### 2.9.2 Engine 字段保护

使用 `sync.RWMutex` 保护 `storage` 和 `dicts` 字段的并发访问：

```go
// getStorage 获取当前存储实例（并发安全）。
func (e *Engine) getStorage() storage.Storage {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.storage
}

// getDicts 获取当前字典管理器（并发安全）。
func (e *Engine) getDicts() *config.DictManager {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.dicts
}

// SetStorage 设置存储后端。
func (e *Engine) SetStorage(s storage.Storage) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.storage = s
}
```

> **设计说明**: 规则集使用 `atomic.Pointer` 实现无锁读取（高频路径），
> 而 storage/dicts 使用 `sync.RWMutex`（低频写入、可并发读取）。

### 2.9.3 存储降级

使用 `atomic.Pointer[storage.Storage]` 存储当前存储实例，配合健康检查实现自动故障转移：

```go
// internal/storage/manager/manager.go
type StorageManager struct {
    primary  storage.Storage
    fallback storage.Storage

    current atomic.Pointer[storage.Storage]

    failures     int32
    maxFailures  int32
    degraded     atomic.Bool
    closed       atomic.Bool

    healthCheckInterval time.Duration
    recoveryInterval    time.Duration

    mu     sync.RWMutex
    stopCh chan struct{}
    wg     sync.WaitGroup
}

func (m *StorageManager) Get(ctx context.Context, key string) (string, error) {
    return m.executeWithFailoverString(func(s storage.Storage) (string, error) {
        return s.Get(ctx, key)
    })
}

// getCurrent 通过原子指针获取当前存储实例
func (m *StorageManager) getCurrent() storage.Storage {
    return *m.current.Load()
}
```

### 2.9.4 本地计数器

本地存储中计数器使用未导出的 `counterEntry` 结构体，包含 `atomic.Int64` 和过期时间：

```go
// internal/storage/local/local.go
type counterEntry struct {
    value   atomic.Int64
    expires time.Time   // 过期时间，零值表示永不过期
}
```

计数器操作通过 `LocalStorage` 的方法实现，使用 `sync.RWMutex` 保护 map 访问，
`atomic.Int64.Add()` 实现无锁原子自增：

```go
func (s *LocalStorage) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
    s.countersMu.Lock()
    entry, exists := s.counters[key]
    // 检查是否过期
    if exists && !entry.expires.IsZero() && time.Now().After(entry.expires) {
        delete(s.counters, key)
        exists = false
    }
    if !exists {
        entry = &counterEntry{}
        s.counters[key] = entry
    }
    s.countersMu.Unlock()

    newVal := entry.value.Add(delta)
    return newVal, nil
}
```

### 2.9.5 请求ID生成

`requestIDCounter` 使用 `atomic.Uint64` 实现无锁自增，避免互斥锁开销：

```go
var requestIDCounter atomic.Uint64

func generateRequestID() string {
    id := requestIDCounter.Add(1)
    return time.Now().Format("20060102150405") + "-" + uintToString(id)
}
```
