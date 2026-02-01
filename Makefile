.PHONY: build test clean run deps docker

# 变量
BINARY=koala
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# 默认目标
all: build

# 构建
build:
	@echo "Building ${BINARY}..."
	@mkdir -p bin
	go build ${LDFLAGS} -o bin/${BINARY} ./cmd/koala

# 测试
test:
	go test -v -race ./...

# 覆盖率测试
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 性能测试
bench:
	go test -bench=. -benchmem ./test/benchmark/

# 清理
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# 运行
run: build
	./bin/${BINARY} -config conf/koala.toml

# 运行（开发模式）
dev:
	go run ./cmd/koala -config conf/koala.toml

# 拉取第三方依赖
deps:
	@echo "Fetching third-party dependencies..."
	@bash scripts/fetch_deps.sh

# 代码检查
lint:
	golangci-lint run ./...

# 格式化
fmt:
	go fmt ./...
	goimports -w .

# Docker 构建
docker:
	docker build -t koala:${VERSION} -f deployments/Dockerfile .

# Docker Compose 启动
docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

# Docker Compose 停止
docker-down:
	docker-compose -f deployments/docker-compose.yml down

# 帮助
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  bench         - Run benchmarks"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Build and run"
	@echo "  dev           - Run in development mode"
	@echo "  deps          - Fetch third-party dependencies"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  docker        - Build Docker image"
	@echo "  docker-up     - Start with Docker Compose"
	@echo "  docker-down   - Stop Docker Compose"
