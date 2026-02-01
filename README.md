# Koala V2

高性能频率控制与反作弊系统

## 快速开始

### 1. 拉取第三方依赖

```bash
make deps
```

### 2. 编译

```bash
make build
```

### 3. 运行

```bash
make run
```

### 4. 测试

```bash
# 健康检查
curl http://localhost:9981/health

# 检查请求
curl -X POST http://localhost:9981/api/v1/browse \
  -H "Content-Type: application/json" \
  -d '{"act": "test", "uid": "12345"}'
```

## 目录结构

```
koala-v2/
├── cmd/koala/          # 程序入口
├── internal/           # 内部包
│   ├── api/           # HTTP 层
│   ├── engine/        # 规则引擎
│   ├── storage/       # 存储层
│   └── config/        # 配置管理
├── third_party/       # 第三方依赖（本地化）
├── pkg/               # 公共包
├── conf/              # 配置文件
├── docs/              # 文档
├── scripts/           # 脚本
└── deployments/       # 部署配置
```

## 文档

详细文档请参阅 [docs/v2-tech-design/](docs/v2-tech-design/00-INDEX.md)

## 配置

- 服务配置: `conf/koala.toml`
- 规则配置: `conf/rules.toml`

## API

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/v1/browse | POST | 检查请求是否允许 |
| /api/v1/update | POST | 记录请求 |
| /api/v1/batch | POST | 批量检查 |
| /health | GET | 健康检查 |
| /metrics | GET | Prometheus 指标 |

## License

MIT
