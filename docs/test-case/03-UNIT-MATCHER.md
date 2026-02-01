# 03 - 匹配器单元测试

## 1. 测试范围

测试 `internal/engine/matcher.go` 中的所有匹配器：
- ExactMatcher (精确匹配)
- AnyMatcher (任意值匹配)
- NotMatcher (取反匹配)
- MultiMatcher (多值匹配)
- RangeMatcher (范围匹配)
- GreaterMatcher (大于匹配)
- LessMatcher (小于匹配)
- IPWildcardMatcher (IP 通配匹配)
- IPRangeMatcher (IP 范围匹配)
- DictMatcher (字典匹配)

---

## 2. ExactMatcher 测试

#### TC-MT-001-Exact_Match_Success 🔴
```
描述: 精确匹配成功
测试步骤:
  1. 创建 ExactMatcher("post")
  2. 调用 Match("post")
预期结果:
  - 返回 true
```

#### TC-MT-002-Exact_Match_Fail 🔴
```
描述: 精确匹配失败
测试步骤:
  1. 创建 ExactMatcher("post")
  2. 调用 Match("comment")
预期结果:
  - 返回 false
```

#### TC-MT-003-Exact_Match_CaseSensitive 🔴
```
描述: 精确匹配区分大小写
测试步骤:
  1. 创建 ExactMatcher("Post")
  2. 调用 Match("post")
预期结果:
  - 返回 false
```

#### TC-MT-004-Exact_Match_Empty 🟡
```
描述: 精确匹配空字符串
测试步骤:
  1. 创建 ExactMatcher("")
  2. 调用 Match("")
预期结果:
  - 返回 true
```

---

## 3. AnyMatcher 测试

#### TC-MT-005-Any_Match_NonEmpty 🔴
```
描述: 任意值匹配非空字符串
测试步骤:
  1. 创建 AnyMatcher
  2. 调用 Match("anything")
预期结果:
  - 返回 true
```

#### TC-MT-006-Any_Match_Empty 🔴
```
描述: 任意值匹配空字符串
测试步骤:
  1. 创建 AnyMatcher
  2. 调用 Match("")
预期结果:
  - 返回 false
```

#### TC-MT-007-Any_Match_Whitespace 🟡
```
描述: 任意值匹配空白字符串
测试步骤:
  1. 创建 AnyMatcher
  2. 调用 Match("   ")
预期结果:
  - 返回 true（非空）
```

---

## 4. NotMatcher 测试

#### TC-MT-008-Not_Match_Inverse 🔴
```
描述: 取反匹配反转结果
测试步骤:
  1. 创建 NotMatcher(ExactMatcher("post"))
  2. 调用 Match("post")
  3. 调用 Match("comment")
预期结果:
  - Match("post") 返回 false
  - Match("comment") 返回 true
```

#### TC-MT-009-Not_Match_DoubleNot 🟡
```
描述: 双重取反
测试步骤:
  1. 创建 NotMatcher(NotMatcher(ExactMatcher("post")))
  2. 调用 Match("post")
预期结果:
  - 返回 true
```

---

## 5. MultiMatcher 测试

#### TC-MT-010-Multi_Match_First 🔴
```
描述: 多值匹配第一个值
测试步骤:
  1. 创建 MultiMatcher(["post", "comment", "reply"])
  2. 调用 Match("post")
预期结果:
  - 返回 true
```

#### TC-MT-011-Multi_Match_Middle 🔴
```
描述: 多值匹配中间值
测试步骤:
  1. 创建 MultiMatcher(["post", "comment", "reply"])
  2. 调用 Match("comment")
预期结果:
  - 返回 true
```

#### TC-MT-012-Multi_Match_Last 🔴
```
描述: 多值匹配最后一个值
测试步骤:
  1. 创建 MultiMatcher(["post", "comment", "reply"])
  2. 调用 Match("reply")
预期结果:
  - 返回 true
```

#### TC-MT-013-Multi_Match_None 🔴
```
描述: 多值匹配不在列表中
测试步骤:
  1. 创建 MultiMatcher(["post", "comment", "reply"])
  2. 调用 Match("delete")
预期结果:
  - 返回 false
```

#### TC-MT-014-Multi_Match_CaseSensitive 🔴
```
描述: 多值匹配区分大小写
测试步骤:
  1. 创建 MultiMatcher(["Post", "Comment"])
  2. 调用 Match("post")
预期结果:
  - 返回 false
```

---

## 6. RangeMatcher 测试

#### TC-MT-015-Range_Match_InRange 🔴
```
描述: 范围匹配在范围内
测试步骤:
  1. 创建 RangeMatcher(1, 100)
  2. 调用 Match("50")
预期结果:
  - 返回 true
```

#### TC-MT-016-Range_Match_AtMin 🔴
```
描述: 范围匹配等于最小值
测试步骤:
  1. 创建 RangeMatcher(1, 100)
  2. 调用 Match("1")
预期结果:
  - 返回 true
```

#### TC-MT-017-Range_Match_AtMax 🔴
```
描述: 范围匹配等于最大值
测试步骤:
  1. 创建 RangeMatcher(1, 100)
  2. 调用 Match("100")
预期结果:
  - 返回 true
```

#### TC-MT-018-Range_Match_BelowMin 🔴
```
描述: 范围匹配低于最小值
测试步骤:
  1. 创建 RangeMatcher(1, 100)
  2. 调用 Match("0")
预期结果:
  - 返回 false
```

#### TC-MT-019-Range_Match_AboveMax 🔴
```
描述: 范围匹配高于最大值
测试步骤:
  1. 创建 RangeMatcher(1, 100)
  2. 调用 Match("101")
预期结果:
  - 返回 false
```

#### TC-MT-020-Range_Match_NonNumeric 🔴
```
描述: 范围匹配非数字
测试步骤:
  1. 创建 RangeMatcher(1, 100)
  2. 调用 Match("abc")
预期结果:
  - 返回 false
```

#### TC-MT-021-Range_Match_Negative 🟡
```
描述: 范围匹配负数
测试步骤:
  1. 创建 RangeMatcher(-100, -1)
  2. 调用 Match("-50")
预期结果:
  - 返回 true
```

---

## 7. GreaterMatcher 测试

#### TC-MT-022-Greater_Match_Above 🔴
```
描述: 大于匹配成功
测试步骤:
  1. 创建 GreaterMatcher(100)
  2. 调用 Match("150")
预期结果:
  - 返回 true
```

#### TC-MT-023-Greater_Match_Equal 🔴
```
描述: 大于匹配等于（不匹配）
测试步骤:
  1. 创建 GreaterMatcher(100)
  2. 调用 Match("100")
预期结果:
  - 返回 false（>100，不是>=100）
```

#### TC-MT-024-Greater_Match_Below 🔴
```
描述: 大于匹配小于
测试步骤:
  1. 创建 GreaterMatcher(100)
  2. 调用 Match("50")
预期结果:
  - 返回 false
```

---

## 8. LessMatcher 测试

#### TC-MT-025-Less_Match_Below 🔴
```
描述: 小于匹配成功
测试步骤:
  1. 创建 LessMatcher(100)
  2. 调用 Match("50")
预期结果:
  - 返回 true
```

#### TC-MT-026-Less_Match_Equal 🔴
```
描述: 小于匹配等于（不匹配）
测试步骤:
  1. 创建 LessMatcher(100)
  2. 调用 Match("100")
预期结果:
  - 返回 false（<100，不是<=100）
```

#### TC-MT-027-Less_Match_Above 🔴
```
描述: 小于匹配大于
测试步骤:
  1. 创建 LessMatcher(100)
  2. 调用 Match("150")
预期结果:
  - 返回 false
```

---

## 9. IPWildcardMatcher 测试

#### TC-MT-028-IPWildcard_Match_Exact 🔴
```
描述: IP 通配无通配符
测试步骤:
  1. 创建 IPWildcardMatcher("192.168.1.1")
  2. 调用 Match("192.168.1.1")
预期结果:
  - 返回 true
```

#### TC-MT-029-IPWildcard_Match_LastOctet 🔴
```
描述: IP 通配最后一段
测试步骤:
  1. 创建 IPWildcardMatcher("192.168.1.*")
  2. 调用 Match("192.168.1.100")
  3. 调用 Match("192.168.1.1")
  4. 调用 Match("192.168.2.1")
预期结果:
  - Match("192.168.1.100") 返回 true
  - Match("192.168.1.1") 返回 true
  - Match("192.168.2.1") 返回 false
```

#### TC-MT-030-IPWildcard_Match_TwoOctets 🔴
```
描述: IP 通配两段
测试步骤:
  1. 创建 IPWildcardMatcher("192.168.*.*")
  2. 调用 Match("192.168.100.200")
  3. 调用 Match("192.169.1.1")
预期结果:
  - Match("192.168.100.200") 返回 true
  - Match("192.169.1.1") 返回 false
```

#### TC-MT-031-IPWildcard_Match_Invalid 🔴
```
描述: IP 通配无效 IP
测试步骤:
  1. 创建 IPWildcardMatcher("192.168.1.*")
  2. 调用 Match("invalid")
  3. 调用 Match("192.168.1")
预期结果:
  - 返回 false
```

---

## 10. IPRangeMatcher 测试

#### TC-MT-032-IPRange_Match_InRange 🔴
```
描述: IP 范围匹配在范围内
测试步骤:
  1. 创建 IPRangeMatcher("192.168.1.1", "192.168.1.255")
  2. 调用 Match("192.168.1.100")
预期结果:
  - 返回 true
```

#### TC-MT-033-IPRange_Match_AtStart 🔴
```
描述: IP 范围匹配等于起始
测试步骤:
  1. 创建 IPRangeMatcher("192.168.1.1", "192.168.1.255")
  2. 调用 Match("192.168.1.1")
预期结果:
  - 返回 true
```

#### TC-MT-034-IPRange_Match_AtEnd 🔴
```
描述: IP 范围匹配等于结束
测试步骤:
  1. 创建 IPRangeMatcher("192.168.1.1", "192.168.1.255")
  2. 调用 Match("192.168.1.255")
预期结果:
  - 返回 true
```

#### TC-MT-035-IPRange_Match_OutOfRange 🔴
```
描述: IP 范围匹配超出范围
测试步骤:
  1. 创建 IPRangeMatcher("192.168.1.1", "192.168.1.255")
  2. 调用 Match("192.168.2.1")
预期结果:
  - 返回 false
```

---

## 11. DictMatcher 测试

#### TC-MT-036-Dict_Match_InDict 🔴
```
描述: 字典匹配在字典中
测试步骤:
  1. 创建 DictMatcher({"123": {}, "456": {}, "789": {}})
  2. 调用 Match("456")
预期结果:
  - 返回 true
```

#### TC-MT-037-Dict_Match_NotInDict 🔴
```
描述: 字典匹配不在字典中
测试步骤:
  1. 创建 DictMatcher({"123": {}, "456": {}, "789": {}})
  2. 调用 Match("999")
预期结果:
  - 返回 false
```

#### TC-MT-038-Dict_Match_Empty 🟡
```
描述: 字典匹配空字典
测试步骤:
  1. 创建 DictMatcher({})
  2. 调用 Match("123")
预期结果:
  - 返回 false
```

---

## 12. ParseMatcher 测试

#### TC-MT-039-Parse_Plus 🔴
```
描述: 解析 + 为 AnyMatcher
测试步骤:
  1. 调用 ParseMatcher("+", dicts)
预期结果:
  - 返回 AnyMatcher 实例
```

#### TC-MT-040-Parse_Not 🔴
```
描述: 解析 ! 前缀为 NotMatcher
测试步骤:
  1. 调用 ParseMatcher("!post", dicts)
预期结果:
  - 返回 NotMatcher 包装 ExactMatcher("post")
```

#### TC-MT-041-Parse_AtDict 🔴
```
描述: 解析 @ 前缀为 DictMatcher
测试步骤:
  1. 准备 dicts = {"whitelist": {"123": {}, "456": {}}}
  2. 调用 ParseMatcher("@whitelist", dicts)
预期结果:
  - 返回 DictMatcher 实例
```

#### TC-MT-042-Parse_NotAtDict 🔴
```
描述: 解析 !@ 组合
测试步骤:
  1. 准备 dicts = {"blacklist": {"bad1": {}, "bad2": {}}}
  2. 调用 ParseMatcher("!@blacklist", dicts)
预期结果:
  - 返回 NotMatcher 包装 DictMatcher
```

#### TC-MT-043-Parse_Range 🔴
```
描述: 解析范围语法
测试步骤:
  1. 调用 ParseMatcher("1-100", dicts)
预期结果:
  - 返回 RangeMatcher(1, 100)
```

#### TC-MT-044-Parse_Greater 🔴
```
描述: 解析大于语法
测试步骤:
  1. 调用 ParseMatcher(">100", dicts)
预期结果:
  - 返回 GreaterMatcher(100)
```

#### TC-MT-045-Parse_Less 🔴
```
描述: 解析小于语法
测试步骤:
  1. 调用 ParseMatcher("<100", dicts)
预期结果:
  - 返回 LessMatcher(100)
```

#### TC-MT-046-Parse_IPWildcard 🔴
```
描述: 解析 IP 通配
测试步骤:
  1. 调用 ParseMatcher("192.168.*.*", dicts)
预期结果:
  - 返回 IPWildcardMatcher
```

#### TC-MT-047-Parse_Multi 🔴
```
描述: 解析多值
测试步骤:
  1. 调用 ParseMatcher("post,comment,reply", dicts)
预期结果:
  - 返回 MultiMatcher(["post", "comment", "reply"])
```

#### TC-MT-048-Parse_Exact 🔴
```
描述: 解析精确值
测试步骤:
  1. 调用 ParseMatcher("post", dicts)
预期结果:
  - 返回 ExactMatcher("post")
```

---

## 13. 错误处理测试

#### TC-MT-049-Parse_InvalidDict 🔴
```
描述: 解析引用不存在的字典
测试步骤:
  1. 调用 ParseMatcher("@nonexistent", dicts)
预期结果:
  - 返回错误 "dict not found: nonexistent"
```

#### TC-MT-050-Parse_InvalidRange 🟡
```
描述: 解析无效范围
测试步骤:
  1. 调用 ParseMatcher("100-abc", dicts)
预期结果:
  - 返回错误或回退到 ExactMatcher
```

#### TC-MT-051-Parse_InvalidGreater 🟡
```
描述: 解析无效大于
测试步骤:
  1. 调用 ParseMatcher(">abc", dicts)
预期结果:
  - 返回错误
```
