# Koala V2

高性能频率控制与反作弊系统

## 快速开始

### 1. 编译

```bash
make build
```

编译产物输出到 `bin/koala`。

### 2. 运行

```bash
# 使用 Makefile（编译+运行）
make run

# 或直接运行
./bin/koala -config conf/koala.toml
```

默认监听端口 `:9981`（在 `conf/koala.toml` 中配置）。

### 3. 验证

```bash
# 健康检查
curl http://localhost:9981/health

# 检查请求
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act": "comment", "uid": "12345"}'

# 记录请求（更新计数器）
curl -X POST http://localhost:9981/api/v1/update \
  -H "Content-Type: application/json" \
  -d '{"act": "comment", "uid": "12345"}'
```

### 4. 测试

```bash
# 全量测试
make test

# 覆盖率测试
make test-coverage

# 性能测试
make bench
```

## 目录结构

```
koala-v2/
├── cmd/koala/          # 程序入口
├── internal/           # 内部包
│   ├── api/           # HTTP 层（路由、中间件、Handler）
│   ├── engine/        # 规则引擎（匹配、算法）
│   │   ├── algorithm/ # 限流算法（Count/Base/Leak/Direct）
│   │   └── matcher/   # 条件匹配器（精确/通配/IP/字典等）
│   ├── storage/       # 存储层
│   │   ├── local/     # 本地存储（Ristretto 缓存）
│   │   ├── redis/     # Redis 存储
│   │   └── manager/   # 存储管理（自动故障转移）
│   └── config/        # 配置管理（加载、验证、热重载）
├── pkg/               # 公共包
│   ├── logger/        # 结构化日志（基于 log/slog）
│   └── errors/        # 错误类型（内部/外部消息分离）
├── conf/              # 配置文件
├── bin/               # 编译产物（make build 输出）
├── test/              # 集成测试与性能测试
├── docs/              # 文档
├── scripts/           # 脚本
└── deployments/       # 部署配置
```

## 文档

| 文档 | 说明 |
|------|------|
| [技术设计文档](docs/v2-tech-design/00-INDEX.md) | 架构、算法、API、配置参考 |
| [用户手册](docs/user_manual.md) | 完整使用指南 |
| [快速参考卡](docs/quick_reference.md) | 常用命令和 API 速查 |

## 配置

- 服务配置: `conf/koala.toml`
- 规则配置: `conf/rules.toml`

存储模式通过 `[storage]` 的 `type` 字段切换（默认 `"local"`，可选 `"redis"`）。

## API

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/v1/browse | POST | 检查请求是否允许（可选 `update:true` 自动更新计数器） |
| /api/v1/update | POST | 记录请求，递增计数器 |
| /api/v1/batch | POST | 批量检查（只读，不更新计数器） |
| /health | GET | 服务存活探测（Liveness） |
| /ready | GET | 服务就绪探测（Readiness） |
| /metrics | GET | Prometheus 格式指标 |

## 依赖

项目使用标准 Go Modules 管理依赖，主要依赖：

- `github.com/gin-gonic/gin` — HTTP 框架
- `github.com/dgraph-io/ristretto/v2` — 本地高性能缓存
- `github.com/redis/go-redis/v9` — Redis 客户端
- `github.com/BurntSushi/toml` — TOML 配置解析
- `github.com/fsnotify/fsnotify` — 文件变更监听（热重载）

## License

MIT
