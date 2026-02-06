# 04 - API 参考

## 4.1 接口总览

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 检查 | POST | /api/v1/browse | 检查请求是否允许（可选自动更新计数器） |
| 记录 | POST | /api/v1/update | 记录请求，增加计数 |
| 批量检查 | POST | /api/v1/batch | 批量检查多个请求（只读，不更新计数器） |
| 健康检查 | GET | /health | 服务存活探测（Liveness） |
| 就绪检查 | GET | /ready | 服务就绪探测（Readiness） |
| 指标 | GET | /metrics | Prometheus 指标 |

### 通用说明

**Content-Type:**

| 场景 | Content-Type |
|------|------|
| API 请求（POST） | `application/json` |
| API 响应（browse/update/batch/health/ready） | `application/json` |
| 指标响应（metrics） | `text/plain; charset=utf-8` |

**X-Request-ID 响应头:**

`LoggingMiddleware` 为每个请求处理 `X-Request-ID` 头：
- 如果请求中包含 `X-Request-ID` 头，服务器原样回传
- 如果请求中未包含，服务器自动生成唯一 ID 并通过 `X-Request-ID` 头返回
- 格式为 `{YYYYMMDDHHmmss}-{递增序号}`，例如 `20260201120000-1`

**超时行为:**

`TimeoutMiddleware` 为所有请求设置超时截止时间（通过 `context.WithTimeout` 注入到请求上下文中）。默认超时时间为 **30 秒**（由 `DefaultRouterConfig()` 指定）。超时后下游操作可通过 `ctx.Done()` 感知。

> **注意**: `TimeoutMiddleware` 只向请求上下文注入截止时间，不会主动中断 HTTP 响应写入。如果下游 Handler 在 30 秒内未完成且未检查 `ctx.Done()`，客户端可能需要等待更久。下游引擎和存储层通过 `ctx.Done()` 感知超时后会主动放弃操作并返回错误（`code: -2`，HTTP 500）。建议客户端自行设置合理的请求超时（如 35 秒），避免长时间挂起。

**API 级限流中间件:**

> **注意**: `RateLimitMiddleware` 已定义（基于客户端 IP 的滑动窗口限流），`RouterConfig` 中也包含 `RateLimitPerSecond` 配置项（默认 10000），但当前 `NewRouter()` **未启用**该中间件。如需启用 API 级限流，需要在路由配置中手动接入 `RateLimitMiddleware`。启用后，超出限制的请求将收到 HTTP 429 响应。

## 快速开始

### 典型集成流程

**基本集成流程：**

1. 调用 `POST /api/v1/browse` 检查用户操作是否被允许
2. 根据响应中的 `allowed`、`code`、`auth_type` 决定处理方式
3. 如果允许且需要记录（如登录成功后才计数），调用 `POST /api/v1/update` 更新计数器
4. 或者在 browse 请求中设置 `"update": true`，合并检查和更新为一步

**结果处理建议：**

| 条件 | 处理方式 |
|------|---------|
| `code = 0` | 允许，正常放行 |
| `code > 0`, `auth_type = 0` | 拒绝，直接提示用户（如 "评论过于频繁"） |
| `code > 0`, `auth_type = 1` | 需要滑块验证，引导用户完成后重试 |
| `code > 0`, `auth_type = 2` | 需要短信验证，引导用户完成后重试 |
| `code < 0` | 系统错误（`-1` 参数错误，`-2` 内部错误，`-500` panic 恢复），建议放行并上报告警 |

### auth_type 值域表

| auth_type | 含义 | 说明 |
|-----------|------|------|
| 0 | 无验证 | 直接拒绝或允许，无需额外验证 |
| 1 | 滑块验证 | 客户端需展示滑块验证组件 |
| 2 | 短信验证 | 客户端需引导用户完成短信验证 |
| 其他正数 | 自定义验证类型 | 由规则配置文件中的 `result.auth_type` 自定义 |

### 规则匹配优先级

引擎按以下固定顺序执行规则匹配（首次匹配即返回，非贪婪）：

```
白名单 → 黑名单 → 业务规则(business) → 发帖规则(post) → 高级规则(advanced) → 默认规则(default)
```

匹配行为说明：

| 阶段 | 匹配时的行为 |
|------|-------------|
| 白名单匹配 | 直接返回对应规则的 result（通常 `code=0`，允许通过） |
| 黑名单匹配 | 直接返回对应规则的 result（通常为拒绝，如 `code=4003`） |
| 限流规则匹配，计数**未超限** | 返回 `code=0, message="ok"`（允许）；若 `update=true` 或通过 Update 接口调用则递增计数器 |
| 限流规则匹配，计数**已超限** | 返回规则配置的 result（如 `code=4102, message="评论过于频繁"`） |
| 无任何规则匹配 | 返回默认允许（`code=0, message="ok"`） |

> **注意**: 每个阶段内的规则按配置文件中的声明顺序逐条检查，首次匹配后立即返回，不再检查后续规则。

## 4.2 检查接口

### POST /api/v1/browse

检查请求是否被规则允许。当设置 `update=true` 时，如果检查结果为允许，还会自动调用引擎更新计数器（等同于先 Browse 再 Update）。

#### 请求

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100",
    "did": "device_abc",
    "ext": {
        "vip": "false",
        "device": "ios"
    },
    "update": true
}
```

**参数说明:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| act | string | 是 | 动作类型 |
| uid | string | 否 | 用户 ID |
| ip | string | 否 | 客户端 IP |
| did | string | 否 | 设备 ID |
| ext | object | 否 | 扩展参数（键值对，值为字符串） |
| update | bool | 否 | 是否在检查通过后自动更新计数器（默认 false）。设置为 true 时，如果 browse 结果为允许（`allowed=true`），引擎会自动调用 Check（含计数器递增）完成更新 |

#### 响应

**成功（允许）:**

当请求未命中任何拒绝规则，或未匹配到任何规则时，返回允许响应。

> **注意**: `rule_name` 和 `auth_type` 字段带有 `omitempty` 标签——当值为零值（空字符串 / 0）时，不会出现在 JSON 响应中。

```json
{
    "allowed": true,
    "code": 0,
    "message": "ok"
}
```

**成功（拒绝）:**

当请求命中限流规则或黑名单规则时，返回拒绝响应。`code`、`message` 和 `auth_type` 的值均来自规则配置中的 `result` 定义，而非硬编码。

```json
{
    "allowed": false,
    "code": 4102,
    "message": "评论过于频繁",
    "rule_name": "comment_limit"
}
```

> 上述示例基于 `rules.toml` 中 `comment_limit` 规则的配置（`act="comment"`, `uid="+"`，60 秒内超过 10 次触发）。实际返回值取决于部署环境中的规则配置文件。

**成功（需要验证）:**

当命中的规则 `result` 配置了非零的 `auth_type` 时，表示需要客户端进行二次验证。

```json
{
    "allowed": false,
    "code": 4001,
    "message": "登录过于频繁，请稍后重试",
    "rule_name": "login_rate_limit",
    "auth_type": 1
}
```

> 上述示例基于 `rules.toml` 中 `login_limit` 结果模板的配置（`code=4001`, `auth_type=1` 表示需要滑块验证）。实际返回值取决于部署环境中的规则配置文件。

**响应字段说明:**

| 字段 | 类型 | omitempty | 说明 |
|------|------|-----------|------|
| allowed | bool | 否 | 是否允许 |
| code | int | 否 | 结果码（0=允许，正数=规则配置的拒绝码） |
| message | string | 否 | 结果消息（允许时为 `"ok"`，拒绝时来自规则配置） |
| rule_name | string | **是** | 命中的规则名称（允许时为零值，不出现在 JSON 中） |
| auth_type | int | **是** | 验证类型（0 时不出现在 JSON 中；具体值由规则配置决定，例如 1=滑块验证） |

**错误响应:**

错误响应使用与正常响应相同的 `APIResponse` 结构，不暴露内部错误详情：

```json
{
    "allowed": false,
    "code": -1,
    "message": "请求参数错误"
}
```

服务器内部错误：
```json
{
    "allowed": false,
    "code": -2,
    "message": "服务器内部错误"
}
```

> **安全说明**: 错误响应中不包含内部错误详情（如堆栈、存储错误等），
> 仅返回安全的通用错误消息，详细错误信息记录在服务端日志中。

#### 示例

**cURL — 基本检查:**
```bash
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100"
  }'
```

**预期响应（允许）:**
```json
{
    "allowed": true,
    "code": 0,
    "message": "ok"
}
```

**cURL — 参数错误（缺少必填字段 act）:**
```bash
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"uid": "user123"}'
```

**预期响应（HTTP 400）:**
```json
{
    "allowed": false,
    "code": -1,
    "message": "请求参数错误"
}
```

**cURL — 触发限流被拒绝:**

以 `comment` 动作为例，`comment_limit` 规则配置为 60 秒内最多 10 次。连续调用超过 10 次后将被拒绝：

```bash
# 连续调用 11 次以触发限流（前 10 次需带 update=true 以递增计数器）
for i in $(seq 1 11); do
  curl -s -X POST http://localhost:9981/api/v1/browse \
    -H "Content-Type: application/json" \
    -d '{
      "act": "comment",
      "uid": "12345",
      "update": true
    }'
  echo ""
done
```

**第 11 次的预期响应（被拒绝）:**
```json
{
    "allowed": false,
    "code": 4102,
    "message": "评论过于频繁",
    "rule_name": "comment_limit"
}
```

> 实际触发所需的调用次数和返回值取决于部署环境中的规则配置文件。

**cURL — 带 update=true 的检查（合并检查与更新）:**

当 `update=true` 时，如果检查结果为允许，引擎会自动递增对应规则的计数器，无需单独调用 `/api/v1/update`。

```bash
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100",
    "update": true
  }'
```

**预期响应（允许并已自动更新计数器）:**
```json
{
    "allowed": true,
    "code": 0,
    "message": "ok"
}
```

> 响应与普通 browse 相同。区别在于服务端已自动递增计数器，后续无需再调用 update 接口。

## 4.3 记录接口

### POST /api/v1/update

对匹配到的限流规则递增计数器（标记一次操作发生）。通常在 browse 返回允许后调用。

**语义说明：**

- Update 的匹配逻辑与 Browse 相同（白名单 → 黑名单 → 业务 → 发帖 → 高级 → 默认）
- 底层实现调用引擎的 `Check` 方法（`matchRules` 的 `updateOnMatch=true`），匹配到限流规则且未触发限制时，递增对应计数器
- 如果请求匹配到白名单/黑名单，直接返回对应 result，不涉及计数器操作
- 如果没有匹配到任何规则，返回默认允许（`code=0, message="ok"`）
- Update 接口始终返回 `allowed=true, code=0`（成功更新），不返回引擎匹配结果的详情
- `update` 字段在此接口中无效（Update 本身就是更新操作）

#### 请求

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100"
}
```

参数与 browse 接口相同（`act` 为必填，其余可选）。`update` 字段在此接口中无效。

#### 响应

**成功:**
```json
{
    "allowed": true,
    "code": 0,
    "message": "ok"
}
```

**错误（参数错误）:**
```json
{
    "allowed": false,
    "code": -1,
    "message": "请求参数错误"
}
```

**错误（服务器错误）:**
```json
{
    "allowed": false,
    "code": -2,
    "message": "服务器内部错误"
}
```

#### 示例

**cURL:**
```bash
curl -X POST http://localhost:9981/api/v1/update \
  -H "Content-Type: application/json" \
  -d '{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100"
  }'
```

#### 使用模式

**模式 1: 分离调用**
```
1. 调用 browse → 返回 allowed=true
2. 执行业务操作
3. 调用 update → 记录请求
```

**模式 2: 合并调用（推荐）**

在 browse 请求中添加 `update: true` 参数：
```json
{
    "act": "comment",
    "uid": "12345",
    "update": true
}
```

如果 browse 返回允许，自动执行 update。

## 4.4 批量检查接口

### POST /api/v1/batch

批量检查多个请求。

> **注意**: 批量接口仅执行只读检查（等同于 Browse，`updateOnMatch=false`），**不支持 `update` 参数，不会更新计数器**。如需在批量检查通过后更新计数器，请对各通过项分别调用 `POST /api/v1/update`。

#### 请求

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
    "requests": [
        {
            "id": "req1",
            "act": "comment",
            "uid": "12345"
        },
        {
            "id": "req2",
            "act": "post",
            "uid": "12345"
        }
    ]
}
```

**参数说明:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| requests | array | 是 | 请求列表（最少 1 项） |
| requests[].id | string | 是 | 请求标识（用于关联响应） |
| requests[].act | string | 是 | 动作类型 |
| requests[].uid | string | 否 | 用户 ID |
| requests[].ip | string | 否 | 客户端 IP |
| requests[].did | string | 否 | 设备 ID |
| requests[].ext | object | 否 | 扩展参数 |

#### 响应

> **注意**: 批量接口整体始终返回 HTTP 200，但个别请求项可能因引擎处理失败而包含 `code: -2`（服务器内部错误）。调用方应逐项检查每个结果的 `allowed` 和 `code` 字段。

> `rule_name` 和 `auth_type` 同样带有 `omitempty`，零值时不出现。

```json
{
    "results": [
        {
            "id": "req1",
            "allowed": true,
            "code": 0,
            "message": "ok"
        },
        {
            "id": "req2",
            "allowed": false,
            "code": 4101,
            "message": "发帖数量已达上限",
            "rule_name": "post_user_limit",
            "auth_type": 1
        }
    ]
}
```

> 上述示例中 `req2` 的拒绝结果基于 `rules.toml` 中 `post_user_limit` 规则（`act="post"`, 3600 秒内超过 20 次触发，`result=post_limit`: `code=4101, auth_type=1`）。实际返回值取决于部署环境中的规则配置文件。

**验证错误消息:**

当批量请求项的 `id` 或 `act` 为空时，该项返回如下错误（不进入引擎处理）：

```json
{
    "id": "",
    "allowed": false,
    "code": -1,
    "message": "请求ID和Act不能为空"
}
```

其他批量级别的验证错误（整体请求格式错误、列表为空、超过 100 项）返回标准错误响应：

| 场景 | code | message |
|------|------|---------|
| JSON 格式错误或缺少 requests 字段 | -1 | 请求参数错误 |
| 请求列表为空 | -1 | 请求列表不能为空 |
| 请求数量超过 100 | -1 | 请求数量超过限制（最大100） |

#### 限制

- 单次最多 100 个请求
- 超时时间与单个请求相同

#### 并行执行

批量请求中的各项使用 goroutine 并行处理（通过 `sync.WaitGroup` 协调），以降低整体延迟。
参数校验（`id` 和 `act` 非空检查）在主协程中同步完成以避免竞态，校验通过的请求项才会并行调用引擎。

#### 示例

**cURL — 基本批量检查:**
```bash
curl -X POST http://localhost:9981/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"id": "req1", "act": "comment", "uid": "12345"},
      {"id": "req2", "act": "post", "uid": "12345"}
    ]
  }'
```

**cURL — 部分失败示例（某项 id 为空）:**

当某项 `id` 或 `act` 为空时，该项返回参数错误，其余项正常处理：

```bash
curl -X POST http://localhost:9981/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"id": "req1", "act": "comment", "uid": "12345"},
      {"id": "", "act": "post", "uid": "12345"},
      {"id": "req3", "act": "login", "uid": "67890"}
    ]
  }'
```

**预期响应（混合结果，HTTP 200）:**
```json
{
    "results": [
        {
            "id": "req1",
            "allowed": true,
            "code": 0,
            "message": "ok"
        },
        {
            "id": "",
            "allowed": false,
            "code": -1,
            "message": "请求ID和Act不能为空"
        },
        {
            "id": "req3",
            "allowed": true,
            "code": 0,
            "message": "ok"
        }
    ]
}
```

> 注意：整体 HTTP 状态码仍为 200，调用方需逐项检查每个 result 的 `code` 字段。

## 4.5 健康检查接口

### GET /health

返回服务存活状态（Liveness Probe）。用于 K8s liveness probe 或负载均衡器的存活检测，表示进程正在运行且未死锁。

#### 响应

**健康:**
```json
{
    "status": "ok",
    "timestamp": "2026-02-01T12:00:00Z"
}
```

#### 示例

**cURL:**
```bash
curl http://localhost:9981/health
```

### GET /ready

返回服务就绪状态（Readiness Probe）。用于 K8s readiness probe，表示服务可以接受流量。与 Health 的区别在于：Health 只检查进程存活，Ready 检查服务是否准备好处理请求（例如依赖组件是否可用）。

#### 响应

**就绪:**

> **注意**: `ReadyResponse` 的 `message` 和 `timestamp` 字段均带有 `omitempty` 标签。
> 当前实现中 `message` 始终为空字符串，因此不会出现在 JSON 输出中；`timestamp` 有值，会正常输出。

```
HTTP 200 OK
```
```json
{
    "ready": true,
    "timestamp": "2026-02-01T12:00:00Z"
}
```

> **注意**: 当前实现中 Ready 端点始终返回 HTTP 200。
> 未来可根据依赖组件检查结果返回非 200 状态码。

| 探针 | 用途 | 语义 | 失败含义 |
|------|------|------|---------|
| Health (`/health`) | Liveness | 进程存活 | 需要重启 Pod |
| Ready (`/ready`) | Readiness | 可接受流量 | 从负载均衡中摘除，不重启 |

#### 示例

**cURL:**
```bash
curl http://localhost:9981/ready
```

## 4.6 指标接口

### GET /metrics

返回简化的 Prometheus 格式指标。指标数据来自 `MetricsMiddleware`（需在 `RouterConfig` 中设置 `EnableMetrics: true`）。

#### 响应说明

**Content-Type:** `text/plain; charset=utf-8`

指标由 `buildPrometheusMetrics()` 函数构建，包含以下指标：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `koala_http_requests_total` | counter | `path` | 按路径统计的 HTTP 请求总数 |
| `koala_http_requests_failed` | counter | `path` | 按路径统计的失败请求数（HTTP 状态码 >= 400） |
| `koala_http_latency_ms` | gauge | `path` | 按路径统计的平均延迟（毫秒） |
| `koala_up` | gauge | 无 | 服务运行状态（始终为 1） |

`path` 标签的值为 `{HTTP方法}_{路由路径}`，例如 `POST_/api/v1/browse`。

#### 响应示例

> **注意**: 实际输出中，只有 `koala_http_requests_total` 有 `# HELP` 和 `# TYPE` 注释行。`koala_http_requests_failed` 和 `koala_http_latency_ms` 没有注释行。输出按路径交错排列——每个路径连续输出 total、failed、latency 三行，而非按指标名分组。

```prometheus
# HELP koala_http_requests_total Koala HTTP请求总数
# TYPE koala_http_requests_total counter
koala_http_requests_total{path="POST_/api/v1/browse"} 12345
koala_http_requests_failed{path="POST_/api/v1/browse"} 23
koala_http_latency_ms{path="POST_/api/v1/browse"} 2
koala_http_requests_total{path="POST_/api/v1/update"} 6789
koala_http_requests_failed{path="POST_/api/v1/update"} 5
koala_http_latency_ms{path="POST_/api/v1/update"} 1
koala_http_requests_total{path="POST_/api/v1/batch"} 100
koala_http_requests_failed{path="POST_/api/v1/batch"} 1
koala_http_latency_ms{path="POST_/api/v1/batch"} 15
koala_http_requests_total{path="GET_/health"} 500
koala_http_requests_failed{path="GET_/health"} 0
koala_http_latency_ms{path="GET_/health"} 0

# HELP koala_up Koala服务是否运行
# TYPE koala_up gauge
koala_up 1
```

> **注意**: 由于 `buildPrometheusMetrics()` 遍历 Go map，路径的输出顺序是**不确定的**（每次请求可能不同）。上述示例仅展示格式参考。

> **注意**: 当 `EnableMetrics` 为 false 或 `MetricsMiddleware` 未初始化时，`/metrics` 端点仍可访问，但仅返回 `koala_http_requests_total 0`。

#### 示例

**cURL:**
```bash
curl http://localhost:9981/metrics
```

## 4.7 错误码

### HTTP 状态码

| HTTP 状态码 | 说明 | 触发场景 |
|------------|------|----------|
| 200 | 请求成功 | 正常请求（包括 browse 返回拒绝结果） |
| 400 | 请求参数错误 | JSON 格式错误、`act` 字段缺失等 |
| 429 | 请求过于频繁 | `RateLimitMiddleware` 触发（当前未默认启用，见 4.1 说明） |
| 500 | 服务内部错误 | 引擎处理异常、panic 恢复 |

### 响应体 code 字段

响应体中的 `code` 字段是业务层面的结果码，与 HTTP 状态码是**独立的两套编码体系**。

| code 值 | 含义 | 说明 |
|---------|------|------|
| 0 | 允许 / 成功 | browse 允许通过，或 update 成功 |
| -1 | 参数错误 | 请求参数校验失败（HTTP 400） |
| -2 | 内部错误 | 引擎处理异常（HTTP 500） |
| -500 | panic 恢复 | `RecoveryMiddleware` 捕获 panic 后返回（HTTP 500） |
| 429 | API 限流 | `RateLimitMiddleware` 触发（HTTP 429，当前未默认启用） |
| 正整数 | 规则拒绝码 | 来自规则配置的 `result.code`，具体值由运维自定义 |

> **重要**: 当 `code >= 0` 时，HTTP 状态码为 200。即使 `allowed=false`（被规则拒绝），HTTP 层面仍为成功。
> 只有框架层面的错误（参数校验、内部异常、panic、限流）才会返回非 200 的 HTTP 状态码。

## 4.8 Handler 实现参考

> 以下代码片段摘自实际实现，供参考理解 API 行为。

### 4.8.1 类型定义

```go
// internal/api/types.go

// APIRequest 表示Browse/Update API请求体。
type APIRequest struct {
    Act    string            `json:"act" binding:"required"`
    UID    string            `json:"uid"`
    IP     string            `json:"ip"`
    DID    string            `json:"did"`
    Ext    map[string]string `json:"ext"`
    Update bool              `json:"update"`
}

// APIResponse 表示API响应体。
type APIResponse struct {
    Allowed  bool   `json:"allowed"`
    Code     int    `json:"code"`
    Message  string `json:"message"`
    RuleName string `json:"rule_name,omitempty"`
    AuthType int    `json:"auth_type,omitempty"`
}
```

### 4.8.2 Browse Handler

```go
// internal/api/handler.go

// Browse 处理频率检查请求。
// POST /api/v1/browse
func (h *Handler) Browse(c *gin.Context) {
    var req APIRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, APIResponse{   // HTTP 400, code=-1
            Allowed: false,
            Code:    -1,
            Message: "请求参数错误",
        })
        return
    }

    engineReq := &EngineRequest{ /* ... */ }

    resp, err := h.engine.Browse(c.Request.Context(), engineReq)
    if err != nil {
        c.JSON(http.StatusInternalServerError, APIResponse{  // HTTP 500, code=-2
            Allowed: false,
            Code:    -2,
            Message: "服务器内部错误",
        })
        return
    }

    c.JSON(http.StatusOK, APIResponse{               // HTTP 200, code 来自引擎/规则
        Allowed:  resp.Allowed,
        Code:     resp.Code,
        Message:  resp.Message,
        RuleName: resp.RuleName,
        AuthType: resp.AuthType,
    })
}
```

### 4.8.3 Router 配置

```go
// internal/api/router.go

func NewRouter(handler *Handler, config *RouterConfig) *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    router := gin.New()

    if config == nil {
        config = DefaultRouterConfig()
    }

    // 全局中间件
    router.Use(RecoveryMiddleware())   // panic 恢复 → code=-500
    router.Use(LoggingMiddleware())    // 请求日志 + X-Request-ID

    if config.EnableCORS {
        router.Use(CORSMiddlewareWithConfig(CORSConfig{
            AllowOrigins: config.CORSAllowOrigins,
        }))
    }

    if config.RequestTimeout > 0 {
        router.Use(TimeoutMiddleware(config.RequestTimeout)) // 默认 30s
    }

    // 指标中间件
    var metricsMiddleware *MetricsMiddleware
    if config.EnableMetrics {
        metricsMiddleware = NewMetricsMiddleware()
        router.Use(metricsMiddleware.Handler())
    }

    // 注意：RateLimitMiddleware 已定义但未在此接入

    // 健康检查
    router.GET("/health", handler.Health)
    router.GET("/ready", handler.Ready)

    // Prometheus 格式指标
    router.GET("/metrics", func(c *gin.Context) {
        c.Header("Content-Type", "text/plain; charset=utf-8")
        c.String(http.StatusOK, buildPrometheusMetrics(metricsMiddleware))
    })

    // API v1
    v1 := router.Group("/api/v1")
    {
        v1.POST("/browse", handler.Browse)
        v1.POST("/update", handler.Update)
        v1.POST("/batch", handler.Batch)
    }

    return router
}
```
