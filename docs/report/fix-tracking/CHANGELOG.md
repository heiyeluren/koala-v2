# Koala V2 变更记录

## Phase 0: 基础设施准备

- **新建** `pkg/logger/logger.go` — 基于 log/slog 的结构化日志封装
- **新建** `pkg/logger/logger_test.go` — 19 个测试
- **新建** `pkg/errors/errors.go` — InternalError 类型，区分内部/外部消息
- **新建** `pkg/errors/errors_test.go` — 12 个测试

## Phase 1: P0 严重问题修复

### Task 1-1: 内部错误信息泄露修复

- **修改** `internal/api/handler.go` — 6 处 `err.Error()` 替换为安全消息，添加 logger 记录
  - Browse 绑定错误: `"请求参数错误: " + err.Error()` -> `"请求参数错误"`
  - Browse 引擎错误: `"内部错误: " + err.Error()` -> `"服务器内部错误"`
  - Update 绑定错误: 同上
  - Update 引擎错误: `"更新失败: " + err.Error()` -> `"服务器内部错误"`
  - Batch 绑定错误: 同上
  - Batch 引擎错误: 同上
- **修改** `internal/api/api_test.go` — 新增 4 个泄露防护测试

### Task 1-2: 日志系统 + Graceful Shutdown

- **修改** `internal/api/middleware.go` — LoggingMiddleware 和 RecoveryMiddleware 用 logger 替换 `_ = map{}`
- **修改** `internal/api/router.go` — Server 增加 `httpServer` 字段、`Shutdown()` 方法
- **修改** `cmd/koala/main.go` — 添加 graceful shutdown 逻辑（信号监听 + ShutdownTimeout）
- **修改** `internal/api/api_test.go` — 新增 TestServer_Shutdown 测试

### Task 1-3: 算法层问题修复

- **修改** `internal/config/rules.go` — Limit 结构体新增 `Base int` 字段
- **修改** `internal/engine/rule.go` — LimitConfig 新增 `Base int64`，buildRateRule 传递 Base
- **修改** `internal/engine/engine.go` — checkRateLimit/updateCounter 传递 Base 到算法层
- **修改** `internal/engine/algorithm/base.go` — totalKey 使用 IncrWithTTL，新增 clampTTL 函数
- **修改** `internal/engine/algorithm/leak.go` — 新增 keyLocks sync.Map，Browse/Update 按 key 加锁
- **修改** `internal/engine/algorithm/algorithm_test.go` — 新增 3 个测试

### Task 1-4: 字典匹配 + 错误丢弃 + Engine 并发安全

- **新建** `internal/engine/dict_bridge.go` — SyncDictsToMatcher 桥接函数
- **修改** `internal/engine/engine.go` — Engine 增加 sync.RWMutex，getStorage/getDicts 方法，错误日志记录
- **修改** `cmd/koala/main.go` — 初始加载和热重载时调用 SyncDictsToMatcher
- **修改** `internal/engine/engine_test.go` — 新增 TestEngine_DictMatching_EndToEnd、TestEngine_ConcurrentSetStorage

## Phase 2: P1 高优先级修复

### Task 2-1: RateLimitMiddleware 内存泄漏 + TimeoutMiddleware

- **修改** `internal/api/middleware.go` — RateLimitMiddleware 重构为结构体，增加后台清理 goroutine（可配置 cleanupInterval），counters 过期自动回收；TimeoutMiddleware 使用 `context.WithTimeout` + `defer cancel()`
- **修改** `internal/api/api_test.go` — 新增 4 个测试：TestRateLimitMiddleware_Cleanup、TestRateLimitMiddleware_Cleanup_ActiveNotRemoved、TestTimeoutMiddleware_SetsDeadline、TestTimeoutMiddleware_CancelsProperly

### Task 2-2: StorageManager failover 补全

- **修改** `internal/storage/manager/manager.go` — 提取 5 个通用 failover 辅助方法（executeWithFailover、executeWithFailoverString/Int64/Bool/Slice），所有 Storage 方法统一降级逻辑
- **修改** `internal/storage/manager/manager_test.go` — 新增 14 个 failover 测试（覆盖 Incr/IncrWithTTL/GetInt/LPush/LLen/LRange/Delete/Exists/Expire/SetInt/IncrBy/LIndex/LTrim/AlreadyDegraded）

### Task 2-3: LocalStorage 过期数据清理

- **修改** `internal/storage/local/local.go` — Config 增加 CleanupInterval 字段，New() 启动后台清理 goroutine，Close() 停止清理，DefaultConfig() 默认 5 分钟清理间隔
- **修改** `internal/storage/local/local_test.go` — 新增 7 个清理测试（ListCleanup、DoesNotRemoveActiveData、StopsOnClose、DisabledByDefault、DefaultConfigHasCleanupInterval、CleanupExpiredListEntries、CleanupConcurrentAccess）

### Task 2-4: 文档更新

- **修改** `docs/v2-tech-design/02-ARCHITECTURE.md` — 增加 Engine 并发安全设计（sync.RWMutex）、SyncDictsToMatcher 桥接、Graceful Shutdown 架构
- **修改** `docs/v2-tech-design/03-CORE-ALGORITHMS.md` — 更新 Base 算法 totalKey TTL（clampTTL）、Leak 算法 per-key 并发锁
- **修改** `docs/v2-tech-design/04-API-REFERENCE.md` — 更新 browse 参数、错误响应安全消息、update 响应、health/ready 端点
- **修改** `docs/v2-tech-design/05-CONFIG-REFERENCE.md` — 增加 limit.base 配置说明和 TTL 公式

## Phase 3: P2 中等问题修复

### Task 3-1: RecoveryMiddleware + requestIDCounter atomic

- **修改** `internal/api/middleware.go` — `requestIDCounter` 从 `sync.Mutex + uint64` 改为 `atomic.Uint64`，移除 `requestIDMutex`
- **修改** `internal/api/api_test.go` — 新增 TestGenerateRequestID_Unique（1000并发唯一性）、TestRecoveryMiddleware_PanicNoLeakage

### Task 3-2: ParseSize 正则缓存 + validateRules append 安全

- **修改** `internal/config/config.go` — `sizeRegexp` 提取为包级预编译变量
- **修改** `internal/config/rules.go` — `dayRegexp` 提取为包级预编译变量；`validateRules` append 链改为独立切片累加
- **修改** `internal/config/config_test.go` — 新增 TestParseSize_Concurrent
- **修改** `internal/config/rules_test.go` — 新增 TestValidateRules_DoesNotMutateOriginal

### Task 3-3: Check/Browse 重复代码提取 + MetricsMiddleware 清理

- **修改** `internal/engine/engine.go` — 提取 `matchRules(ctx, ruleSet, req, updateOnMatch)` 公共方法，简化 Check/Browse
- **修改** `internal/api/middleware.go` — MetricsMiddleware 使用 `c.FullPath()` 路由模式替代原始路径，防止 map 无限增长
- **修改** `internal/engine/engine_test.go` — 新增 TestEngine_MatchRules_Consistency（6 个子测试）
- **修改** `internal/api/api_test.go` — 新增 TestMetricsMiddleware_PathNormalization

### Task 3-4: CORS 可配置 + Batch 并行化

- **修改** `internal/api/middleware.go` — 新增 CORSConfig 结构体和 CORSMiddlewareWithConfig，支持可配置允许来源列表
- **修改** `internal/api/router.go` — RouterConfig 增加 CORSAllowOrigins 字段
- **修改** `internal/api/handler.go` — Batch 处理改为 goroutine + sync.WaitGroup 并行化
- **修改** `internal/api/api_test.go` — 新增 TestCORSMiddleware_ConfiguredOrigins（9 个子测试）、TestBatchHandler_Parallel

## Phase 4: 集成验证

- `go build -o /dev/null ./cmd/koala` — 编译成功
- `go test -race -count=1 ./internal/... ./pkg/...` — 9/10 包通过（redis Expire 为预存问题）
- `go vet ./...` — 无问题
- 总覆盖率: **75.4%**
- pkg/errors: 100.0%, pkg/logger: 100.0%, engine/matcher: 91.3%, storage/manager: 91.9%
