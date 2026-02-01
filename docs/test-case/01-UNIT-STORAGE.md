# 01 - 存储层单元测试

## 1. 测试范围

测试 `internal/storage/` 下的所有存储实现：
- 本地存储 (LocalStorage)
- Redis 存储 (RedisStorage)
- 存储管理器 (StorageManager)

---

## 2. 本地存储测试 (LocalStorage)

### 2.1 String 操作测试

#### TC-ST-001-LocalStorage_Set_Get_Success 🔴
```
描述: 测试本地存储 Set 和 Get 基本功能
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 Set("key1", "value1", 60s)
  2. 调用 Get("key1")
预期结果:
  - Set 返回 nil
  - Get 返回 "value1", nil
```

#### TC-ST-002-LocalStorage_Get_NotFound 🔴
```
描述: 测试获取不存在的 key
前置条件: LocalStorage 已初始化，key2 不存在
测试步骤:
  1. 调用 Get("key2")
预期结果:
  - 返回 "", ErrKeyNotFound
```

#### TC-ST-003-LocalStorage_Set_WithTTL_Expire 🔴
```
描述: 测试 TTL 过期功能
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 Set("key3", "value3", 100ms)
  2. 等待 150ms
  3. 调用 Get("key3")
预期结果:
  - Get 返回 "", ErrKeyNotFound
```

#### TC-ST-004-LocalStorage_Set_Overwrite 🔴
```
描述: 测试覆盖写入
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 Set("key4", "value4a", 60s)
  2. 调用 Set("key4", "value4b", 60s)
  3. 调用 Get("key4")
预期结果:
  - Get 返回 "value4b", nil
```

#### TC-ST-005-LocalStorage_Delete_Success 🔴
```
描述: 测试删除功能
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 Set("key5", "value5", 60s)
  2. 调用 Delete("key5")
  3. 调用 Get("key5")
预期结果:
  - Delete 返回 nil
  - Get 返回 "", ErrKeyNotFound
```

#### TC-ST-006-LocalStorage_Delete_NotExist 🟡
```
描述: 测试删除不存在的 key
前置条件: LocalStorage 已初始化，key6 不存在
测试步骤:
  1. 调用 Delete("key6")
预期结果:
  - 返回 nil（不报错）
```

#### TC-ST-007-LocalStorage_Exists_True 🔴
```
描述: 测试 Exists 返回 true
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 Set("key7", "value7", 60s)
  2. 调用 Exists("key7")
预期结果:
  - Exists 返回 true, nil
```

#### TC-ST-008-LocalStorage_Exists_False 🔴
```
描述: 测试 Exists 返回 false
前置条件: LocalStorage 已初始化，key8 不存在
测试步骤:
  1. 调用 Exists("key8")
预期结果:
  - Exists 返回 false, nil
```

#### TC-ST-009-LocalStorage_Expire_Success 🟡
```
描述: 测试修改过期时间
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 Set("key9", "value9", 60s)
  2. 调用 Expire("key9", 100ms)
  3. 等待 150ms
  4. 调用 Get("key9")
预期结果:
  - Get 返回 "", ErrKeyNotFound
```

### 2.2 计数器操作测试

#### TC-ST-010-LocalStorage_Incr_NewKey 🔴
```
描述: 测试对新 key 的 Incr
前置条件: LocalStorage 已初始化，key10 不存在
测试步骤:
  1. 调用 Incr("key10")
预期结果:
  - 返回 1, nil
```

#### TC-ST-011-LocalStorage_Incr_Existing 🔴
```
描述: 测试对已存在 key 的 Incr
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 SetInt("key11", 5, 60s)
  2. 调用 Incr("key11")
  3. 调用 Incr("key11")
预期结果:
  - 第一次 Incr 返回 6, nil
  - 第二次 Incr 返回 7, nil
```

#### TC-ST-012-LocalStorage_IncrWithTTL_NewKey 🔴
```
描述: 测试带 TTL 的 Incr（新 key）
前置条件: LocalStorage 已初始化，key12 不存在
测试步骤:
  1. 调用 IncrWithTTL("key12", 100ms)
  2. 调用 GetInt("key12")
  3. 等待 150ms
  4. 调用 GetInt("key12")
预期结果:
  - IncrWithTTL 返回 1, nil
  - 第一次 GetInt 返回 1, nil
  - 第二次 GetInt 返回 0, ErrKeyNotFound
```

#### TC-ST-013-LocalStorage_IncrWithTTL_Existing 🔴
```
描述: 测试带 TTL 的 Incr（已存在 key，TTL 不应改变）
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 IncrWithTTL("key13", 5s)
  2. 等待 100ms
  3. 调用 IncrWithTTL("key13", 100ms)
  4. 等待 150ms
  5. 调用 GetInt("key13")
预期结果:
  - 第一次 IncrWithTTL 返回 1
  - 第二次 IncrWithTTL 返回 2
  - GetInt 返回 2（TTL 保持原来的 5s，未过期）
```

#### TC-ST-014-LocalStorage_Incr_Concurrent 🔴
```
描述: 测试并发 Incr 的原子性
前置条件: LocalStorage 已初始化
测试步骤:
  1. 启动 100 个 goroutine
  2. 每个 goroutine 调用 Incr("key14") 100 次
  3. 等待所有 goroutine 完成
  4. 调用 GetInt("key14")
预期结果:
  - GetInt 返回 10000, nil
```

#### TC-ST-015-LocalStorage_GetInt_NotFound 🔴
```
描述: 测试 GetInt 获取不存在的 key
前置条件: LocalStorage 已初始化，key15 不存在
测试步骤:
  1. 调用 GetInt("key15")
预期结果:
  - 返回 0, ErrKeyNotFound
```

### 2.3 List 操作测试

#### TC-ST-016-LocalStorage_LPush_NewKey 🔴
```
描述: 测试对新 key 的 LPush
前置条件: LocalStorage 已初始化，key16 不存在
测试步骤:
  1. 调用 LPush("key16", 100, 200, 300)
  2. 调用 LLen("key16")
预期结果:
  - LPush 返回 nil
  - LLen 返回 3, nil
```

#### TC-ST-017-LocalStorage_LPush_Existing 🔴
```
描述: 测试对已存在 key 的 LPush（头部插入）
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 LPush("key17", 100)
  2. 调用 LPush("key17", 200)
  3. 调用 LIndex("key17", 0)
  4. 调用 LIndex("key17", 1)
预期结果:
  - LIndex(0) 返回 200, nil（最新的在头部）
  - LIndex(1) 返回 100, nil
```

#### TC-ST-018-LocalStorage_LLen_Empty 🔴
```
描述: 测试空列表的 LLen
前置条件: LocalStorage 已初始化，key18 不存在
测试步骤:
  1. 调用 LLen("key18")
预期结果:
  - 返回 0, nil
```

#### TC-ST-019-LocalStorage_LIndex_Success 🔴
```
描述: 测试 LIndex 正常获取
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 LPush("key19", 100, 200, 300)（列表变为 [300, 200, 100]）
  2. 调用 LIndex("key19", 0)
  3. 调用 LIndex("key19", 1)
  4. 调用 LIndex("key19", 2)
预期结果:
  - LIndex(0) 返回 300
  - LIndex(1) 返回 200
  - LIndex(2) 返回 100
```

#### TC-ST-020-LocalStorage_LIndex_OutOfRange 🔴
```
描述: 测试 LIndex 越界
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 LPush("key20", 100)
  2. 调用 LIndex("key20", 5)
预期结果:
  - 返回 0, ErrIndexOutOfRange
```

#### TC-ST-021-LocalStorage_LTrim_Success 🔴
```
描述: 测试 LTrim 裁剪列表
前置条件: LocalStorage 已初始化
测试步骤:
  1. 调用 LPush("key21", 100, 200, 300, 400, 500)
  2. 调用 LTrim("key21", 0, 2)
  3. 调用 LLen("key21")
预期结果:
  - LLen 返回 3
```

#### TC-ST-022-LocalStorage_List_Concurrent 🟡
```
描述: 测试并发 List 操作
前置条件: LocalStorage 已初始化
测试步骤:
  1. 启动 10 个 goroutine
  2. 每个 goroutine 调用 LPush("key22", timestamp) 10 次
  3. 等待所有 goroutine 完成
  4. 调用 LLen("key22")
预期结果:
  - LLen 返回 100
```

### 2.4 内存管理测试

#### TC-ST-023-LocalStorage_MaxSize_Eviction 🟡
```
描述: 测试达到最大内存时的淘汰机制
前置条件: LocalStorage 初始化，max_size = 1MB
测试步骤:
  1. 循环写入大量数据直到超过 1MB
  2. 检查旧数据是否被淘汰
预期结果:
  - 总内存不超过配置的 max_size
  - 旧数据被 LRU 淘汰
```

#### TC-ST-024-LocalStorage_Cleanup_Expired 🟡
```
描述: 测试后台清理过期数据
前置条件: LocalStorage 初始化，cleanup_interval = 100ms
测试步骤:
  1. 写入 100 个 TTL=50ms 的 key
  2. 等待 200ms
  3. 检查内存占用
预期结果:
  - 过期数据被后台清理
```

---

## 3. Redis 存储测试 (RedisStorage)

### 3.1 连接测试

#### TC-ST-025-RedisStorage_Connect_Success 🔴
```
描述: 测试 Redis 连接成功
前置条件: Redis 服务可用
测试步骤:
  1. 使用正确配置创建 RedisStorage
  2. 调用 Ping()
预期结果:
  - Ping 返回 nil
```

#### TC-ST-026-RedisStorage_Connect_Fail 🔴
```
描述: 测试 Redis 连接失败
前置条件: Redis 服务不可用或配置错误
测试步骤:
  1. 使用错误地址创建 RedisStorage
  2. 调用 Ping()
预期结果:
  - Ping 返回连接错误
```

### 3.2 String 操作测试

#### TC-ST-027-RedisStorage_Set_Get_Success 🔴
```
描述: 测试 Redis Set 和 Get
前置条件: RedisStorage 已连接
测试步骤:
  1. 调用 Set("rkey1", "rvalue1", 60s)
  2. 调用 Get("rkey1")
预期结果:
  - Get 返回 "rvalue1", nil
```

#### TC-ST-028-RedisStorage_TTL_Expire 🔴
```
描述: 测试 Redis TTL 过期
前置条件: RedisStorage 已连接
测试步骤:
  1. 调用 Set("rkey2", "rvalue2", 1s)
  2. 等待 1.5s
  3. 调用 Get("rkey2")
预期结果:
  - Get 返回 "", ErrKeyNotFound
```

### 3.3 计数器操作测试

#### TC-ST-029-RedisStorage_Incr_Atomic 🔴
```
描述: 测试 Redis Incr 原子性
前置条件: RedisStorage 已连接
测试步骤:
  1. 启动 100 个 goroutine
  2. 每个 goroutine 调用 Incr("rkey3") 100 次
  3. 调用 GetInt("rkey3")
预期结果:
  - GetInt 返回 10000
```

### 3.4 List 操作测试

#### TC-ST-030-RedisStorage_LPush_LIndex 🔴
```
描述: 测试 Redis List 操作
前置条件: RedisStorage 已连接
测试步骤:
  1. 调用 LPush("rkey4", 100, 200, 300)
  2. 调用 LIndex("rkey4", 0)
  3. 调用 LLen("rkey4")
预期结果:
  - LIndex(0) 返回 300
  - LLen 返回 3
```

---

## 4. 存储管理器测试 (StorageManager)

### 4.1 初始化测试

#### TC-ST-031-StorageManager_Init_Redis 🔴
```
描述: 测试使用 Redis 作为主存储初始化
前置条件: Redis 可用
测试步骤:
  1. 配置 storage.type = "redis"
  2. 创建 StorageManager
预期结果:
  - 初始化成功
  - 使用 Redis 作为当前存储
```

#### TC-ST-032-StorageManager_Init_Local 🔴
```
描述: 测试使用本地存储初始化
前置条件: 无
测试步骤:
  1. 配置 storage.type = "local"
  2. 创建 StorageManager
预期结果:
  - 初始化成功
  - 使用 LocalStorage 作为当前存储
```

#### TC-ST-033-StorageManager_Init_Redis_Unavailable 🔴
```
描述: 测试 Redis 不可用时的初始化
前置条件: Redis 不可用，fallback.enabled = true
测试步骤:
  1. 配置 storage.type = "redis"
  2. 创建 StorageManager
预期结果:
  - 初始化成功
  - 自动降级到 LocalStorage
  - 日志记录警告
```

### 4.2 降级测试

#### TC-ST-034-StorageManager_Failover_ToLocal 🔴
```
描述: 测试 Redis 故障时自动切换到本地
前置条件: StorageManager 使用 Redis，fallback.enabled = true
测试步骤:
  1. 正常使用 Redis 进行几次操作
  2. 模拟 Redis 连续 3 次失败
  3. 检查当前使用的存储
  4. 继续进行操作
预期结果:
  - 自动切换到 LocalStorage
  - 后续操作正常
```

#### TC-ST-035-StorageManager_Recovery_ToRedis 🔴
```
描述: 测试 Redis 恢复后自动切回
前置条件: StorageManager 已降级到本地存储
测试步骤:
  1. 等待 recovery_interval
  2. 模拟 Redis 恢复可用
  3. 检查当前使用的存储
预期结果:
  - 自动切回 Redis
```

#### TC-ST-036-StorageManager_Fallback_Disabled 🟡
```
描述: 测试禁用降级时的行为
前置条件: fallback.enabled = false
测试步骤:
  1. 模拟 Redis 不可用
  2. 尝试进行存储操作
预期结果:
  - 操作返回错误
  - 不会切换到本地存储
```

### 4.3 健康检查测试

#### TC-ST-037-StorageManager_HealthCheck_Success 🟡
```
描述: 测试健康检查通过
前置条件: StorageManager 正常运行
测试步骤:
  1. 调用 Ping()
预期结果:
  - 返回 nil
```

#### TC-ST-038-StorageManager_HealthCheck_Degraded 🟡
```
描述: 测试降级状态的健康检查
前置条件: StorageManager 已降级到本地
测试步骤:
  1. 调用 Ping()
  2. 调用 Status()
预期结果:
  - Ping 返回 nil
  - Status 显示 using_fallback = true
```

---

## 5. 边界条件测试

#### TC-ST-039-Storage_EmptyKey 🟡
```
描述: 测试空 key
测试步骤:
  1. 调用 Set("", "value", 60s)
预期结果:
  - 返回参数错误
```

#### TC-ST-040-Storage_EmptyValue 🟡
```
描述: 测试空 value
测试步骤:
  1. 调用 Set("key", "", 60s)
  2. 调用 Get("key")
预期结果:
  - Set 成功
  - Get 返回 ""
```

#### TC-ST-041-Storage_LargeValue 🟡
```
描述: 测试大 value（1MB）
测试步骤:
  1. 生成 1MB 的字符串
  2. 调用 Set("key", largeValue, 60s)
  3. 调用 Get("key")
预期结果:
  - Set 成功
  - Get 返回相同的值
```

#### TC-ST-042-Storage_SpecialCharacters 🟡
```
描述: 测试特殊字符
测试步骤:
  1. 调用 Set("key:with:colons", "value|with|pipes", 60s)
  2. 调用 Get("key:with:colons")
预期结果:
  - Get 返回 "value|with|pipes"
```

#### TC-ST-043-Storage_Unicode 🟡
```
描述: 测试 Unicode 字符
测试步骤:
  1. 调用 Set("中文key", "中文value", 60s)
  2. 调用 Get("中文key")
预期结果:
  - Get 返回 "中文value"
```

#### TC-ST-044-Storage_ZeroTTL 🟡
```
描述: 测试 TTL=0（永不过期）
测试步骤:
  1. 调用 Set("key", "value", 0)
  2. 等待一段时间
  3. 调用 Get("key")
预期结果:
  - Get 返回 "value"
```

#### TC-ST-045-Storage_NegativeIndex 🟡
```
描述: 测试负数索引
测试步骤:
  1. 调用 LPush("key", 100, 200, 300)
  2. 调用 LIndex("key", -1)
预期结果:
  - 返回错误或最后一个元素（根据实现）
```
