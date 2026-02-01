# Koala V2 测试用例索引

## 测试用例总览

| 文档 | 类型 | 用例数 | 说明 |
|------|------|--------|------|
| [01-UNIT-STORAGE](./01-UNIT-STORAGE.md) | 单元测试 | 45 | 存储层测试 |
| [02-UNIT-ALGORITHM](./02-UNIT-ALGORITHM.md) | 单元测试 | 52 | 四种算法测试 |
| [03-UNIT-MATCHER](./03-UNIT-MATCHER.md) | 单元测试 | 38 | 匹配器测试 |
| [04-UNIT-CONFIG](./04-UNIT-CONFIG.md) | 单元测试 | 28 | 配置加载测试 |
| [05-INTEGRATION-API](./05-INTEGRATION-API.md) | 集成测试 | 35 | API 接口测试 |
| [06-INTEGRATION-ENGINE](./06-INTEGRATION-ENGINE.md) | 集成测试 | 25 | 引擎集成测试 |
| [07-SCENARIO-BUSINESS](./07-SCENARIO-BUSINESS.md) | 场景测试 | 30 | 业务场景测试 |
| [08-PERFORMANCE](./08-PERFORMANCE.md) | 性能测试 | 15 | 性能基准测试 |

**总计: 268 个测试用例**

---

## 测试用例命名规范

```
TC-{模块}-{序号}-{简述}

模块代码:
  ST = Storage (存储)
  AL = Algorithm (算法)
  MT = Matcher (匹配器)
  CF = Config (配置)
  AP = API (接口)
  EN = Engine (引擎)
  BZ = Business (业务场景)
  PF = Performance (性能)

示例:
  TC-ST-001-LocalStorage_Set_Get_Success
  TC-AL-015-LeakAlgorithm_Browse_Hit
  TC-BZ-003-AIQuestion_DailyLimit_NonVIP
```

---

## 测试优先级

| 优先级 | 说明 | 标记 |
|--------|------|------|
| P0 | 核心功能，必须通过 | 🔴 |
| P1 | 重要功能，应该通过 | 🟡 |
| P2 | 次要功能，建议通过 | 🟢 |

---

## 测试覆盖目标

| 模块 | 行覆盖率目标 | 分支覆盖率目标 |
|------|-------------|---------------|
| storage | ≥ 90% | ≥ 85% |
| algorithm | ≥ 95% | ≥ 90% |
| matcher | ≥ 95% | ≥ 90% |
| config | ≥ 85% | ≥ 80% |
| api | ≥ 80% | ≥ 75% |
| engine | ≥ 85% | ≥ 80% |

---

## 测试环境要求

### 单元测试
- Go 1.21+
- 无外部依赖（使用 mock）

### 集成测试
- Go 1.21+
- Redis 6.0+（可选，有本地存储降级）

### 性能测试
- Go 1.21+
- 8 核 CPU / 16GB 内存
- Redis 6.0+

---

## 运行测试命令

```bash
# 运行所有单元测试
make test

# 运行指定模块测试
go test -v ./internal/storage/...
go test -v ./internal/engine/algorithm/...

# 运行集成测试
go test -v -tags=integration ./test/integration/...

# 运行性能测试
go test -bench=. -benchmem ./test/benchmark/...

# 生成覆盖率报告
make test-coverage
```
