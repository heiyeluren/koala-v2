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
│  │                      Policy (atomic.Pointer)                   │      │
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
│  │                    Circuit Breaker                               │    │
│  │         ┌──────────────────┬──────────────────┐                 │    │
│  │         ▼                  ▼                  ▼                 │    │
│  │  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │    │
│  │  │    Redis    │   │   Redis     │   │    Local    │           │    │
│  │  │   Single    │   │   Cluster   │   │   Storage   │           │    │
│  │  └─────────────┘   └─────────────┘   └─────────────┘           │    │
│  │                                              │                  │    │
│  │                           ┌──────────────────┼──────────────┐  │    │
│  │                           ▼                  ▼              ▼  │    │
│  │                    ┌───────────┐      ┌───────────┐  ┌────────┐│    │
│  │                    │ Ristretto │      │  Counter  │  │  List  ││    │
│  │                    │  (String) │      │   Store   │  │ Store  ││    │
│  │                    └───────────┘      └───────────┘  └────────┘│    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Config Watcher                                   │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  fsnotify → Parse → Validate → atomic.Pointer.Store(newPolicy)  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

## 2.2 模块职责

### 2.2.1 HTTP Layer

| 模块 | 职责 |
|------|------|
| Router | 路由注册与分发 |
| Logger Middleware | 请求日志记录 |
| Recovery Middleware | panic 恢复 |
| Metrics Middleware | Prometheus 指标采集 |

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
| Policy | 存储当前生效的规则集（原子指针） |
| AccessRules | 白名单/黑名单规则 |
| RateRules | 频率控制规则（按优先级分组） |
| Algorithms | 四种限流算法实现 |
| Matcher | 规则匹配器 |

### 2.2.4 Storage Manager

| 组件 | 职责 |
|------|------|
| StorageManager | 存储统一入口，管理主备切换 |
| CircuitBreaker | 熔断器，检测故障并触发降级 |
| RedisStorage | Redis 存储实现 |
| LocalStorage | 本地存储实现 |

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
5. 构建新的 Policy 对象
   │
6. atomic.Pointer.Store(newPolicy)
   │
7. 旧 Policy 等待 GC 回收
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
│   │   ├── handler/
│   │   │   ├── browse.go           # Browse 处理器
│   │   │   ├── update.go           # Update 处理器
│   │   │   ├── batch.go            # 批量处理器
│   │   │   └── health.go           # 健康检查
│   │   ├── middleware/
│   │   │   ├── logger.go           # 日志中间件
│   │   │   ├── recovery.go         # 恢复中间件
│   │   │   └── metrics.go          # 指标中间件
│   │   └── router.go               # 路由定义
│   │
│   ├── engine/
│   │   ├── engine.go               # 规则引擎
│   │   ├── policy.go               # 策略定义
│   │   ├── matcher.go              # 规则匹配器
│   │   ├── evaluator.go            # 规则评估器
│   │   └── algorithm/
│   │       ├── interface.go        # 算法接口
│   │       ├── direct.go           # Direct 算法（访问控制）
│   │       ├── count.go            # Count 算法（type="count"）
│   │       ├── base.go             # Base 算法（type="accumulate"）
│   │       └── leak.go             # Leak 算法（type="freq"）
│   │
│   ├── storage/
│   │   ├── interface.go            # 存储接口定义
│   │   ├── manager.go              # 存储管理器（含降级）
│   │   ├── redis/
│   │   │   ├── redis.go            # Redis 单机实现
│   │   │   └── cluster.go          # Redis 集群实现
│   │   └── local/
│   │       ├── storage.go          # 本地存储统一入口
│   │       ├── ristretto.go        # Ristretto 适配
│   │       ├── counter.go          # 原子计数器
│   │       └── list.go             # List 存储（Leak 算法）
│   │
│   └── config/
│       ├── config.go               # 配置结构定义
│       ├── loader.go               # 配置加载器
│       ├── watcher.go              # 文件监听器
│       ├── parser.go               # TOML 解析器
│       ├── validator.go            # 配置验证器
│       └── dict.go                 # 字典加载器
│
├── third_party/                     # 第三方依赖（本地化）
│   ├── ristretto/                  # dgraph-io/ristretto
│   │   ├── cache.go
│   │   ├── policy.go
│   │   ├── store.go
│   │   ├── ttl.go
│   │   └── ...
│   ├── zap/                        # uber-go/zap
│   │   ├── logger.go
│   │   ├── encoder.go
│   │   └── ...
│   └── redis/                      # go-redis/redis
│       ├── client.go
│       ├── cluster.go
│       └── ...
│
├── pkg/                             # 可复用的公共包
│   ├── logger/
│   │   └── logger.go               # Zap 封装
│   ├── metrics/
│   │   └── prometheus.go           # Prometheus 指标
│   └── circuitbreaker/
│       └── breaker.go              # 熔断器实现
│
├── conf/                            # 配置文件
│   ├── koala.toml                  # 服务配置
│   ├── rules.toml                  # 规则配置
│   ├── uid_whitelist.txt           # 用户白名单
│   ├── uid_blacklist.txt           # 用户黑名单
│   ├── ip_whitelist.txt            # IP 白名单
│   └── ip_blacklist.txt            # IP 黑名单
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
│   ├── build.sh                    # 构建脚本
│   ├── start.sh                    # 启动脚本
│   └── stop.sh                     # 停止脚本
│
├── deployments/                     # 部署配置
│   ├── Dockerfile
│   └── docker-compose.yml
│
├── test/                            # 测试
│   ├── integration/                # 集成测试
│   └── benchmark/                  # 性能测试
│
├── go.mod                           # Go 模块定义
├── go.sum                           # 依赖校验
├── Makefile                         # 构建命令
└── README.md                        # 项目说明
```

## 2.5 依赖管理

### 2.5.1 第三方依赖本地化策略

为避免远程依赖版本变更风险，关键依赖代码拉取到 `third_party/` 目录：

| 依赖 | 版本 | 原始仓库 | 说明 |
|------|------|---------|------|
| ristretto | v0.1.1 | github.com/dgraph-io/ristretto | 高性能本地缓存 |
| zap | v1.26.0 | go.uber.org/zap | 结构化日志 |
| redis | v9.3.0 | github.com/redis/go-redis/v9 | Redis 客户端 |
| fsnotify | v1.7.0 | github.com/fsnotify/fsnotify | 文件监听 |
| toml | v1.3.2 | github.com/BurntSushi/toml | TOML 解析 |

### 2.5.2 go.mod 配置

```go
module koala

go 1.21

// 使用 replace 指令将依赖指向本地目录
replace (
    github.com/dgraph-io/ristretto => ./third_party/ristretto
    go.uber.org/zap => ./third_party/zap
    github.com/redis/go-redis/v9 => ./third_party/redis
)

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/prometheus/client_golang v1.17.0
    github.com/fsnotify/fsnotify v1.7.0
    github.com/BurntSushi/toml v1.3.2
)
```

### 2.5.3 依赖拉取流程

```bash
# 1. 克隆依赖仓库到 third_party
cd third_party
git clone --depth 1 --branch v0.1.1 https://github.com/dgraph-io/ristretto.git
git clone --depth 1 --branch v1.26.0 https://github.com/uber-go/zap.git
git clone --depth 1 --branch v9.3.0 https://github.com/redis/go-redis.git redis

# 2. 移除 .git 目录（可选，转为项目代码）
rm -rf ristretto/.git zap/.git redis/.git

# 3. 修改 package 声明（如需要）
# 确保 import path 与 replace 指令一致
```

## 2.6 核心数据结构

### 2.6.1 Policy（策略）

```go
type Policy struct {
    Version     string                 // 配置版本
    LoadedAt    time.Time              // 加载时间
    Access      AccessRules            // 访问控制规则
    Rate        RateRules              // 频率控制规则
    Results     map[string]Result      // 结果模板
    Dicts       map[string]Dict        // 字典数据
}

type AccessRules struct {
    Whitelist []AccessRule           // 白名单（命中则放行）
    Blacklist []AccessRule           // 黑名单（命中则拒绝）
}

type RateRules struct {
    Business []RateRule              // 业务规则（优先级 1）
    Post     []RateRule              // 发帖规则（优先级 2）
    Advanced []RateRule              // 高级规则（优先级 3）
    Default  []RateRule              // 默认规则（优先级 4）
}
```

### 2.6.2 Rule（规则）

```go
type RateRule struct {
    Name       string                 // 规则名称
    Type       string                 // 算法类型: count/freq/accumulate
    Match      map[string]string      // 匹配条件
    Limit      LimitConfig            // 限制参数
    Result     string                 // 结果模板引用
    Desc       string                 // 规则描述
}

type LimitConfig struct {
    Time   time.Duration            // 时间窗口
    Count  int64                    // 计数限制
    Base   int64                    // 基础阈值（accumulate 算法）
}
```

### 2.6.3 Result（结果）

```go
type Result struct {
    Code     int    `json:"code"`      // 结果码
    Message  string `json:"message"`   // 结果消息
    AuthType int    `json:"auth_type"` // 认证类型（0=无，1=滑块...）
}
```

### 2.6.4 Request/Response

```go
// 请求
type EvaluateRequest struct {
    Act string            // 动作类型
    UID string            // 用户 ID
    IP  string            // 客户端 IP
    Ext map[string]string // 扩展参数
}

// 响应
type EvaluateResponse struct {
    Allowed  bool   `json:"allowed"`    // 是否允许
    Code     int    `json:"code"`       // 结果码
    Message  string `json:"message"`    // 结果消息
    RuleName string `json:"rule_name"`  // 命中规则名
    AuthType int    `json:"auth_type"`  // 认证类型
}
```

## 2.7 并发安全设计

### 2.7.1 规则热重载

使用 `atomic.Pointer` 实现无锁读取：

```go
type RuleEngine struct {
    policy atomic.Pointer[Policy]
}

// 读取（无锁，高性能）
func (e *RuleEngine) Evaluate(req *Request) *Result {
    policy := e.policy.Load()  // 原子读取
    return policy.evaluate(req)
}

// 重载（后台协程）
func (e *RuleEngine) Reload(newPolicy *Policy) {
    e.policy.Store(newPolicy)  // 原子写入
}
```

### 2.7.2 存储降级

使用 `atomic.Value` 存储当前存储实例：

```go
type StorageManager struct {
    current atomic.Value  // 当前存储
    primary Storage       // 主存储
    fallback Storage      // 备用存储
}

func (m *StorageManager) Get(key string) (string, error) {
    return m.current.Load().(Storage).Get(key)
}
```

### 2.7.3 本地计数器

使用 `atomic.Int64` 实现原子计数：

```go
type Counter struct {
    value atomic.Int64
}

func (c *Counter) Incr() int64 {
    return c.value.Add(1)
}
```
