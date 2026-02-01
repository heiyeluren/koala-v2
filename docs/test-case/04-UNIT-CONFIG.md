# 04 - 配置加载单元测试

## 1. 测试范围

测试 `internal/config/` 下的配置相关功能：
- 服务配置加载 (koala.toml)
- 规则配置加载 (rules.toml)
- 字典文件加载
- 配置验证
- 配置热重载

---

## 2. 服务配置加载测试

### 2.1 基本加载测试

#### TC-CF-001-LoadConfig_Success 🔴
```
描述: 成功加载有效配置文件
前置条件: 存在有效的 koala.toml
测试步骤:
  1. 调用 config.Load("testdata/valid_koala.toml")
预期结果:
  - 返回 *Config, nil
  - 所有字段正确解析
```

#### TC-CF-002-LoadConfig_FileNotFound 🔴
```
描述: 配置文件不存在
测试步骤:
  1. 调用 config.Load("nonexistent.toml")
预期结果:
  - 返回 nil, error
  - 错误包含 "no such file"
```

#### TC-CF-003-LoadConfig_InvalidToml 🔴
```
描述: 配置文件格式错误
前置条件: 存在格式错误的 toml 文件
测试步骤:
  1. 调用 config.Load("testdata/invalid.toml")
预期结果:
  - 返回 nil, error
  - 错误包含 TOML 解析错误信息
```

### 2.2 字段解析测试

#### TC-CF-004-LoadConfig_ServerSection 🔴
```
描述: 正确解析 [server] 配置
测试数据:
  [server]
  listen = ":9981"
  read_timeout = "5s"
  write_timeout = "5s"
测试步骤:
  1. 加载配置
  2. 检查 config.Server 字段
预期结果:
  - Listen = ":9981"
  - ReadTimeout = 5 * time.Second
  - WriteTimeout = 5 * time.Second
```

#### TC-CF-005-LoadConfig_StorageSection 🔴
```
描述: 正确解析 [storage] 配置
测试数据:
  [storage]
  type = "redis"
  [storage.redis]
  addr = "127.0.0.1:6379"
  pool_size = 100
测试步骤:
  1. 加载配置
  2. 检查 config.Storage 字段
预期结果:
  - Type = "redis"
  - Redis.Addr = "127.0.0.1:6379"
  - Redis.PoolSize = 100
```

#### TC-CF-006-LoadConfig_DurationParsing 🔴
```
描述: 正确解析时间格式
测试数据:
  read_timeout = "5s"
  write_timeout = "1m"
  shutdown_timeout = "30s"
测试步骤:
  1. 加载配置
预期结果:
  - 5s 解析为 5 * time.Second
  - 1m 解析为 1 * time.Minute
  - 30s 解析为 30 * time.Second
```

#### TC-CF-007-LoadConfig_SizeParsing 🟡
```
描述: 正确解析大小格式
测试数据:
  max_size = "1GB"
测试步骤:
  1. 加载配置
预期结果:
  - 1GB 解析为 1073741824 字节
```

### 2.3 默认值测试

#### TC-CF-008-LoadConfig_DefaultValues 🔴
```
描述: 缺失字段使用默认值
测试数据:
  [server]
  listen = ":9981"
  # 其他字段缺失
测试步骤:
  1. 加载配置
预期结果:
  - ReadTimeout = 默认值 (如 5s)
  - WriteTimeout = 默认值
```

---

## 3. 规则配置加载测试

### 3.1 基本加载测试

#### TC-CF-009-LoadRules_Success 🔴
```
描述: 成功加载有效规则文件
前置条件: 存在有效的 rules.toml
测试步骤:
  1. 调用 config.LoadRules("testdata/valid_rules.toml")
预期结果:
  - 返回 *Policy, nil
```

#### TC-CF-010-LoadRules_InvalidFormat 🔴
```
描述: 规则文件格式错误
测试步骤:
  1. 调用 config.LoadRules("testdata/invalid_rules.toml")
预期结果:
  - 返回 nil, error
```

### 3.2 规则解析测试

#### TC-CF-011-LoadRules_AccessWhitelist 🔴
```
描述: 正确解析访问白名单规则
测试数据:
  [[access.whitelist]]
  name = "vip_users"
  match = { uid = "@vip_list" }
  result = "allow"
测试步骤:
  1. 加载规则
  2. 检查 policy.Access.Whitelist
预期结果:
  - 包含名为 "vip_users" 的规则
  - match.uid = "@vip_list"
  - result = "allow"
```

#### TC-CF-012-LoadRules_AccessBlacklist 🔴
```
描述: 正确解析访问黑名单规则
测试数据:
  [[access.blacklist]]
  name = "banned_users"
  match = { uid = "@banned_list" }
  result = "deny"
测试步骤:
  1. 加载规则
  2. 检查 policy.Access.Blacklist
预期结果:
  - 包含名为 "banned_users" 的规则
```

#### TC-CF-013-LoadRules_BusinessRules 🔴
```
描述: 正确解析业务规则
测试数据:
  [[rules.business]]
  name = "daily_limit"
  type = "count"
  match = { act = "post", uid = "+" }
  limit = { time = "24h", count = 10 }
  result = "deny"
测试步骤:
  1. 加载规则
  2. 检查 policy.Rate.Business
预期结果:
  - 规则名称、类型、匹配条件、限制参数正确
```

#### TC-CF-014-LoadRules_AdvancedRules 🔴
```
描述: 正确解析高级规则（Base 类型）
测试数据:
  [[rules.advanced]]
  name = "base_control"
  type = "base"
  match = { act = "ask", ip = "+" }
  limit = { base = 10, time = "5s", count = 1 }
  result = "deny"
测试步骤:
  1. 加载规则
  2. 检查 policy.Rate.Advanced
预期结果:
  - limit.base = 10
  - limit.time = 5s
  - limit.count = 1
```

#### TC-CF-015-LoadRules_LeakRules 🔴
```
描述: 正确解析 Leak 类型规则
测试数据:
  [[rules.advanced]]
  name = "leak_control"
  type = "leak"
  match = { act = "api_call", uid = "+" }
  limit = { time = "60s", count = 100 }
  result = "deny"
测试步骤:
  1. 加载规则
预期结果:
  - type = "leak"
  - limit 正确解析
```

### 3.3 结果模板测试

#### TC-CF-016-LoadRules_Results 🔴
```
描述: 正确解析结果模板
测试数据:
  [results]
  allow = { code = 0, message = "Allow" }
  deny = { code = 10, message = "Deny" }
  auth_slider = { code = 20, message = "Auth", auth_type = 1 }
测试步骤:
  1. 加载规则
  2. 检查 policy.Results
预期结果:
  - Results["allow"].Code = 0
  - Results["deny"].Code = 10
  - Results["auth_slider"].AuthType = 1
```

#### TC-CF-017-LoadRules_ResultReference 🔴
```
描述: 规则正确引用结果模板
测试步骤:
  1. 加载包含规则引用结果的配置
  2. 调用 policy.GetResult(rule.Result)
预期结果:
  - 返回正确的 Result 对象
```

---

## 4. 字典文件加载测试

#### TC-CF-018-LoadDict_Success 🔴
```
描述: 成功加载字典文件
前置条件: 存在有效的字典文件
测试数据 (dict.txt):
  123
  456
  789
测试步骤:
  1. 调用 LoadDict("testdata/dict.txt")
预期结果:
  - 返回 map{"123": {}, "456": {}, "789": {}}
```

#### TC-CF-019-LoadDict_WithComments 🔴
```
描述: 加载带注释的字典文件
测试数据:
  # This is a comment
  123
  # Another comment
  456
测试步骤:
  1. 调用 LoadDict("testdata/dict_with_comments.txt")
预期结果:
  - 只包含 "123" 和 "456"
  - 注释被忽略
```

#### TC-CF-020-LoadDict_WithEmptyLines 🔴
```
描述: 加载带空行的字典文件
测试数据:
  123

  456

  789
测试步骤:
  1. 调用 LoadDict
预期结果:
  - 空行被忽略
  - 只包含 "123", "456", "789"
```

#### TC-CF-021-LoadDict_FileNotFound 🔴
```
描述: 字典文件不存在
测试步骤:
  1. 调用 LoadDict("nonexistent.txt")
预期结果:
  - 返回错误
```

#### TC-CF-022-LoadDict_Empty 🟡
```
描述: 加载空字典文件
测试步骤:
  1. 调用 LoadDict("testdata/empty.txt")
预期结果:
  - 返回空 map
  - 无错误
```

---

## 5. 配置验证测试

#### TC-CF-023-Validate_ValidConfig 🔴
```
描述: 验证有效配置
测试步骤:
  1. 创建完整有效的配置
  2. 调用 config.Validate()
预期结果:
  - 返回 nil
```

#### TC-CF-024-Validate_MissingListen 🔴
```
描述: 验证缺少监听地址
测试步骤:
  1. 创建配置，listen = ""
  2. 调用 config.Validate()
预期结果:
  - 返回错误 "listen address is required"
```

#### TC-CF-025-Validate_InvalidStorageType 🔴
```
描述: 验证无效存储类型
测试步骤:
  1. 创建配置，storage.type = "invalid"
  2. 调用 config.Validate()
预期结果:
  - 返回错误 "invalid storage type"
```

#### TC-CF-026-Validate_DuplicateRuleName 🔴
```
描述: 验证重复规则名称
测试步骤:
  1. 创建包含两个同名规则的配置
  2. 调用 config.Validate()
预期结果:
  - 返回错误 "duplicate rule name"
```

#### TC-CF-027-Validate_InvalidAlgorithmType 🔴
```
描述: 验证无效算法类型
测试步骤:
  1. 创建规则，type = "unknown"
  2. 调用 config.Validate()
预期结果:
  - 返回错误 "invalid algorithm type"
```

#### TC-CF-028-Validate_MissingResultTemplate 🔴
```
描述: 验证规则引用不存在的结果模板
测试步骤:
  1. 创建规则，result = "nonexistent"
  2. 调用 config.Validate()
预期结果:
  - 返回错误 "result template not found"
```

---

## 6. 配置热重载测试

#### TC-CF-029-Watcher_DetectChange 🔴
```
描述: 检测文件变化
前置条件: Watcher 已启动监听
测试步骤:
  1. 修改 rules.toml 文件
  2. 等待事件触发
预期结果:
  - 触发回调函数
```

#### TC-CF-030-Watcher_ReloadOnChange 🔴
```
描述: 文件变化时重新加载
前置条件: Watcher 已启动，旧规则已加载
测试步骤:
  1. 修改 rules.toml 添加新规则
  2. 等待重载完成
  3. 检查新规则是否生效
预期结果:
  - 新规则被加载
```

#### TC-CF-031-Watcher_InvalidConfigKeepOld 🔴
```
描述: 无效配置时保留旧配置
前置条件: 有效配置已加载
测试步骤:
  1. 将 rules.toml 修改为无效格式
  2. 等待重载尝试
  3. 检查当前配置
预期结果:
  - 旧配置仍然生效
  - 日志记录错误
```

#### TC-CF-032-Watcher_Debounce 🟡
```
描述: 防抖处理多次快速修改
测试步骤:
  1. 快速修改文件 10 次
  2. 统计回调触发次数
预期结果:
  - 回调只触发 1 次（防抖）
```

#### TC-CF-033-Watcher_DictChange 🔴
```
描述: 字典文件变化时重载
前置条件: 规则引用字典文件
测试步骤:
  1. 修改字典文件添加新条目
  2. 等待重载完成
  3. 测试新条目是否生效
预期结果:
  - 字典更新生效
```

---

## 7. 边界条件测试

#### TC-CF-034-Config_LargeFile 🟡
```
描述: 加载大配置文件
测试步骤:
  1. 创建包含 10000 条规则的配置文件
  2. 加载配置
预期结果:
  - 加载成功
  - 性能可接受（< 5s）
```

#### TC-CF-035-Config_Unicode 🟡
```
描述: 配置包含 Unicode 字符
测试数据:
  desc = "中文描述"
测试步骤:
  1. 加载配置
预期结果:
  - Unicode 字符正确解析
```

#### TC-CF-036-Config_SpecialChars 🟡
```
描述: 配置包含特殊字符
测试数据:
  message = "Error: \"something\" went wrong"
测试步骤:
  1. 加载配置
预期结果:
  - 转义字符正确处理
```
