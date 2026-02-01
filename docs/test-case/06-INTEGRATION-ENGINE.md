# 06 - 引擎集成测试

## 1. 测试范围

测试 `internal/engine/` 规则引擎的集成功能：
- 规则执行顺序
- 多规则匹配
- 热重载
- 降级处理

---

## 2. 规则执行顺序测试

### 2.1 Phase 1: 访问控制优先

#### TC-EN-001-Order_Whitelist_First 🔴
```
描述: 白名单优先于所有业务规则
前置条件:
  - 白名单包含 uid "vip001"
  - 存在业务规则限制该 uid 的 post 行为
测试步骤:
  1. 请求 {"act": "post", "uid": "vip001"}
预期结果:
  - 返回 allowed=true
  - rule_name = "uid_whitelist"（白名单规则）
  - 业务规则未被检查
```

#### TC-EN-002-Order_Blacklist_First 🔴
```
描述: 黑名单优先于所有业务规则
前置条件:
  - 黑名单包含 uid "banned001"
  - 不存在其他会拦截的规则
测试步骤:
  1. 请求 {"act": "post", "uid": "banned001"}
预期结果:
  - 返回 allowed=false
  - rule_name = "uid_blacklist"
```

#### TC-EN-003-Order_Whitelist_Before_Blacklist 🔴
```
描述: 白名单在黑名单之前检查
前置条件:
  - uid "special001" 同时在白名单和黑名单中
测试步骤:
  1. 请求 {"act": "post", "uid": "special001"}
预期结果:
  - 返回 allowed=true（白名单优先）
```

### 2.2 Phase 2: 业务规则优先级

#### TC-EN-004-Order_Business_Before_Post 🔴
```
描述: Business 规则优先于 Post 规则
前置条件:
  - rules.business 包含规则 A
  - rules.post 包含规则 B
  - 两个规则都能匹配请求
测试步骤:
  1. 发送能匹配两个规则的请求
预期结果:
  - 命中规则 A（business 优先级更高）
```

#### TC-EN-005-Order_Post_Before_Advanced 🔴
```
描述: Post 规则优先于 Advanced 规则
前置条件:
  - rules.post 包含规则 A
  - rules.advanced 包含规则 B
  - 两个规则都能匹配请求
测试步骤:
  1. 发送能匹配两个规则的请求
预期结果:
  - 命中规则 A（post 优先级更高）
```

#### TC-EN-006-Order_FirstMatch_Wins 🔴
```
描述: 同优先级内第一个匹配的规则生效
前置条件:
  - rules.business 包含规则 A、B（A 在 B 之前）
  - 两个规则都能匹配请求
测试步骤:
  1. 发送能匹配两个规则的请求
预期结果:
  - 命中规则 A（配置顺序在前）
```

---

## 3. 多规则场景测试

#### TC-EN-007-MultiRule_DifferentKeys 🔴
```
描述: 不同参数匹配不同规则
前置条件:
  - 规则 A: act="post"
  - 规则 B: act="comment"
测试步骤:
  1. 请求 {"act": "post", "uid": "123"}
  2. 请求 {"act": "comment", "uid": "123"}
预期结果:
  - 第一个请求匹配规则 A
  - 第二个请求匹配规则 B
```

#### TC-EN-008-MultiRule_SameAct_DiffUid 🔴
```
描述: 相同行为不同用户
前置条件:
  - 规则: act="post", uid="+"，每日限 5 次
测试步骤:
  1. uid="123" 请求 5 次
  2. uid="456" 请求 1 次
预期结果:
  - uid="123" 第 6 次被拒绝
  - uid="456" 第 1 次被允许
```

#### TC-EN-009-MultiRule_Cascade 🔴
```
描述: 规则级联（通过前一规则后检查下一规则）
前置条件:
  - 规则 A: act="post", uid="+", limit=10/day
  - 规则 B: act="post", uid="+", limit=1/min
测试步骤:
  1. 1 分钟内请求 2 次
预期结果:
  - 第 2 次被规则 B 拦截（每分钟限 1 次）
```

---

## 4. 热重载测试

#### TC-EN-010-HotReload_AddRule 🔴
```
描述: 热重载添加新规则
前置条件: 规则引擎运行中
测试步骤:
  1. 确认规则 "new_rule" 不存在
  2. 修改 rules.toml 添加 "new_rule"
  3. 等待热重载完成
  4. 发送能匹配 "new_rule" 的请求
预期结果:
  - 新规则生效
  - 请求被新规则处理
```

#### TC-EN-011-HotReload_RemoveRule 🔴
```
描述: 热重载删除规则
前置条件: 规则引擎运行中，存在规则 "old_rule"
测试步骤:
  1. 确认规则 "old_rule" 存在
  2. 修改 rules.toml 删除 "old_rule"
  3. 等待热重载完成
  4. 发送原本会匹配 "old_rule" 的请求
预期结果:
  - 请求不再被 "old_rule" 处理
```

#### TC-EN-012-HotReload_ModifyLimit 🔴
```
描述: 热重载修改限制参数
前置条件: 存在规则 limit=5
测试步骤:
  1. 请求 3 次（计数=3）
  2. 修改规则 limit=2
  3. 等待热重载完成
  4. 再次请求
预期结果:
  - 第 4 次请求被拒绝（新限制 limit=2，当前计数=3）
```

#### TC-EN-013-HotReload_ModifyDict 🔴
```
描述: 热重载修改字典
前置条件: 白名单不包含 uid "new_vip"
测试步骤:
  1. 确认 "new_vip" 不在白名单
  2. 向白名单文件添加 "new_vip"
  3. 等待热重载完成
  4. 请求 {"uid": "new_vip"}
预期结果:
  - "new_vip" 被白名单放行
```

#### TC-EN-014-HotReload_InvalidConfig_KeepOld 🔴
```
描述: 热重载无效配置时保留旧配置
前置条件: 有效配置运行中
测试步骤:
  1. 将 rules.toml 修改为无效格式
  2. 等待热重载尝试
  3. 检查当前配置
预期结果:
  - 旧配置仍然生效
  - 日志记录加载失败
```

#### TC-EN-015-HotReload_Concurrent 🔴
```
描述: 热重载期间的并发请求
前置条件: 规则引擎运行中
测试步骤:
  1. 启动 100 个并发请求
  2. 同时触发热重载
  3. 检查所有请求结果
预期结果:
  - 所有请求正常处理
  - 无竞态条件错误
  - 热重载成功完成
```

---

## 5. 存储降级测试

#### TC-EN-016-Fallback_RedisDown 🔴
```
描述: Redis 故障时降级到本地存储
前置条件: 使用 Redis 作为存储
测试步骤:
  1. 正常请求几次
  2. 断开 Redis 连接
  3. 继续请求
预期结果:
  - 自动切换到本地存储
  - 请求继续正常处理
```

#### TC-EN-017-Fallback_RedisRecover 🔴
```
描述: Redis 恢复后切回
前置条件: 已降级到本地存储
测试步骤:
  1. 恢复 Redis 连接
  2. 等待恢复检测
  3. 检查当前存储
预期结果:
  - 自动切回 Redis
```

#### TC-EN-018-Fallback_CounterReset 🟡
```
描述: 降级时计数器状态
前置条件: Redis 中有计数器数据
测试步骤:
  1. 在 Redis 中设置计数器=5
  2. 触发降级到本地
  3. 检查 Browse 结果
预期结果:
  - 本地存储计数器从 0 开始（数据不同步）
  - 或根据配置决定行为
```

---

## 6. 边界条件测试

#### TC-EN-019-Edge_NoRules 🔴
```
描述: 无任何规则时的行为
前置条件: 规则列表为空
测试步骤:
  1. 发送任意请求
预期结果:
  - 返回 allowed=true（默认放行）
```

#### TC-EN-020-Edge_AllRulesSkipped 🔴
```
描述: 所有规则都不匹配
前置条件: 存在规则，但都不匹配请求参数
测试步骤:
  1. 发送不匹配任何规则的请求
预期结果:
  - 返回 allowed=true
```

#### TC-EN-021-Edge_EmptyParams 🔴
```
描述: 空参数请求
测试步骤:
  1. 发送 {"act": ""}
预期结果:
  - 返回错误或默认放行
```

#### TC-EN-022-Edge_VeryLongParams 🟡
```
描述: 超长参数
测试步骤:
  1. 发送 uid 为 10000 字符的请求
预期结果:
  - 正常处理或返回参数错误
```

---

## 7. 性能相关测试

#### TC-EN-023-Perf_ManyRules 🟡
```
描述: 大量规则时的性能
前置条件: 配置 1000 条规则
测试步骤:
  1. 发送请求
  2. 测量响应时间
预期结果:
  - 响应时间 < 10ms
```

#### TC-EN-024-Perf_LargeDict 🟡
```
描述: 大字典时的性能
前置条件: 白名单包含 100 万条目
测试步骤:
  1. 发送请求检查白名单
  2. 测量响应时间
预期结果:
  - 响应时间 < 5ms（字典查找是 O(1)）
```

#### TC-EN-025-Perf_HotReload_LargeConfig 🟡
```
描述: 大配置热重载性能
前置条件: 配置包含 1000 条规则
测试步骤:
  1. 触发热重载
  2. 测量重载时间
预期结果:
  - 重载时间 < 1s
  - 服务不中断
```
