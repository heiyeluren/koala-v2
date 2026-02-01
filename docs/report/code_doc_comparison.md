# 代码与文档对比详表

**生成时间**: 2026-02-01

本文档详细对比代码实现与技术设计文档的差异。

---

## 1. 算法类型对比

### 代码实现 (`internal/config/rules.go:14-18`)

```go
const (
    RuleTypeCount      = "count"       // 固定窗口计数
    RuleTypeFreq       = "freq"        // 滑动窗口/漏桶
    RuleTypeAccumulate = "accumulate"  // 累积阈值+二级限流
)
```

### 技术设计文档 (`docs/v2-tech-design/05-CONFIG-REFERENCE.md:476`)

```
| type | string | 是 | 算法类型: direct/count/base/leak |
```

### 对比表

| 文档值 | 代码值 | 状态 | 说明 |
|--------|--------|------|------|
| `direct` | - | ❌ 不存在 | 文档有，代码无 |
| `count` | `count` | ✅ 一致 | 固定窗口计数 |
| `base` | `accumulate` | ❌ 名称不同 | 功能相同但名称不一致 |
| `leak` | `freq` | ❌ 名称不同 | 功能相同但名称不一致 |

### 修复建议

更新 `docs/v2-tech-design/05-CONFIG-REFERENCE.md:476`:
```
- | type | string | 是 | 算法类型: direct/count/base/leak |
+ | type | string | 是 | 算法类型: count/freq/accumulate |
```

---

## 2. 存储类型对比

### 代码实现 (`internal/config/config.go:18-19`)

```go
const (
    StorageTypeLocal = "local"
    StorageTypeRedis = "redis"
)
```

### 代码验证 (`internal/config/validate.go:122`)

```go
validTypes := []string{StorageTypeLocal, StorageTypeRedis}
```

### 技术设计文档 (`docs/v2-tech-design/05-CONFIG-REFERENCE.md:48`)

```toml
# 存储类型: redis | redis_cluster | local
type = "redis"
```

### 对比表

| 文档值 | 代码支持 | 状态 |
|--------|----------|------|
| `local` | ✅ | 一致 |
| `redis` | ✅ | 一致 |
| `redis_cluster` | ❌ | 文档有，代码未实现 |

### 修复建议

更新 `docs/v2-tech-design/05-CONFIG-REFERENCE.md:48`:
```toml
- # 存储类型: redis | redis_cluster | local
+ # 存储类型: redis | local
type = "redis"
```

同时删除 `redis_cluster` 相关配置示例（行 62-74）。

---

## 3. 配置节对比

### 用户手册 vs 代码支持

| 配置节 | 代码支持 | 用户手册 | 状态 |
|--------|----------|----------|------|
| `[server]` | ✅ | ✅ | 一致 |
| `[rules]` | ✅ | ✅ | 一致 |
| `[storage]` | ✅ | ✅ | 一致 |
| `[storage.local]` | ✅ | ✅ | 一致 |
| `[storage.redis]` | ✅ | ✅ | 一致 |
| `[storage.fallback]` | ✅ | ❌ | 缺失文档 |
| `[logging]` | ✅ | ✅ | 一致 |
| `[logging.file]` | ✅ | ❌ | 缺失文档 |
| `[metrics]` | ✅ | ✅ | 一致 |
| `[performance]` | 需验证 | ❌ | 可能缺失 |

---

## 4. 匹配器类型对比

### 代码实现 (`internal/engine/matcher/`)

| 文件 | 模式 | 示例 |
|------|------|------|
| `exact.go` | 精确匹配 | `"login"` |
| `any.go` | 任意非空 | `"+"` |
| `not.go` | 取反 | `"!login"` |
| `dict.go` | 字典引用 | `"@whitelist"` |
| `multi.go` | 多值 | `"a,b,c"` |
| `range.go` | 范围 | `"1-100"` |
| `greater.go` | 大于 | `">1000"` |
| `less.go` | 小于 | `"<100"` |
| `ip.go` | IP通配 | `"192.168.*.*"` |

### 用户手册文档

| 模式 | 已文档化 |
|------|----------|
| 精确匹配 | ✅ |
| 任意非空 (`+`) | ✅ |
| 字典引用 (`@`) | ✅ |
| IP通配符 | ✅ |
| 取反 (`!`) | ❌ 未文档化 |
| 多值 (`,`) | ❌ 未文档化 |
| 范围 (`-`) | ❌ 未文档化 |
| 大于 (`>`) | ❌ 未文档化 |
| 小于 (`<`) | ❌ 未文档化 |

---

## 5. API 端点对比

### 代码实现 (`internal/api/router.go`)

| 端点 | 方法 | 代码 | 文档 | 状态 |
|------|------|------|------|------|
| `/health` | GET | ✅ | ✅ | 一致 |
| `/ready` | GET | ✅ | ✅ | 一致 |
| `/metrics` | GET | ✅ | ✅ | 一致 |
| `/api/v1/browse` | POST | ✅ | ✅ | 一致 |
| `/api/v1/update` | POST | ✅ | ✅ | 一致 |
| `/api/v1/batch` | POST | ✅ | ✅ | 一致 |

---

## 6. 错误码对比

### 代码/配置定义 (`conf/rules.toml`)

| 错误码 | 消息 | 在配置中 |
|--------|------|----------|
| 0 | ok | ✅ |
| 4001 | 登录过于频繁 | ✅ |
| 4002 | 该IP登录过于频繁 | ✅ |
| 4003 | IP已被封禁 | ✅ |
| 4004 | 设备已被封禁 | ✅ |
| 4101 | 发帖数量已达上限 | ✅ |
| 4102 | 评论过于频繁 | ✅ |
| 4201 | API调用频率超限 | ✅ |
| 4999 | 操作过于频繁 | ✅ |

### 用户手册错误码表

| 错误码 | 在文档中 | 状态 |
|--------|----------|------|
| 0 | ✅ | 一致 |
| 4001 | ✅ | 一致 |
| 4002 | ⚠️ | 存在但使用场景不明确 |
| 4003 | ✅ | 一致 |
| 4004 | ✅ | 一致 |
| 4101 | ✅ | 一致 |
| 4102 | ✅ | 一致 |
| 4201 | ❌ | 未在用户手册错误码表中 |
| 4999 | ✅ | 一致 |

---

## 7. 测试覆盖验证

### 各模块测试文件存在性

| 模块 | 测试文件 | 存在 |
|------|----------|------|
| storage/local | local_test.go | ✅ |
| storage/redis | redis_test.go | ✅ |
| storage/manager | manager_test.go | ✅ |
| engine | engine_test.go | ✅ |
| engine/matcher | matcher_test.go | ✅ |
| engine/algorithm | algorithm_test.go | ✅ |
| config | config_test.go | ✅ |
| config (rules) | rules_test.go | ✅ |
| config (dict) | dict_test.go | ✅ |
| api | api_test.go | ✅ |

### API 测试文件

| 测试文件 | 测试类型 |
|----------|----------|
| browse_test.go | Browse API E2E |
| update_test.go | Update API E2E |
| batch_test.go | Batch API E2E |
| health_test.go | 健康检查 E2E |
| stress_test.go | 压力测试 |

---

## 总结

### 必须修复

1. **算法类型名称**: `base`→`accumulate`, `leak`→`freq`, 移除 `direct`
2. **存储类型**: 移除 `redis_cluster`

### 建议补充

1. 用户手册添加 `[storage.fallback]` 配置说明
2. 用户手册添加 `[logging.file]` 配置说明
3. 用户手册补充高级匹配语法（取反、多值、范围、比较）
4. 用户手册错误码表添加 4201

### 一致性良好

- API 端点定义
- 请求/响应格式
- 基本匹配语法
- 基本错误码
