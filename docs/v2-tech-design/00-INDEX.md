# Koala V2 技术设计文档索引

## 文档导航

| 序号 | 文档 | 说明 | 适合阅读场景 |
|------|------|------|-------------|
| 01 | [PRODUCT-OVERVIEW](./01-PRODUCT-OVERVIEW.md) | 产品概述 | 了解系统定位、核心价值、技术栈 |
| 02 | [ARCHITECTURE](./02-ARCHITECTURE.md) | 架构设计 | 了解模块划分、数据流、目录结构 |
| 03 | [CORE-ALGORITHMS](./03-CORE-ALGORITHMS.md) | 核心算法 | 了解四种限流算法原理与实现 |
| 04 | [API-REFERENCE](./04-API-REFERENCE.md) | API 参考 | 接口开发、调用接口 |
| 05 | [CONFIG-REFERENCE](./05-CONFIG-REFERENCE.md) | 配置参考 | 配置服务、编写规则 |
| 06 | [IMPLEMENTATION-GUIDE](./06-IMPLEMENTATION-GUIDE.md) | 实现指南 | 开发实现、代码编写 |

## 快速定位

### 我想了解...

| 问题 | 阅读文档 | 章节 |
|------|---------|------|
| 系统是做什么的？ | 01-PRODUCT-OVERVIEW | 1.1 系统定位 |
| 用了什么技术栈？ | 01-PRODUCT-OVERVIEW | 1.3 技术选型 |
| 整体架构是什么样？ | 02-ARCHITECTURE | 2.1 系统架构图 |
| 项目目录怎么组织？ | 02-ARCHITECTURE | 2.4 目录结构 |
| 第三方依赖怎么管理？ | 02-ARCHITECTURE | 2.5 依赖管理 |
| 四种算法怎么工作？ | 03-CORE-ALGORITHMS | 3.2-3.5 各算法详解 |
| 规则匹配语法？ | 03-CORE-ALGORITHMS | 3.6 匹配模式 |
| 有哪些 API？ | 04-API-REFERENCE | 4.1 接口总览 |
| 配置文件怎么写？ | 05-CONFIG-REFERENCE | 5.2-5.3 配置详解 |
| 如何开始开发？ | 06-IMPLEMENTATION-GUIDE | 6.1 开发环境 |
| 实现顺序是什么？ | 06-IMPLEMENTATION-GUIDE | 6.3 实现顺序 |

## 配置文件示例

- [koala.toml](./conf/koala.toml) - 服务配置示例
- [rules.toml](./conf/rules.toml) - 规则配置示例

## 文档版本

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2024-01-xx | 初始版本 |

## 项目信息

- **项目名称**: Koala V2
- **项目路径**: `/Users/black/Documents/Projects/gil/be/gil_koala/koala-v2`
- **Go 版本**: 1.21+
- **主要用途**: 高性能频率控制与反作弊系统
