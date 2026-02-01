# 04 - API 参考

## 4.1 接口总览

| 接口 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 检查 | POST | /api/v1/browse | 检查请求是否允许 |
| 记录 | POST | /api/v1/update | 记录请求，增加计数 |
| 批量检查 | POST | /api/v1/batch | 批量检查多个请求 |
| 健康检查 | GET | /health | 服务健康状态 |
| 就绪检查 | GET | /ready | 服务就绪状态 |
| 指标 | GET | /metrics | Prometheus 指标 |

## 4.2 检查接口

### POST /api/v1/browse

检查请求是否被规则允许。

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
    "ext": {
        "vip": "false",
        "device": "ios"
    }
}
```

**参数说明:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| act | string | 是 | 动作类型 |
| uid | string | 否 | 用户 ID |
| ip | string | 否 | 客户端 IP |
| ext | object | 否 | 扩展参数 |

#### 响应

**成功（允许）:**
```json
{
    "allowed": true,
    "code": 0,
    "message": "Allow",
    "rule_name": "",
    "auth_type": 0
}
```

**成功（拒绝）:**
```json
{
    "allowed": false,
    "code": 10,
    "message": "Deny",
    "rule_name": "daily_comment_limit",
    "auth_type": 0
}
```

**成功（需要验证）:**
```json
{
    "allowed": false,
    "code": 20,
    "message": "Auth",
    "rule_name": "suspicious_activity",
    "auth_type": 1
}
```

**响应字段说明:**

| 字段 | 类型 | 说明 |
|------|------|------|
| allowed | bool | 是否允许 |
| code | int | 结果码（0=允许，10=拒绝，20+=需要验证） |
| message | string | 结果消息 |
| rule_name | string | 命中的规则名称（空表示无命中） |
| auth_type | int | 验证类型（0=无，1=滑块，2=密码，3=短信，4=邮箱，5=图形验证码） |

**错误响应:**
```json
{
    "error": "invalid request",
    "code": 400
}
```

#### 示例

**cURL:**
```bash
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100"
  }'
```

**Go:**
```go
type BrowseRequest struct {
    Act string            `json:"act"`
    UID string            `json:"uid"`
    IP  string            `json:"ip"`
    Ext map[string]string `json:"ext,omitempty"`
}

type BrowseResponse struct {
    Allowed  bool   `json:"allowed"`
    Code     int    `json:"code"`
    Message  string `json:"message"`
    RuleName string `json:"rule_name"`
    AuthType int    `json:"auth_type"`
}

func Browse(ctx context.Context, req *BrowseRequest) (*BrowseResponse, error) {
    // ...
}
```

## 4.3 记录接口

### POST /api/v1/update

记录请求，增加计数器。通常在 browse 返回允许后调用。

#### 请求

**Body:**
```json
{
    "act": "comment",
    "uid": "12345",
    "ip": "192.168.1.100"
}
```

参数与 browse 接口相同。

#### 响应

**成功:**
```json
{
    "success": true
}
```

**错误:**
```json
{
    "success": false,
    "error": "storage error"
}
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

#### 请求

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
| requests | array | 是 | 请求列表 |
| requests[].id | string | 是 | 请求标识（用于关联响应） |
| requests[].* | - | - | 其他参数同 browse |

#### 响应

```json
{
    "results": [
        {
            "id": "req1",
            "allowed": true,
            "code": 0,
            "message": "Allow",
            "rule_name": "",
            "auth_type": 0
        },
        {
            "id": "req2",
            "allowed": false,
            "code": 10,
            "message": "Deny",
            "rule_name": "daily_post_limit",
            "auth_type": 0
        }
    ]
}
```

#### 限制

- 单次最多 100 个请求
- 超时时间与单个请求相同

## 4.5 健康检查接口

### GET /health

返回服务健康状态。

#### 响应

**健康:**
```json
{
    "status": "healthy",
    "storage": "ok",
    "uptime": 3600
}
```

**不健康:**
```json
{
    "status": "unhealthy",
    "storage": "degraded",
    "uptime": 3600,
    "error": "redis connection failed"
}
```

### GET /ready

返回服务就绪状态（用于 K8s readiness probe）。

#### 响应

**就绪:**
```
HTTP 200 OK
```

**未就绪:**
```
HTTP 503 Service Unavailable
```

## 4.6 指标接口

### GET /metrics

返回 Prometheus 格式的指标。

#### 响应示例

```prometheus
# HELP koala_requests_total Total number of requests
# TYPE koala_requests_total counter
koala_requests_total{action="browse",result="allow"} 12345
koala_requests_total{action="browse",result="deny"} 678
koala_requests_total{action="update"} 12345

# HELP koala_request_duration_seconds Request duration in seconds
# TYPE koala_request_duration_seconds histogram
koala_request_duration_seconds_bucket{action="browse",le="0.001"} 10000
koala_request_duration_seconds_bucket{action="browse",le="0.005"} 11500
koala_request_duration_seconds_bucket{action="browse",le="0.01"} 12000
koala_request_duration_seconds_bucket{action="browse",le="+Inf"} 12345
koala_request_duration_seconds_sum{action="browse"} 25.5
koala_request_duration_seconds_count{action="browse"} 12345

# HELP koala_rules_matched_total Total number of rules matched
# TYPE koala_rules_matched_total counter
koala_rules_matched_total{rule="daily_comment_limit"} 678
koala_rules_matched_total{rule="api_rate_limit"} 123

# HELP koala_storage_status Storage status (1=primary, 0=fallback)
# TYPE koala_storage_status gauge
koala_storage_status 1

# HELP koala_policy_reload_total Total number of policy reloads
# TYPE koala_policy_reload_total counter
koala_policy_reload_total{status="success"} 5
koala_policy_reload_total{status="failure"} 0
```

## 4.7 错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|------------|------|
| 400 | 400 | 请求参数错误 |
| 500 | 500 | 服务内部错误 |
| 503 | 503 | 服务不可用 |

## 4.8 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 500 | 服务内部错误 |
| 503 | 服务不可用 |

## 4.9 Handler 实现

### 4.9.1 Browse Handler

```go
// internal/api/handler/browse.go

package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "koala/internal/engine"
)

type BrowseHandler struct {
    engine *engine.RuleEngine
}

func NewBrowseHandler(engine *engine.RuleEngine) *BrowseHandler {
    return &BrowseHandler{engine: engine}
}

type BrowseRequest struct {
    Act    string            `json:"act" binding:"required"`
    UID    string            `json:"uid"`
    IP     string            `json:"ip"`
    Ext    map[string]string `json:"ext"`
    Update bool              `json:"update"`
}

type BrowseResponse struct {
    Allowed  bool   `json:"allowed"`
    Code     int    `json:"code"`
    Message  string `json:"message"`
    RuleName string `json:"rule_name"`
    AuthType int    `json:"auth_type"`
}

func (h *BrowseHandler) Handle(c *gin.Context) {
    var req BrowseRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "invalid request",
            "code":  400,
        })
        return
    }

    // 构建引擎请求
    engineReq := &engine.Request{
        Act: req.Act,
        UID: req.UID,
        IP:  req.IP,
        Ext: req.Ext,
    }

    // 评估规则
    result := h.engine.Evaluate(c.Request.Context(), engineReq)

    // 如果允许且需要更新
    if result.Allowed && req.Update {
        go h.engine.Update(c.Request.Context(), engineReq)
    }

    c.JSON(http.StatusOK, BrowseResponse{
        Allowed:  result.Allowed,
        Code:     result.Code,
        Message:  result.Message,
        RuleName: result.RuleName,
        AuthType: result.AuthType,
    })
}
```

### 4.9.2 Router 定义

```go
// internal/api/router.go

package api

import (
    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "koala/internal/api/handler"
    "koala/internal/api/middleware"
    "koala/internal/engine"
)

func NewRouter(engine *engine.RuleEngine) *gin.Engine {
    r := gin.New()

    // 中间件
    r.Use(middleware.Logger())
    r.Use(middleware.Recovery())
    r.Use(middleware.Metrics())

    // 健康检查
    r.GET("/health", handler.NewHealthHandler(engine).Handle)
    r.GET("/ready", handler.NewReadyHandler(engine).Handle)

    // Prometheus 指标
    r.GET("/metrics", gin.WrapH(promhttp.Handler()))

    // API v1
    v1 := r.Group("/api/v1")
    {
        browseHandler := handler.NewBrowseHandler(engine)
        updateHandler := handler.NewUpdateHandler(engine)
        batchHandler := handler.NewBatchHandler(engine)

        v1.POST("/browse", browseHandler.Handle)
        v1.POST("/update", updateHandler.Handle)
        v1.POST("/batch", batchHandler.Handle)
    }

    return r
}
```
