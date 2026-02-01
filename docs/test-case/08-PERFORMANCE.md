# 08 - 性能测试

## 1. 测试范围

性能基准测试，验证系统满足性能指标要求。

---

## 2. 性能指标目标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| QPS | ≥ 100,000 | 每秒请求数 |
| P50 延迟 | ≤ 1ms | 50% 请求延迟 |
| P99 延迟 | ≤ 10ms | 99% 请求延迟 |
| P999 延迟 | ≤ 50ms | 99.9% 请求延迟 |
| 内存 | ≤ 1GB | 常驻内存 |
| CPU | ≤ 50% | 8核机器负载 |

---

## 3. 基准测试

### 3.1 吞吐量测试

#### TC-PF-001-Throughput_LocalStorage 🔴
```
描述: 本地存储吞吐量
测试环境:
  - 存储: LocalStorage (Ristretto)
  - 规则: 10 条
  - 机器: 8 核 CPU
测试方法:
  - 工具: go test -bench
  - 并发: GOMAXPROCS=8
  - 持续: 30 秒
测试步骤:
  1. 预热 10 秒
  2. 运行基准测试
  3. 记录 QPS
预期结果:
  - QPS ≥ 200,000（本地存储）
```

#### TC-PF-002-Throughput_Redis 🔴
```
描述: Redis 存储吞吐量
测试环境:
  - 存储: Redis (本地)
  - 规则: 10 条
  - Redis: 单实例
测试方法:
  - 并发: 100 goroutines
  - 持续: 30 秒
预期结果:
  - QPS ≥ 50,000（受 Redis 限制）
```

#### TC-PF-003-Throughput_MixedWorkload 🔴
```
描述: 混合负载吞吐量
测试环境:
  - 80% Browse + 20% Update
  - 存储: LocalStorage
测试方法:
  - 模拟真实流量比例
预期结果:
  - QPS ≥ 150,000
```

### 3.2 延迟测试

#### TC-PF-004-Latency_P50 🔴
```
描述: P50 延迟
测试方法:
  - 发送 100,000 次请求
  - 记录每次延迟
  - 计算 P50
预期结果:
  - P50 ≤ 1ms（本地存储）
  - P50 ≤ 2ms（Redis 存储）
```

#### TC-PF-005-Latency_P99 🔴
```
描述: P99 延迟
测试方法:
  - 发送 100,000 次请求
  - 计算 P99
预期结果:
  - P99 ≤ 10ms
```

#### TC-PF-006-Latency_P999 🔴
```
描述: P999 延迟
测试方法:
  - 发送 1,000,000 次请求
  - 计算 P999
预期结果:
  - P999 ≤ 50ms
```

#### TC-PF-007-Latency_Distribution 🟡
```
描述: 延迟分布
测试方法:
  - 生成延迟直方图
预期结果:
  - 无明显长尾
  - 大部分请求 < 5ms
```

---

## 4. 压力测试

### 4.1 持续压力

#### TC-PF-008-Stress_Sustained 🔴
```
描述: 持续高压力测试
测试方法:
  - 持续 10 分钟
  - 保持 80% 目标 QPS
测试步骤:
  1. 逐步增加压力到目标
  2. 持续运行 10 分钟
  3. 监控各项指标
预期结果:
  - 无内存泄漏
  - 延迟稳定
  - 无错误
```

#### TC-PF-009-Stress_Spike 🔴
```
描述: 突发流量测试
测试方法:
  - 正常流量运行
  - 突然增加 5 倍流量
  - 持续 1 分钟
  - 恢复正常流量
预期结果:
  - 系统不崩溃
  - 恢复后指标正常
```

### 4.2 资源极限

#### TC-PF-010-Stress_MaxQPS 🟡
```
描述: 最大 QPS 测试
测试方法:
  - 逐步增加压力直到系统饱和
  - 记录最大 QPS
预期结果:
  - 记录系统极限 QPS
  - 识别瓶颈
```

#### TC-PF-011-Stress_MaxConnections 🟡
```
描述: 最大连接数测试
测试方法:
  - 逐步增加并发连接数
  - 记录最大支持连接数
预期结果:
  - 记录最大连接数
```

---

## 5. 资源测试

### 5.1 内存测试

#### TC-PF-012-Memory_Baseline 🔴
```
描述: 内存基线
测试方法:
  - 启动服务
  - 记录初始内存
预期结果:
  - 初始内存 < 100MB
```

#### TC-PF-013-Memory_UnderLoad 🔴
```
描述: 负载下内存
测试方法:
  - 持续压力 10 分钟
  - 监控内存变化
预期结果:
  - 内存 < 1GB
  - 无持续增长（内存泄漏）
```

#### TC-PF-014-Memory_LargeDict 🟡
```
描述: 大字典内存占用
测试方法:
  - 加载 100 万条目的字典
  - 测量内存增量
预期结果:
  - 每 100 万条目 < 100MB
```

### 5.2 CPU 测试

#### TC-PF-015-CPU_UnderLoad 🔴
```
描述: 负载下 CPU
测试方法:
  - 目标 QPS 压力
  - 监控 CPU 使用率
预期结果:
  - CPU < 50%（8核机器）
```

---

## 6. 算法性能测试

### 6.1 各算法基准

#### TC-PF-016-Algo_Direct_Bench 🔴
```
描述: Direct 算法基准
测试方法:
  go test -bench=BenchmarkDirectBrowse
预期结果:
  - < 100ns/op
```

#### TC-PF-017-Algo_Count_Bench 🔴
```
描述: Count 算法基准
测试方法:
  go test -bench=BenchmarkCountBrowse
预期结果:
  - < 500ns/op（本地存储）
```

#### TC-PF-018-Algo_Base_Bench 🔴
```
描述: Base 算法基准
测试方法:
  go test -bench=BenchmarkBaseBrowse
预期结果:
  - < 1μs/op（本地存储）
```

#### TC-PF-019-Algo_Leak_Bench 🔴
```
描述: Leak 算法基准
测试方法:
  go test -bench=BenchmarkLeakBrowse
预期结果:
  - < 2μs/op（本地存储）
```

### 6.2 匹配器性能

#### TC-PF-020-Matcher_Dict_Bench 🔴
```
描述: 字典匹配器基准
测试方法:
  - 字典包含 100 万条目
  go test -bench=BenchmarkDictMatcher
预期结果:
  - < 50ns/op（hash 查找）
```

---

## 7. 并发性能测试

#### TC-PF-021-Concurrent_Scaling 🔴
```
描述: 并发扩展性
测试方法:
  - 分别测试 1, 2, 4, 8, 16, 32 并发
  - 记录各并发级别的 QPS
预期结果:
  - 近线性扩展（直到 CPU 饱和）
```

#### TC-PF-022-Concurrent_Contention 🟡
```
描述: 并发竞争
测试方法:
  - 高并发访问同一 key
  - 测量锁竞争影响
预期结果:
  - 性能下降可控
```

---

## 8. 热重载性能测试

#### TC-PF-023-HotReload_Duration 🔴
```
描述: 热重载耗时
测试方法:
  - 配置 1000 条规则
  - 触发热重载
  - 测量耗时
预期结果:
  - 重载耗时 < 500ms
```

#### TC-PF-024-HotReload_NoDowntime 🔴
```
描述: 热重载无停机
测试方法:
  - 持续发送请求
  - 触发热重载
  - 检查请求是否失败
预期结果:
  - 无请求失败
  - 延迟轻微抖动可接受
```

---

## 9. 测试报告模板

```markdown
# 性能测试报告

## 测试环境
- 机器配置: 8核 CPU, 16GB 内存
- 操作系统: Linux 5.x
- Go 版本: 1.21
- 存储类型: LocalStorage / Redis

## 测试结果

### 吞吐量
| 测试项 | 结果 | 目标 | 状态 |
|--------|------|------|------|
| 本地存储 QPS | xxx | 100,000 | ✅/❌ |
| Redis QPS | xxx | 50,000 | ✅/❌ |

### 延迟
| 百分位 | 结果 | 目标 | 状态 |
|--------|------|------|------|
| P50 | xxx ms | 1ms | ✅/❌ |
| P99 | xxx ms | 10ms | ✅/❌ |
| P999 | xxx ms | 50ms | ✅/❌ |

### 资源
| 指标 | 结果 | 目标 | 状态 |
|------|------|------|------|
| 内存 | xxx MB | 1GB | ✅/❌ |
| CPU | xxx % | 50% | ✅/❌ |

## 结论
[总结性能测试结果]
```

---

## 10. 性能测试代码示例

```go
// test/benchmark/browse_test.go

package benchmark

import (
    "context"
    "testing"

    "koala/internal/engine"
    "koala/internal/storage/local"
)

func BenchmarkBrowse_LocalStorage(b *testing.B) {
    store, _ := local.New(local.DefaultConfig())
    eng := engine.New(store)
    eng.LoadPolicy(testPolicy)

    req := &engine.Request{
        Act: "post",
        UID: "12345",
    }

    ctx := context.Background()

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            eng.Evaluate(ctx, req)
        }
    })
}

func BenchmarkBrowse_WithUpdate(b *testing.B) {
    store, _ := local.New(local.DefaultConfig())
    eng := engine.New(store)
    eng.LoadPolicy(testPolicy)

    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        req := &engine.Request{
            Act: "post",
            UID: fmt.Sprintf("user_%d", i%1000),
        }
        eng.Evaluate(ctx, req)
        eng.Update(ctx, req)
    }
}
```
