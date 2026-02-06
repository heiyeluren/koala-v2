# Koala 快速参考卡

## 启动/停止

```bash
# 启动
./bin/koala -config conf/koala.toml

# 后台启动
nohup ./bin/koala -config conf/koala.toml > logs/koala.log 2>&1 &

# 停止
pkill -f koala
```

## 健康检查

```bash
curl http://localhost:9981/health
curl http://localhost:9981/ready
```

## Browse - 检查是否允许

```bash
# 基本请求
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123"}'

# 带IP和设备ID
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123","ip":"192.168.1.1","did":"device001"}'

# 检查并更新计数器
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123","update":true}'
```

## Update - 更新计数器

```bash
curl -X POST http://localhost:9981/api/v1/update \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123"}'
```

## Batch - 批量检查

```bash
curl -X POST http://localhost:9981/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"id":"1","act":"login","uid":"user001"},
      {"id":"2","act":"login","uid":"user002"}
    ]
  }'
```

Batch 响应示例：

```json
{
  "results": [
    {"id":"1","allowed":true,"code":0,"message":"ok"},
    {"id":"2","allowed":false,"code":4001,"message":"登录过于频繁，请稍后重试","rule_name":"login_rate_limit","auth_type":1}
  ]
}
```

## 响应示例

```json
// 允许
{"allowed":true,"code":0,"message":"ok"}

// 白名单放行
{"allowed":true,"code":0,"message":"VIP用户放行","rule_name":"vip_user_whitelist"}

// 黑名单拦截
{"allowed":false,"code":4003,"message":"IP已被封禁","rule_name":"blocked_ip_blacklist"}

// 限流拒绝
{"allowed":false,"code":4001,"message":"登录过于频繁，请稍后重试","rule_name":"login_rate_limit","auth_type":1}
```

> **注意**：`rule_name` 和 `auth_type` 字段带有 `omitempty`，当值为零值（空字符串或0）时不会出现在 JSON 响应中。

## 常用错误码

### 系统错误码

| 码 | 含义 | 触发场景 |
|----|------|----------|
| -1 | 参数错误 | 请求参数绑定失败或校验不通过 |
| -2 | 内部错误 | 引擎处理异常 |
| -500 | panic恢复 | 服务器发生未捕获的异常 |
| 429 | API限流 | API自身限流中间件拦截 |

### 业务错误码

| 码 | 含义 | 对应规则 |
|----|------|----------|
| 0 | 正常通过 | — |
| 4001 | 登录限流 | login_rate_limit |
| 4002 | IP登录限流 | login_ip_rate_limit |
| 4003 | IP封禁 | blocked_ip_blacklist |
| 4004 | 设备封禁 | blocked_device_blacklist |
| 4101 | 发帖限流 | post_user_limit |
| 4102 | 评论限流 | comment_limit |
| 4201 | API调用限流 | api_global_limit |
| 4999 | 全局默认限流 | global_default_limit |

## 日志查看

```bash
tail -f logs/koala.log              # 实时日志
grep "error" logs/koala.log         # 错误日志
grep "user123" logs/koala.log       # 搜索用户
```

## Redis 调试

```bash
redis-cli KEYS "koala:*"            # 查看所有key
redis-cli FLUSHALL                  # 清空计数器
```
