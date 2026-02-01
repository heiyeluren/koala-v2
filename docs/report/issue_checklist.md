# 问题清单与修复状态

**更新时间**: 2026-02-01

## 修复状态说明

- ⬜ 待修复
- 🔄 进行中
- ✅ 已完成
- ❌ 不修复（说明原因）

---

## 严重问题 (S)

### S-01: 算法类型名称不一致 ✅

**问题描述**: 技术设计文档中的算法类型名称与代码实现不一致

| 来源 | 值 |
|------|-----|
| 技术设计文档 | `direct`, `count`, `base`, `leak` |
| 实际代码 | `count`, `freq`, `accumulate` |

**影响文件**:
- `docs/v2-tech-design/05-CONFIG-REFERENCE.md`
- `docs/v2-tech-design/03-CORE-ALGORITHMS.md`

**修复方案**:
将文档中的算法类型更新为:
- `count` - 固定窗口计数
- `freq` - 滑动窗口/漏桶
- `accumulate` - 累积阈值 + 二级限流

**修复完成**: 已更新两个技术设计文档中的算法类型名称和相关示例。

---

### S-02: redis_cluster 存储类型未实现 ✅

**问题描述**: 技术设计文档声称支持 `redis_cluster`，但代码未实现

**影响文件**:
- `docs/v2-tech-design/05-CONFIG-REFERENCE.md`
- `docs/v2-tech-design/conf/koala.toml`

**修复方案**: 移除文档中 `redis_cluster` 类型的描述，保持代码使用单一 `addr` 字段。

**修复完成**:
- 移除了 `redis_cluster` 存储类型相关文档
- 保持使用 `addr` 单地址配置

---

## 重要问题 (I)

### I-01: 用户手册缺少 storage.fallback 配置说明 ✅

**问题描述**: 代码支持降级存储配置，但用户手册未说明

**影响文件**:
- `docs/user_manual.md`

**修复完成**: 已添加 `[storage.fallback]` 配置节和降级存储说明。

---

### I-02: 用户手册缺少 logging.file 配置说明 ✅

**问题描述**: 日志文件轮转配置未文档化

**影响文件**:
- `docs/user_manual.md`

**修复完成**: 已添加 `[logging.file]` 配置节和字段说明表。

---

### I-03: 测试缺少规则优先级验证 ✅

**问题描述**: 缺少验证多规则匹配优先级的测试

**影响文件**:
- `internal/engine/engine_test.go`

**修复完成**: 添加了4个新测试用例:
- `TestEngine_Check_AllPhasePriority` - 测试所有4个阶段的优先级
- `TestEngine_Check_PostVsAdvancedPriority` - 测试 Post > Advanced
- `TestEngine_Check_AdvancedVsDefaultPriority` - 测试 Advanced > Default
- `TestEngine_Check_DefaultFallback` - 测试默认规则兜底

---

### I-04: 测试缺少故障转移边缘情况 ⬜

**问题描述**: 存储管理器缺少故障边缘情况测试

**影响文件**:
- `internal/storage/manager/manager_test.go`

**修复方案**: 添加测试用例:
- `TestStorageManager_PartialFailure`
- `TestStorageManager_BothStoragesFail`
- `TestStorageManager_ConnectionTimeout`
- `TestStorageManager_DataConsistency`

---

### I-05: 错误码 4002 文档不清晰 ✅

**问题描述**: 用户手册列出错误码 4002，但使用场景不明确

**影响文件**:
- `docs/user_manual.md`

**修复完成**: 更新错误码表，添加"触发条件"列，明确说明 4002 用于 IP 级别登录限流。

---

## 一般问题 (M)

### M-01: TOCTOU 竞态条件未文档化 ⬜

**问题描述**: Browse → Update 之间的时间窗口竞态未在文档中说明

**修复方案**: 在技术设计文档中添加说明:
> 注意：Browse 和 Update 是分开的操作，在高并发场景下可能存在时间窗口竞态。
> 这是设计预期的行为，系统采用最终一致性模型。

---

### M-02: 降级数据一致性未说明 ⬜

**问题描述**: Redis 故障降级后数据如何处理未说明

**修复方案**: 补充降级策略说明:
- 降级期间计数器数据存储在本地
- Redis 恢复后，新请求使用 Redis（不同步历史数据）
- 本地数据在 TTL 过期后自动清理

---

### M-03: 批量请求原子性未说明 ⬜

**问题描述**: 批量 API 的原子性语义未明确

**修复方案**: 在 API 文档中添加说明:
> 批量请求不保证原子性，每个请求独立处理，部分成功是可能的。

---

### M-04: 冗余测试代码 ⬜

**问题描述**: 多处测试重复

| 测试类型 | 重复位置 |
|----------|----------|
| 健康检查 | `health_test.go`, `api_test.go` |
| CORS | `health_test.go`, `api_test.go` |
| 响应格式 | `browse_test.go`, `update_test.go`, `batch_test.go` |

**修复方案**: 合并重复测试，保留一处

---

### M-05: 高级匹配语法未文档化 ⬜

**问题描述**: 用户手册未说明范围匹配、比较匹配等高级语法

**修复方案**: 在用户手册匹配模式章节添加:
| 模式 | 示例 | 说明 |
|------|------|------|
| 范围 | `"1-100"` | 数值在范围内 |
| 大于 | `">1000"` | 数值大于阈值 |
| 小于 | `"<100"` | 数值小于阈值 |

---

### M-06: 性能配置节未文档化 ⬜

**问题描述**: `[performance]` 配置节在代码中支持但未文档化

**修复方案**: 如果确实支持，添加文档说明

---

### M-07: 规则 desc 字段未文档化 ⬜

**问题描述**: 规则的 `desc` 描述字段未在用户手册中说明

**修复方案**: 在规则配置示例中添加:
```toml
[[rules.business]]
name = "login_rate_limit"
desc = "登录频率限制：每分钟最多5次"  # 可选描述字段
type = "count"
...
```

---

### M-08: 字典文件注释支持未说明 ⬜

**问题描述**: 字典文件支持 `#` 注释但未文档化

**修复方案**: 在字典文件格式说明中添加:
```
# 这是注释行
vip_user_001
vip_user_002  # 行内注释也支持
```

---

## 优化建议 (O)

### O-01: 添加指标验证测试 ⬜

**建议**: 添加 `test/api/metrics_test.go` 验证 Prometheus 指标准确性

---

### O-02: 添加性能回归测试 ⬜

**建议**: 在 CI 中添加性能基准测试，防止性能退化

---

### O-03: 添加模糊测试 ⬜

**建议**: 使用 go-fuzz 对输入验证进行模糊测试

---

### O-04: 统一测试并发级别 ⬜

**建议**: 将分散的并发测试统一到 `stress_test.go`

---

## 统计

| 级别 | 总数 | 已完成 | 进行中 | 待处理 |
|------|------|--------|--------|--------|
| 严重 (S) | 2 | 2 | 0 | 0 |
| 重要 (I) | 5 | 4 | 0 | 1 |
| 一般 (M) | 8 | 0 | 0 | 8 |
| 建议 (O) | 4 | 0 | 0 | 4 |
| **总计** | **19** | **6** | **0** | **13** |
