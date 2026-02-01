FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git make

# 复制 go.mod 和依赖
COPY go.mod go.sum ./
COPY third_party/ third_party/
RUN go mod download

# 复制源码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o koala ./cmd/koala

# 运行镜像
FROM alpine:3.19

WORKDIR /app

# 安装时区数据
RUN apk add --no-cache tzdata

# 复制二进制和配置
COPY --from=builder /app/koala .
COPY conf/ conf/

# 创建日志目录
RUN mkdir -p logs

# 暴露端口
EXPOSE 9981

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9981/health || exit 1

# 启动
CMD ["./koala", "-config", "conf/koala.toml"]
