# Koala 快速参考卡

## 启动/停止

```bash
# 启动
./koala -config conf/koala.toml

# 后台启动
nohup ./koala -config conf/koala.toml > logs/koala.log 2>&1 &

# 停止
pkill -f koala
```

## 健康检查

```bash
curl http://localhost:18080/health
curl http://localhost:18080/ready
```

## Browse - 检查是否允许

```bash
# 基本请求
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123"}'

# 带IP和设备ID
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123","ip":"192.168.1.1","did":"device001"}'

# 检查并更新计数器
curl -X POST http://localhost:18080/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123","update":true}'
```

## Update - 更新计数器

```bash
curl -X POST http://localhost:18080/api/v1/update \
  -H "Content-Type: application/json" \
  -d '{"act":"login","uid":"user123"}'
```

## Batch - 批量检查

```bash
curl -X POST http://localhost:18080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "requests": [
      {"id":"1","act":"login","uid":"user001"},
      {"id":"2","act":"login","uid":"user002"}
    ]
  }'
```

## 响应示例

```json
// 允许
{"allowed":true,"code":0,"message":"ok"}

// 白名单放行
{"allowed":true,"code":0,"message":"VIP用户放行","rule_name":"vip_user_whitelist"}

// 黑名单拦截
{"allowed":false,"code":4003,"message":"IP已被封禁","rule_name":"blocked_ip_blacklist"}

// 限流
{"allowed":false,"code":4001,"message":"登录过于频繁，请稍后重试","auth_type":1}
```

## 常用错误码

| 码 | 含义 |
|----|------|
| 0 | 正常 |
| 4001 | 登录限流 |
| 4003 | IP封禁 |
| 4004 | 设备封禁 |
| 4999 | 全局限流 |

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
