# 06 - 实现指南

## 6.1 开发环境

### 6.1.1 环境要求

| 组件 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 编程语言 |
| Redis | 6.0+ | 存储（可选） |
| Make | - | 构建工具 |
| Docker | 20.0+ | 容器化（可选） |

### 6.1.2 环境搭建

```bash
# 1. 克隆项目
cd /Users/black/Documents/Projects/gil/be/gil_koala/koala-v2

# 2. 初始化 Go 模块
go mod init koala
go mod tidy

# 3. 拉取第三方依赖到本地（见 6.2 节）

# 4. 验证编译
go build -o bin/koala ./cmd/koala

# 5. 运行测试
go test ./...
```

### 6.1.3 IDE 配置

**VSCode settings.json:**
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "gopls": {
        "build.directoryFilters": ["-third_party"]
    }
}
```

## 6.2 第三方依赖本地化

### 6.2.1 依赖列表

| 依赖 | 版本 | 原始仓库 | 用途 |
|------|------|---------|------|
| ristretto | v0.1.1 | github.com/dgraph-io/ristretto | 本地缓存 |
| zap | v1.26.0 | go.uber.org/zap | 日志 |
| redis | v9.3.0 | github.com/redis/go-redis/v9 | Redis 客户端 |
| gin | v1.9.1 | github.com/gin-gonic/gin | HTTP 框架 |
| fsnotify | v1.7.0 | github.com/fsnotify/fsnotify | 文件监听 |
| toml | v1.3.2 | github.com/BurntSushi/toml | TOML 解析 |
| prometheus | v1.17.0 | github.com/prometheus/client_golang | 指标 |

### 6.2.2 拉取脚本

```bash
#!/bin/bash
# scripts/fetch_deps.sh

set -e

THIRD_PARTY_DIR="third_party"
mkdir -p $THIRD_PARTY_DIR
cd $THIRD_PARTY_DIR

echo "Fetching ristretto..."
git clone --depth 1 --branch v2.4.0 \
    https://github.com/dgraph-io/ristretto.git ristretto
rm -rf ristretto/.git

echo "Fetching zap..."
git clone --depth 1 --branch v1.27.1 \
    https://github.com/uber-go/zap.git zap
rm -rf zap/.git

echo "Fetching go-redis..."
git clone --depth 1 --branch v9.17.3 \
    https://github.com/redis/go-redis.git redis
rm -rf redis/.git

echo "Done!"
```

### 6.2.3 go.mod 配置

```go
module koala

go 1.21

// 将依赖指向本地目录
replace (
    github.com/dgraph-io/ristretto => ./third_party/ristretto
    go.uber.org/zap => ./third_party/zap
    github.com/redis/go-redis/v9 => ./third_party/redis
)

require (
    github.com/dgraph-io/ristretto v0.1.1
    github.com/gin-gonic/gin v1.9.1
    github.com/redis/go-redis/v9 v9.3.0
    go.uber.org/zap v1.26.0
    github.com/fsnotify/fsnotify v1.7.0
    github.com/BurntSushi/toml v1.3.2
    github.com/prometheus/client_golang v1.17.0
)
```

## 6.3 实现顺序

### 6.3.1 推荐顺序

```
Phase 1: 基础设施
├── 1.1 项目结构搭建
├── 1.2 配置加载 (config/)
├── 1.3 日志系统 (pkg/logger/)
└── 1.4 存储接口定义 (storage/interface.go)

Phase 2: 存储层
├── 2.1 本地存储实现 (storage/local/)
│   ├── Ristretto 适配
│   ├── Counter 存储
│   └── List 存储
├── 2.2 Redis 存储实现 (storage/redis/)
└── 2.3 存储管理器 (storage/manager.go)

Phase 3: 规则引擎
├── 3.1 数据结构定义 (engine/policy.go)
├── 3.2 匹配器实现 (engine/matcher.go)
├── 3.3 算法实现 (engine/algorithm/)
│   ├── Direct
│   ├── Count
│   ├── Base
│   └── Leak
└── 3.4 引擎整合 (engine/engine.go)

Phase 4: HTTP 层
├── 4.1 Handler 实现 (api/handler/)
├── 4.2 中间件 (api/middleware/)
├── 4.3 路由定义 (api/router.go)
└── 4.4 主程序 (cmd/koala/main.go)

Phase 5: 完善
├── 5.1 Prometheus 指标
├── 5.2 热重载
├── 5.3 单元测试
└── 5.4 集成测试
```

### 6.3.2 依赖关系图

```
                    ┌─────────────┐
                    │   config    │
                    └─────────────┘
                          │
            ┌─────────────┼─────────────┐
            ▼             ▼             ▼
      ┌──────────┐ ┌──────────┐ ┌──────────┐
      │  logger  │ │ storage  │ │  engine  │
      └──────────┘ └──────────┘ └──────────┘
            │             │             │
            └─────────────┼─────────────┘
                          ▼
                    ┌──────────┐
                    │   api    │
                    └──────────┘
                          │
                          ▼
                    ┌──────────┐
                    │   main   │
                    └──────────┘
```

## 6.4 关键实现要点

### 6.4.1 配置加载

```go
// internal/config/config.go

package config

import (
    "time"

    "github.com/BurntSushi/toml"
)

type Config struct {
    Server   ServerConfig   `toml:"server"`
    Rules    RulesConfig    `toml:"rules"`
    Storage  StorageConfig  `toml:"storage"`
    Logging  LoggingConfig  `toml:"logging"`
    Metrics  MetricsConfig  `toml:"metrics"`
}

type ServerConfig struct {
    Listen          string        `toml:"listen"`
    ReadTimeout     time.Duration `toml:"read_timeout"`
    WriteTimeout    time.Duration `toml:"write_timeout"`
    ShutdownTimeout time.Duration `toml:"shutdown_timeout"`
}

// ... 其他配置结构

func Load(path string) (*Config, error) {
    var cfg Config
    if _, err := toml.DecodeFile(path, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### 6.4.2 存储接口

```go
// internal/storage/interface.go

package storage

import (
    "context"
    "errors"
    "time"
)

var (
    ErrKeyNotFound     = errors.New("key not found")
    ErrIndexOutOfRange = errors.New("index out of range")
)

type Storage interface {
    // String 操作
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value string, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Expire(ctx context.Context, key string, ttl time.Duration) error

    // 计数器操作
    GetInt(ctx context.Context, key string) (int64, error)
    SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error
    Incr(ctx context.Context, key string) (int64, error)
    IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error)

    // List 操作 (Leak 算法)
    LPush(ctx context.Context, key string, values ...int64) error
    LLen(ctx context.Context, key string) (int64, error)
    LIndex(ctx context.Context, key string, index int64) (int64, error)
    LTrim(ctx context.Context, key string, start, stop int64) error

    // 管理操作
    Ping(ctx context.Context) error
    Close() error
}
```

### 6.4.3 规则引擎

```go
// internal/engine/engine.go

package engine

import (
    "context"
    "sync/atomic"
)

type RuleEngine struct {
    policy  atomic.Pointer[Policy]
    storage storage.Storage
}

func New(storage storage.Storage) *RuleEngine {
    return &RuleEngine{
        storage: storage,
    }
}

func (e *RuleEngine) LoadPolicy(policy *Policy) {
    e.policy.Store(policy)
}

func (e *RuleEngine) Evaluate(ctx context.Context, req *Request) *Result {
    policy := e.policy.Load()
    if policy == nil {
        return &Result{Allowed: true, Code: 0, Message: "Allow"}
    }

    // Phase 1: 访问控制
    // 检查白名单
    for _, rule := range policy.Access.Whitelist {
        if rule.Match(req) {
            return policy.GetResult(rule.Result)
        }
    }

    // 检查黑名单
    for _, rule := range policy.Access.Blacklist {
        if rule.Match(req) {
            return policy.GetResult(rule.Result)
        }
    }

    // Phase 2: 频率控制（按优先级）
    ruleGroups := [][]RateRule{
        policy.Rate.Business,
        policy.Rate.Post,
        policy.Rate.Advanced,
        policy.Rate.Default,
    }

    for _, rules := range ruleGroups {
        for _, rule := range rules {
            if rule.Match(req) {
                hit, err := rule.Check(ctx, req, e.storage)
                if err != nil {
                    // 存储错误，放行但记录日志
                    continue
                }
                if hit {
                    result := policy.GetResult(rule.Result)
                    result.RuleName = rule.Name
                    return result
                }
            }
        }
    }

    // Phase 3: 默认放行
    return &Result{Allowed: true, Code: 0, Message: "Allow"}
}

func (e *RuleEngine) Update(ctx context.Context, req *Request) error {
    policy := e.policy.Load()
    if policy == nil {
        return nil
    }

    // 遍历所有规则，更新匹配的
    allRules := append(policy.Rate.Business, policy.Rate.Post...)
    allRules = append(allRules, policy.Rate.Advanced...)
    allRules = append(allRules, policy.Rate.Default...)

    for _, rule := range allRules {
        if rule.Match(req) {
            _ = rule.Update(ctx, req, e.storage)
        }
    }

    return nil
}
```

### 6.4.4 主程序

```go
// cmd/koala/main.go

package main

import (
    "context"
    "flag"
    "os"
    "os/signal"
    "syscall"
    "time"

    "koala/internal/api"
    "koala/internal/config"
    "koala/internal/engine"
    "koala/internal/storage"
    "koala/pkg/logger"
)

func main() {
    configPath := flag.String("config", "conf/koala.toml", "config file path")
    flag.Parse()

    // 1. 加载配置
    cfg, err := config.Load(*configPath)
    if err != nil {
        panic("failed to load config: " + err.Error())
    }

    // 2. 初始化日志
    log := logger.New(cfg.Logging)
    defer log.Sync()

    // 3. 初始化存储
    store, err := storage.NewManager(cfg.Storage)
    if err != nil {
        log.Fatal("failed to init storage", "error", err)
    }
    defer store.Close()

    // 4. 初始化规则引擎
    ruleEngine := engine.New(store)

    // 5. 加载规则
    policy, err := config.LoadRules(cfg.Rules.File)
    if err != nil {
        log.Fatal("failed to load rules", "error", err)
    }
    ruleEngine.LoadPolicy(policy)

    // 6. 启动配置监听
    go config.Watch(cfg.Rules.File, func(newPolicy *engine.Policy) {
        ruleEngine.LoadPolicy(newPolicy)
        log.Info("policy reloaded")
    })

    // 7. 启动 HTTP 服务
    router := api.NewRouter(ruleEngine)
    srv := &http.Server{
        Addr:         cfg.Server.Listen,
        Handler:      router,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }

    go func() {
        log.Info("server starting", "addr", cfg.Server.Listen)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("server error", "error", err)
        }
    }()

    // 8. 优雅关闭
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Info("shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Error("shutdown error", "error", err)
    }

    log.Info("server stopped")
}
```

## 6.5 测试策略

### 6.5.1 单元测试

```go
// internal/engine/algorithm/count_test.go

package algorithm

import (
    "context"
    "testing"
    "time"

    "koala/internal/storage/local"
)

func TestCountAlgorithm_Browse(t *testing.T) {
    store, _ := local.New(local.DefaultConfig())
    defer store.Close()

    algo := NewCount()
    ctx := context.Background()
    key := "test:count:1"
    limit := LimitConfig{Time: time.Minute, Count: 3}

    // 首次检查，应该放行
    hit, err := algo.Browse(ctx, key, limit, store)
    if err != nil {
        t.Fatal(err)
    }
    if hit {
        t.Error("expected not hit on first browse")
    }

    // 更新 3 次
    for i := 0; i < 3; i++ {
        algo.Update(ctx, key, limit, store)
    }

    // 再次检查，应该命中
    hit, err = algo.Browse(ctx, key, limit, store)
    if err != nil {
        t.Fatal(err)
    }
    if !hit {
        t.Error("expected hit after 3 updates")
    }
}
```

### 6.5.2 集成测试

```go
// test/integration/api_test.go

package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "koala/internal/api"
    "koala/internal/engine"
    "koala/internal/storage/local"
)

func TestBrowseAPI(t *testing.T) {
    // 初始化
    store, _ := local.New(local.DefaultConfig())
    eng := engine.New(store)
    router := api.NewRouter(eng)

    // 发送请求
    body := map[string]string{"act": "test", "uid": "123"}
    jsonBody, _ := json.Marshal(body)

    req := httptest.NewRequest("POST", "/api/v1/browse", bytes.NewReader(jsonBody))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // 验证响应
    if w.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", w.Code)
    }

    var resp map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &resp)

    if resp["allowed"] != true {
        t.Error("expected allowed=true")
    }
}
```

### 6.5.3 性能测试

```go
// test/benchmark/browse_test.go

package benchmark

import (
    "context"
    "testing"

    "koala/internal/engine"
    "koala/internal/storage/local"
)

func BenchmarkBrowse(b *testing.B) {
    store, _ := local.New(local.DefaultConfig())
    eng := engine.New(store)

    // 加载测试策略
    eng.LoadPolicy(testPolicy)

    req := &engine.Request{
        Act: "test",
        UID: "12345",
        IP:  "192.168.1.1",
    }

    ctx := context.Background()

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            eng.Evaluate(ctx, req)
        }
    })
}
```

## 6.6 构建与部署

### 6.6.1 Makefile

```makefile
# Makefile

.PHONY: build test clean

# 变量
BINARY=koala
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# 构建
build:
	go build ${LDFLAGS} -o bin/${BINARY} ./cmd/koala

# 测试
test:
	go test -v -race ./...

# 性能测试
bench:
	go test -bench=. -benchmem ./test/benchmark/

# 清理
clean:
	rm -rf bin/

# 运行
run: build
	./bin/${BINARY} -config conf/koala.toml

# Docker 构建
docker:
	docker build -t koala:${VERSION} -f deployments/Dockerfile .

# 拉取依赖
deps:
	bash scripts/fetch_deps.sh
```

### 6.6.2 Dockerfile

```dockerfile
# deployments/Dockerfile

FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖
COPY go.mod go.sum ./
COPY third_party/ third_party/
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -o koala ./cmd/koala

# 运行镜像
FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/koala .
COPY conf/ conf/

EXPOSE 9981

CMD ["./koala", "-config", "conf/koala.toml"]
```

### 6.6.3 docker-compose.yml

```yaml
# deployments/docker-compose.yml

version: '3.8'

services:
  koala:
    build:
      context: ..
      dockerfile: deployments/Dockerfile
    ports:
      - "9981:9981"
    volumes:
      - ../conf:/app/conf
      - ../logs:/app/logs
    depends_on:
      - redis
    environment:
      - TZ=Asia/Shanghai

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  redis_data:
```

## 6.7 常见问题

### Q1: 第三方依赖更新后如何同步？

```bash
# 1. 进入 third_party 目录
cd third_party/ristretto

# 2. 更新到指定版本
git fetch origin
git checkout v0.1.2

# 3. 删除 .git（可选）
rm -rf .git

# 4. 测试编译
cd ../..
go build ./...
```

### Q2: 本地存储内存不足怎么办？

调整 `storage.local.max_size` 配置，或启用 Redis 作为主存储。

### Q3: 热重载不生效？

1. 检查文件权限
2. 检查 fsnotify 是否支持当前系统
3. 查看日志中的重载记录

### Q4: 规则匹配性能优化？

1. 减少规则数量
2. 将高频匹配的规则放在前面
3. 使用更精确的匹配条件
