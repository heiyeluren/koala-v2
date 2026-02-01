# 05 - API 集成测试

## 1. 测试范围

测试 `internal/api/` 下的 HTTP 接口：
- POST /api/v1/browse
- POST /api/v1/update
- POST /api/v1/batch
- GET /health
- GET /ready
- GET /metrics

---

## 2. Browse 接口测试

### 2.1 基本功能测试

#### TC-AP-001-Browse_Allow_NoRule 🔴
```
描述: 无匹配规则时返回允许
前置条件: 规则引擎已初始化，无匹配规则
请求:
  POST /api/v1/browse
  Content-Type: application/json
  {"act": "unknown_action", "uid": "12345"}
预期响应:
  Status: 200
  {
    "allowed": true,
    "code": 0,
    "message": "Allow",
    "rule_name": "",
    "auth_type": 0
  }
```

#### TC-AP-002-Browse_Deny_RuleHit 🔴
```
描述: 命中规则时返回拒绝
前置条件: 存在规则 daily_limit，当前计数已达上限
请求:
  POST /api/v1/browse
  {"act": "post", "uid": "12345"}
预期响应:
  Status: 200
  {
    "allowed": false,
    "code": 10,
    "message": "Deny",
    "rule_name": "daily_limit",
    "auth_type": 0
  }
```

#### TC-AP-003-Browse_Allow_UnderLimit 🔴
```
描述: 未达限制时返回允许
前置条件: 存在规则 daily_limit(count=10)，当前计数=5
请求:
  POST /api/v1/browse
  {"act": "post", "uid": "12345"}
预期响应:
  Status: 200
  {
    "allowed": true,
    "code": 0,
    "message": "Allow",
    "rule_name": "",
    "auth_type": 0
  }
```

#### TC-AP-004-Browse_Auth_Required 🔴
```
描述: 需要验证时返回验证码
前置条件: 存在规则触发滑块验证
请求:
  POST /api/v1/browse
  {"act": "login_fail", "uid": "12345"}
预期响应:
  Status: 200
  {
    "allowed": false,
    "code": 20,
    "message": "Auth",
    "rule_name": "login_protection",
    "auth_type": 1
  }
```

### 2.2 白名单/黑名单测试

#### TC-AP-005-Browse_Whitelist_Allow 🔴
```
描述: 白名单用户直接放行
前置条件: uid "vip001" 在白名单中
请求:
  POST /api/v1/browse
  {"act": "any_action", "uid": "vip001"}
预期响应:
  {
    "allowed": true,
    "rule_name": "uid_whitelist"
  }
```

#### TC-AP-006-Browse_Blacklist_Deny 🔴
```
描述: 黑名单用户直接拒绝
前置条件: uid "banned001" 在黑名单中
请求:
  POST /api/v1/browse
  {"act": "any_action", "uid": "banned001"}
预期响应:
  {
    "allowed": false,
    "rule_name": "uid_blacklist"
  }
```

#### TC-AP-007-Browse_IP_Whitelist 🔴
```
描述: IP 白名单放行
前置条件: ip "127.0.0.1" 在 IP 白名单中
请求:
  POST /api/v1/browse
  {"act": "any_action", "ip": "127.0.0.1"}
预期响应:
  {
    "allowed": true
  }
```

#### TC-AP-008-Browse_IP_Blacklist 🔴
```
描述: IP 黑名单拒绝
前置条件: ip "1.2.3.4" 在 IP 黑名单中
请求:
  POST /api/v1/browse
  {"act": "any_action", "ip": "1.2.3.4"}
预期响应:
  {
    "allowed": false
  }
```

### 2.3 Update 联动测试

#### TC-AP-009-Browse_WithUpdate_True 🔴
```
描述: Browse 带 update=true 参数
前置条件: 规则存在，当前计数=0
请求:
  POST /api/v1/browse
  {"act": "post", "uid": "12345", "update": true}
预期结果:
  - 返回 allowed=true
  - 计数器增加到 1
```

#### TC-AP-010-Browse_WithUpdate_False 🔴
```
描述: Browse 带 update=false 参数
前置条件: 规则存在，当前计数=0
请求:
  POST /api/v1/browse
  {"act": "post", "uid": "12345", "update": false}
预期结果:
  - 返回 allowed=true
  - 计数器保持 0
```

### 2.4 参数验证测试

#### TC-AP-011-Browse_MissingAct 🔴
```
描述: 缺少必填参数 act
请求:
  POST /api/v1/browse
  {"uid": "12345"}
预期响应:
  Status: 400
  {"error": "act is required", "code": 400}
```

#### TC-AP-012-Browse_EmptyBody 🔴
```
描述: 请求体为空
请求:
  POST /api/v1/browse
  (empty body)
预期响应:
  Status: 400
  {"error": "invalid request", "code": 400}
```

#### TC-AP-013-Browse_InvalidJson 🔴
```
描述: 无效 JSON
请求:
  POST /api/v1/browse
  {invalid json}
预期响应:
  Status: 400
```

#### TC-AP-014-Browse_ExtParams 🔴
```
描述: 使用扩展参数
前置条件: 存在规则匹配 vip="true"
请求:
  POST /api/v1/browse
  {
    "act": "ai_question",
    "uid": "12345",
    "ext": {"vip": "true"}
  }
预期结果:
  - 扩展参数参与规则匹配
```

---

## 3. Update 接口测试

#### TC-AP-015-Update_Success 🔴
```
描述: 成功更新计数
前置条件: 规则存在
请求:
  POST /api/v1/update
  {"act": "post", "uid": "12345"}
预期响应:
  Status: 200
  {"success": true}
```

#### TC-AP-016-Update_Increment 🔴
```
描述: 多次更新递增计数
前置条件: 规则存在，当前计数=0
测试步骤:
  1. 调用 Update 3 次
  2. 调用 Browse 检查
预期结果:
  - 计数器为 3
```

#### TC-AP-017-Update_NoMatchingRule 🟡
```
描述: 无匹配规则时的更新
请求:
  POST /api/v1/update
  {"act": "unknown_action", "uid": "12345"}
预期响应:
  Status: 200
  {"success": true}
说明: 无匹配规则时不报错，只是没有计数器被更新
```

#### TC-AP-018-Update_MissingAct 🔴
```
描述: Update 缺少 act 参数
请求:
  POST /api/v1/update
  {"uid": "12345"}
预期响应:
  Status: 400
```

---

## 4. Batch 接口测试

#### TC-AP-019-Batch_SingleRequest 🔴
```
描述: 批量接口单个请求
请求:
  POST /api/v1/batch
  {
    "requests": [
      {"id": "req1", "act": "post", "uid": "12345"}
    ]
  }
预期响应:
  {
    "results": [
      {"id": "req1", "allowed": true, ...}
    ]
  }
```

#### TC-AP-020-Batch_MultipleRequests 🔴
```
描述: 批量接口多个请求
请求:
  POST /api/v1/batch
  {
    "requests": [
      {"id": "req1", "act": "post", "uid": "12345"},
      {"id": "req2", "act": "comment", "uid": "12345"},
      {"id": "req3", "act": "post", "uid": "67890"}
    ]
  }
预期响应:
  - results 包含 3 个结果
  - 每个结果 id 与请求对应
```

#### TC-AP-021-Batch_MixedResults 🔴
```
描述: 批量接口返回混合结果
前置条件: uid "12345" 的 post 已达上限
请求:
  POST /api/v1/batch
  {
    "requests": [
      {"id": "req1", "act": "post", "uid": "12345"},
      {"id": "req2", "act": "post", "uid": "67890"}
    ]
  }
预期响应:
  - req1: allowed=false
  - req2: allowed=true
```

#### TC-AP-022-Batch_EmptyRequests 🔴
```
描述: 批量接口空请求列表
请求:
  POST /api/v1/batch
  {"requests": []}
预期响应:
  Status: 400
  {"error": "requests cannot be empty"}
```

#### TC-AP-023-Batch_MissingId 🔴
```
描述: 批量请求缺少 id
请求:
  POST /api/v1/batch
  {
    "requests": [
      {"act": "post", "uid": "12345"}
    ]
  }
预期响应:
  Status: 400
  {"error": "request id is required"}
```

#### TC-AP-024-Batch_TooManyRequests 🟡
```
描述: 批量请求超过限制
请求:
  POST /api/v1/batch
  {"requests": [... 101 个请求 ...]}
预期响应:
  Status: 400
  {"error": "too many requests, max 100"}
```

---

## 5. Health 接口测试

#### TC-AP-025-Health_Healthy 🔴
```
描述: 服务健康
前置条件: 服务正常运行
请求:
  GET /health
预期响应:
  Status: 200
  {
    "status": "healthy",
    "storage": "ok",
    "uptime": <number>
  }
```

#### TC-AP-026-Health_StorageDegraded 🔴
```
描述: 存储降级
前置条件: Redis 不可用，已降级到本地存储
请求:
  GET /health
预期响应:
  Status: 200
  {
    "status": "healthy",
    "storage": "degraded"
  }
```

#### TC-AP-027-Ready_Ready 🔴
```
描述: 服务就绪
前置条件: 服务完全初始化
请求:
  GET /ready
预期响应:
  Status: 200
```

#### TC-AP-028-Ready_NotReady 🔴
```
描述: 服务未就绪
前置条件: 服务正在初始化
请求:
  GET /ready
预期响应:
  Status: 503
```

---

## 6. Metrics 接口测试

#### TC-AP-029-Metrics_Format 🔴
```
描述: Metrics 返回 Prometheus 格式
请求:
  GET /metrics
预期响应:
  Status: 200
  Content-Type: text/plain
  Body: 包含 Prometheus 格式的指标
```

#### TC-AP-030-Metrics_RequestCount 🔴
```
描述: Metrics 包含请求计数
前置条件: 已处理一些请求
请求:
  GET /metrics
预期结果:
  - 包含 koala_requests_total
  - 数值与实际请求数匹配
```

---

## 7. 错误处理测试

#### TC-AP-031-Error_InternalError 🔴
```
描述: 内部错误处理
前置条件: 模拟存储层错误
请求:
  POST /api/v1/browse
  {"act": "post", "uid": "12345"}
预期响应:
  Status: 500
  {"error": "internal error", "code": 500}
```

#### TC-AP-032-Error_Timeout 🟡
```
描述: 请求超时
前置条件: 模拟存储操作超时
请求:
  POST /api/v1/browse
预期响应:
  Status: 504 或 200（降级处理）
```

---

## 8. 并发测试

#### TC-AP-033-Concurrent_Browse 🔴
```
描述: 并发 Browse 请求
测试步骤:
  1. 启动 100 个并发请求
  2. 检查所有响应
预期结果:
  - 所有请求返回有效响应
  - 无竞态条件错误
```

#### TC-AP-034-Concurrent_Update 🔴
```
描述: 并发 Update 请求
测试步骤:
  1. 启动 100 个并发 Update 请求（同一 key）
  2. 检查最终计数
预期结果:
  - 计数器值为 100
```

#### TC-AP-035-Concurrent_Mixed 🔴
```
描述: 并发混合请求
测试步骤:
  1. 启动 50 个 Browse + 50 个 Update
  2. 检查结果
预期结果:
  - 所有请求正常处理
  - Browse 和 Update 数据一致
```
