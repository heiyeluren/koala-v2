# 02 - 算法单元测试

## 1. 测试范围

测试 `internal/engine/algorithm/` 下的四种限流算法：
- Direct 算法
- Count 算法
- Base 算法
- Leak 算法

---

## 2. Direct 算法测试

### 2.1 Browse 测试

#### TC-AL-001-Direct_Browse_AlwaysHit 🔴
```
描述: Direct 算法 Browse 总是返回命中
前置条件: Direct 算法实例已创建
测试步骤:
  1. 调用 Browse(ctx, "any_key", limit, storage)
预期结果:
  - 返回 hit=true, err=nil
说明: Direct 算法用于白名单/黑名单，到达 Browse 说明已匹配
```

### 2.2 Update 测试

#### TC-AL-002-Direct_Update_NoOp 🔴
```
描述: Direct 算法 Update 是空操作
前置条件: Direct 算法实例已创建
测试步骤:
  1. 调用 Update(ctx, "any_key", limit, storage)
预期结果:
  - 返回 nil
  - 存储无变化
```

---

## 3. Count 算法测试

### 3.1 Browse 测试

#### TC-AL-003-Count_Browse_FirstRequest 🔴
```
描述: Count 算法首次请求，计数器不存在
前置条件: Count 算法实例，key 不存在
测试步骤:
  1. 调用 Browse(ctx, "count_key1", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=false（未超限）
```

#### TC-AL-004-Count_Browse_UnderLimit 🔴
```
描述: Count 算法计数未达限制
前置条件: Count 算法实例，当前计数=3，限制=5
测试步骤:
  1. 设置 storage 中 key 的值为 3
  2. 调用 Browse(ctx, "count_key2", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=false（3 < 5，未超限）
```

#### TC-AL-005-Count_Browse_AtLimit 🔴
```
描述: Count 算法计数等于限制
前置条件: Count 算法实例，当前计数=5，限制=5
测试步骤:
  1. 设置 storage 中 key 的值为 5
  2. 调用 Browse(ctx, "count_key3", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=true（5 >= 5，超限）
```

#### TC-AL-006-Count_Browse_OverLimit 🔴
```
描述: Count 算法计数超过限制
前置条件: Count 算法实例，当前计数=10，限制=5
测试步骤:
  1. 设置 storage 中 key 的值为 10
  2. 调用 Browse(ctx, "count_key4", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=true（10 >= 5，超限）
```

### 3.2 Update 测试

#### TC-AL-007-Count_Update_FirstRequest 🔴
```
描述: Count 算法首次更新，创建计数器
前置条件: Count 算法实例，key 不存在
测试步骤:
  1. 调用 Update(ctx, "count_key5", {time: 60s, count: 5}, storage)
  2. 调用 storage.GetInt("count_key5")
预期结果:
  - Update 返回 nil
  - GetInt 返回 1
```

#### TC-AL-008-Count_Update_Increment 🔴
```
描述: Count 算法增量更新
前置条件: Count 算法实例，当前计数=3
测试步骤:
  1. 设置 storage 中 key 的值为 3
  2. 调用 Update(ctx, "count_key6", {time: 60s, count: 5}, storage)
  3. 调用 storage.GetInt("count_key6")
预期结果:
  - GetInt 返回 4
```

#### TC-AL-009-Count_Update_WithTTL 🔴
```
描述: Count 算法更新设置 TTL
前置条件: Count 算法实例，key 不存在
测试步骤:
  1. 调用 Update(ctx, "count_key7", {time: 100ms, count: 5}, storage)
  2. 等待 150ms
  3. 调用 storage.GetInt("count_key7")
预期结果:
  - GetInt 返回 0, ErrKeyNotFound
```

### 3.3 自然天处理测试

#### TC-AL-010-Count_NaturalDay_TTL 🔴
```
描述: Count 算法 24h 时间窗口使用自然天
前置条件: Count 算法实例
测试步骤:
  1. 调用 Update(ctx, "count_key8", {time: 24h, count: 5}, storage)
  2. 检查 key 的 TTL
预期结果:
  - TTL 应该是到当天 23:59:59 的剩余秒数
  - 而不是固定的 86400 秒
```

#### TC-AL-011-Count_NaturalDay_Midnight 🟡
```
描述: Count 算法在午夜附近的行为
前置条件: Count 算法实例，时间接近午夜
测试步骤:
  1. 在 23:59:50 调用 Update
  2. 等待跨过午夜
  3. 调用 Browse
预期结果:
  - 新的一天计数器已重置
```

### 3.4 并发测试

#### TC-AL-012-Count_Concurrent_Update 🔴
```
描述: Count 算法并发更新
前置条件: Count 算法实例
测试步骤:
  1. 启动 100 个 goroutine
  2. 每个 goroutine 调用 Update 10 次
  3. 调用 storage.GetInt
预期结果:
  - GetInt 返回 1000
```

#### TC-AL-013-Count_Concurrent_BrowseUpdate 🔴
```
描述: Count 算法并发 Browse 和 Update
前置条件: Count 算法实例，limit=50
测试步骤:
  1. 启动 100 个 goroutine
  2. 每个 goroutine 先 Browse 再 Update
  3. 统计 hit=true 的次数
预期结果:
  - 至少有 50 次 hit=true
  - 计数器最终值为 100
```

---

## 4. Base 算法测试

### 4.1 Browse 测试（未达基础阈值）

#### TC-AL-014-Base_Browse_UnderBase 🔴
```
描述: Base 算法主计数器未达基础阈值
前置条件: Base 算法实例，base=10，当前计数=5
测试步骤:
  1. 设置主计数器值为 5
  2. 调用 Browse(ctx, "base_key1", {base: 10, time: 5s, count: 1}, storage)
预期结果:
  - 返回 hit=false（主计数器 < base，直接放行）
```

#### TC-AL-015-Base_Browse_AtBase_NoSecondary 🔴
```
描述: Base 算法达到基础阈值，次级计数器不存在
前置条件: Base 算法实例，base=10，主计数器=10
测试步骤:
  1. 设置主计数器值为 10
  2. 调用 Browse(ctx, "base_key2", {base: 10, time: 5s, count: 1}, storage)
预期结果:
  - 返回 hit=false（次级计数器不存在或为0）
```

### 4.2 Browse 测试（已达基础阈值）

#### TC-AL-016-Base_Browse_OverBase_UnderSecondary 🔴
```
描述: Base 算法超过基础阈值，次级计数器未达限制
前置条件: Base 算法实例，base=10，主=15，次=0
测试步骤:
  1. 设置主计数器值为 15
  2. 设置次级计数器值为 0
  3. 调用 Browse(ctx, "base_key3", {base: 10, time: 5s, count: 2}, storage)
预期结果:
  - 返回 hit=false（次级计数器 0 < 2）
```

#### TC-AL-017-Base_Browse_OverBase_AtSecondary 🔴
```
描述: Base 算法超过基础阈值，次级计数器达到限制
前置条件: Base 算法实例，base=10，主=15，次=2
测试步骤:
  1. 设置主计数器值为 15
  2. 设置次级计数器值为 2
  3. 调用 Browse(ctx, "base_key4", {base: 10, time: 5s, count: 2}, storage)
预期结果:
  - 返回 hit=true（次级计数器 2 >= 2，超限）
```

### 4.3 Update 测试

#### TC-AL-018-Base_Update_UnderBase 🔴
```
描述: Base 算法更新，未达基础阈值
前置条件: Base 算法实例，主计数器=5，base=10
测试步骤:
  1. 设置主计数器值为 5
  2. 调用 Update(ctx, "base_key5", {base: 10, time: 5s, count: 1}, storage)
  3. 检查主计数器和次级计数器
预期结果:
  - 主计数器=6
  - 次级计数器不存在（未创建）
```

#### TC-AL-019-Base_Update_CrossBase 🔴
```
描述: Base 算法更新，跨过基础阈值
前置条件: Base 算法实例，主计数器=9，base=10
测试步骤:
  1. 设置主计数器值为 9
  2. 调用 Update(ctx, "base_key6", {base: 10, time: 5s, count: 1}, storage)
  3. 检查主计数器和次级计数器
预期结果:
  - 主计数器=10
  - 次级计数器=1（刚创建）
```

#### TC-AL-020-Base_Update_OverBase 🔴
```
描述: Base 算法更新，已超过基础阈值
前置条件: Base 算法实例，主计数器=15，次级=1，base=10
测试步骤:
  1. 设置主计数器值为 15
  2. 设置次级计数器值为 1
  3. 调用 Update(ctx, "base_key7", {base: 10, time: 5s, count: 2}, storage)
  4. 检查主计数器和次级计数器
预期结果:
  - 主计数器=16
  - 次级计数器=2
```

### 4.4 次级计数器 TTL 测试

#### TC-AL-021-Base_Secondary_TTL_Expire 🔴
```
描述: Base 算法次级计数器过期重置
前置条件: Base 算法实例
测试步骤:
  1. 设置主计数器=15
  2. 调用 Update（次级 TTL=100ms）
  3. 等待 150ms
  4. 调用 Browse
预期结果:
  - Browse 返回 hit=false（次级计数器已过期重置）
```

#### TC-AL-022-Base_Secondary_TTL_NotExtend 🔴
```
描述: Base 算法次级计数器 TTL 不延长
前置条件: Base 算法实例
测试步骤:
  1. 设置主计数器=15
  2. 调用 Update（次级 TTL=500ms）
  3. 等待 300ms
  4. 调用 Update
  5. 等待 250ms
  6. 调用 Browse
预期结果:
  - Browse 返回 hit=false（次级计数器在第一次的 500ms 后过期）
```

---

## 5. Leak 算法测试

### 5.1 Browse 测试

#### TC-AL-023-Leak_Browse_Empty 🔴
```
描述: Leak 算法空列表
前置条件: Leak 算法实例，列表不存在
测试步骤:
  1. 调用 Browse(ctx, "leak_key1", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=false（列表长度 0 <= 5）
```

#### TC-AL-024-Leak_Browse_UnderCount 🔴
```
描述: Leak 算法列表长度未达限制
前置条件: Leak 算法实例，列表有 3 个元素
测试步骤:
  1. 向列表添加 3 个时间戳
  2. 调用 Browse(ctx, "leak_key2", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=false（列表长度 3 <= 5）
```

#### TC-AL-025-Leak_Browse_AtCount_OldTimestamp 🔴
```
描述: Leak 算法列表长度超过限制，但时间戳已过期
前置条件: Leak 算法实例，列表有 6 个元素
测试步骤:
  1. 向列表添加 6 个 70 秒前的时间戳
  2. 调用 Browse(ctx, "leak_key3", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=false（第 5 个元素时间戳已超过 60s）
```

#### TC-AL-026-Leak_Browse_AtCount_RecentTimestamp 🔴
```
描述: Leak 算法列表长度超过限制，时间戳在窗口内
前置条件: Leak 算法实例，列表有 6 个元素
测试步骤:
  1. 向列表添加 6 个 30 秒前的时间戳
  2. 调用 Browse(ctx, "leak_key4", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=true（第 5 个元素时间戳在 60s 内）
```

#### TC-AL-027-Leak_Browse_EdgeCase 🔴
```
描述: Leak 算法边界情况（时间戳刚好等于窗口）
前置条件: Leak 算法实例
测试步骤:
  1. 向列表添加 6 个刚好 60 秒前的时间戳
  2. 调用 Browse(ctx, "leak_key5", {time: 60s, count: 5}, storage)
预期结果:
  - 返回 hit=true（等于边界时仍算命中）
```

### 5.2 Update 测试

#### TC-AL-028-Leak_Update_AddTimestamp 🔴
```
描述: Leak 算法更新添加时间戳
前置条件: Leak 算法实例
测试步骤:
  1. 调用 Update(ctx, "leak_key6", {time: 60s, count: 5}, storage)
  2. 调用 storage.LLen("leak_key6")
  3. 调用 storage.LIndex("leak_key6", 0)
预期结果:
  - LLen 返回 1
  - LIndex(0) 返回当前时间戳（±1秒误差）
```

#### TC-AL-029-Leak_Update_MultipleAdds 🔴
```
描述: Leak 算法多次更新
前置条件: Leak 算法实例
测试步骤:
  1. 调用 Update 5 次，每次间隔 10ms
  2. 调用 storage.LLen("leak_key7")
预期结果:
  - LLen 返回 5
```

### 5.3 清理测试

#### TC-AL-030-Leak_Cleanup_TrimList 🔴
```
描述: Leak 算法清理多余元素
前置条件: Leak 算法实例，count=5
测试步骤:
  1. 向列表添加 10 个时间戳
  2. 调用 Update（触发清理）
  3. 等待异步清理完成
  4. 调用 storage.LLen
预期结果:
  - LLen 返回 6（count+1）
```

#### TC-AL-031-Leak_Cleanup_Async 🟡
```
描述: Leak 算法清理是异步的
前置条件: Leak 算法实例
测试步骤:
  1. 向列表添加 100 个时间戳
  2. 调用 Update
  3. 立即检查 LLen
  4. 等待 100ms
  5. 再次检查 LLen
预期结果:
  - 第一次 LLen 可能 > count+1
  - 第二次 LLen 应该 = count+1
```

### 5.4 并发测试

#### TC-AL-032-Leak_Concurrent_Update 🔴
```
描述: Leak 算法并发更新
前置条件: Leak 算法实例
测试步骤:
  1. 启动 100 个 goroutine
  2. 每个 goroutine 调用 Update
  3. 等待所有完成和清理
  4. 调用 storage.LLen
预期结果:
  - LLen <= count+1（清理后）
```

#### TC-AL-033-Leak_Concurrent_BrowseUpdate 🔴
```
描述: Leak 算法并发 Browse 和 Update
前置条件: Leak 算法实例，count=10
测试步骤:
  1. 启动 50 个 goroutine 调用 Update
  2. 同时启动 50 个 goroutine 调用 Browse
  3. 统计结果
预期结果:
  - 无竞态条件错误
  - 结果符合预期
```

---

## 6. 算法工厂测试

#### TC-AL-034-Factory_Create_Direct 🔴
```
描述: 工厂创建 Direct 算法
测试步骤:
  1. 调用 algorithm.New("direct")
预期结果:
  - 返回 *DirectAlgorithm 实例
```

#### TC-AL-035-Factory_Create_Count 🔴
```
描述: 工厂创建 Count 算法
测试步骤:
  1. 调用 algorithm.New("count")
预期结果:
  - 返回 *CountAlgorithm 实例
```

#### TC-AL-036-Factory_Create_Base 🔴
```
描述: 工厂创建 Base 算法
测试步骤:
  1. 调用 algorithm.New("base")
预期结果:
  - 返回 *BaseAlgorithm 实例
```

#### TC-AL-037-Factory_Create_Leak 🔴
```
描述: 工厂创建 Leak 算法
测试步骤:
  1. 调用 algorithm.New("leak")
预期结果:
  - 返回 *LeakAlgorithm 实例
```

#### TC-AL-038-Factory_Create_Unknown 🟡
```
描述: 工厂创建未知类型
测试步骤:
  1. 调用 algorithm.New("unknown")
预期结果:
  - 返回默认算法（Count）或错误
```

---

## 7. 缓存键生成测试

#### TC-AL-039-CacheKey_Basic 🔴
```
描述: 基本缓存键生成
测试步骤:
  1. 调用 GenerateCacheKey("rule1", {act: "post", uid: "123"}, ["act", "uid"])
预期结果:
  - 返回 "koala:rule1:act=post|uid=123"
```

#### TC-AL-040-CacheKey_Sorted 🔴
```
描述: 缓存键参数排序
测试步骤:
  1. 调用 GenerateCacheKey("rule1", {uid: "123", act: "post"}, ["uid", "act"])
预期结果:
  - 返回 "koala:rule1:act=post|uid=123"（按字母排序）
```

#### TC-AL-041-CacheKey_PartialParams 🔴
```
描述: 缓存键部分参数
测试步骤:
  1. 调用 GenerateCacheKey("rule1", {act: "post", uid: "123", ip: "1.1.1.1"}, ["act", "uid"])
预期结果:
  - 返回 "koala:rule1:act=post|uid=123"（只包含匹配键）
```

#### TC-AL-042-CacheKey_MissingParam 🟡
```
描述: 缓存键缺少参数
测试步骤:
  1. 调用 GenerateCacheKey("rule1", {act: "post"}, ["act", "uid"])
预期结果:
  - 返回 "koala:rule1:act=post"（跳过缺失参数）
```

---

## 8. 错误处理测试

#### TC-AL-043-Count_StorageError 🔴
```
描述: Count 算法存储错误处理
前置条件: Mock 存储返回错误
测试步骤:
  1. 配置存储返回 error
  2. 调用 Browse
预期结果:
  - 返回 hit=false, err!=nil
```

#### TC-AL-044-Base_StorageError 🔴
```
描述: Base 算法存储错误处理
前置条件: Mock 存储返回错误
测试步骤:
  1. 配置存储返回 error
  2. 调用 Browse
预期结果:
  - 返回 hit=false, err!=nil
```

#### TC-AL-045-Leak_StorageError 🔴
```
描述: Leak 算法存储错误处理
前置条件: Mock 存储返回错误
测试步骤:
  1. 配置存储返回 error
  2. 调用 Browse
预期结果:
  - 返回 hit=false, err!=nil
```

---

## 9. 性能测试

#### TC-AL-046-Count_Performance 🟡
```
描述: Count 算法性能
测试步骤:
  1. 运行 1000000 次 Browse
  2. 计算平均耗时
预期结果:
  - 平均耗时 < 1μs（本地存储）
```

#### TC-AL-047-Leak_Performance 🟡
```
描述: Leak 算法性能
测试步骤:
  1. 运行 1000000 次 Browse
  2. 计算平均耗时
预期结果:
  - 平均耗时 < 5μs（本地存储）
```

---

## 10. 回归测试

#### TC-AL-048-Count_Regression_OffByOne 🔴
```
描述: 回归测试 - Count 边界条件
测试步骤:
  1. 设置 count=5
  2. 更新 4 次
  3. Browse（应该放行）
  4. 更新 1 次
  5. Browse（应该拦截）
预期结果:
  - 第一次 Browse hit=false
  - 第二次 Browse hit=true
```

#### TC-AL-049-Base_Regression_SecondaryCreate 🔴
```
描述: 回归测试 - Base 次级计数器创建时机
测试步骤:
  1. 设置 base=10
  2. 更新到 9
  3. 检查次级计数器不存在
  4. 更新到 10
  5. 检查次级计数器存在
预期结果:
  - 第 9 次更新后，次级计数器不存在
  - 第 10 次更新后，次级计数器=1
```

#### TC-AL-050-Leak_Regression_IndexBoundary 🔴
```
描述: 回归测试 - Leak 索引边界
测试步骤:
  1. 设置 count=5
  2. 添加刚好 5 个时间戳
  3. Browse（应该放行）
  4. 添加第 6 个时间戳
  5. Browse（可能拦截）
预期结果:
  - 5 个元素时放行（length <= count）
  - 6 个元素时根据时间戳决定
```

#### TC-AL-051-Algorithm_Context_Cancel 🟡
```
描述: 测试 Context 取消
测试步骤:
  1. 创建可取消的 Context
  2. 取消 Context
  3. 调用 Browse
预期结果:
  - 返回 context.Canceled 错误
```

#### TC-AL-052-Algorithm_Context_Timeout 🟡
```
描述: 测试 Context 超时
测试步骤:
  1. 创建 1ms 超时的 Context
  2. 模拟存储操作延迟 10ms
  3. 调用 Browse
预期结果:
  - 返回 context.DeadlineExceeded 错误
```
