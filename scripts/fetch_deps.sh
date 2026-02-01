#!/bin/bash
# 拉取第三方依赖到本地
# 使用方法: bash scripts/fetch_deps.sh

set -e

THIRD_PARTY_DIR="third_party"

echo "=========================================="
echo "Koala V2 - 拉取第三方依赖"
echo "=========================================="

mkdir -p $THIRD_PARTY_DIR
cd $THIRD_PARTY_DIR

# 函数：克隆仓库
clone_repo() {
    local name=$1
    local repo=$2
    local branch=$3
    local dir=$4

    if [ -d "$dir" ]; then
        echo "[$name] 已存在，跳过..."
        return
    fi

    echo "[$name] 正在拉取..."
    if [ -n "$branch" ]; then
        git clone --depth 1 --branch "$branch" "$repo" "$dir"
    else
        git clone --depth 1 "$repo" "$dir"
    fi
    rm -rf "$dir/.git"
    echo "[$name] 完成"
}

echo ""
echo "=== 核心依赖 ==="

clone_repo "ristretto" \
    "https://github.com/dgraph-io/ristretto.git" \
    "v2.4.0" \
    "ristretto"

clone_repo "zap" \
    "https://github.com/uber-go/zap.git" \
    "v1.27.1" \
    "zap"

clone_repo "go-redis" \
    "https://github.com/redis/go-redis.git" \
    "v9.7.3" \
    "redis"

clone_repo "gin" \
    "https://github.com/gin-gonic/gin.git" \
    "v1.10.0" \
    "gin"

clone_repo "fsnotify" \
    "https://github.com/fsnotify/fsnotify.git" \
    "v1.7.0" \
    "fsnotify"

clone_repo "toml" \
    "https://github.com/BurntSushi/toml.git" \
    "v1.3.2" \
    "toml"

clone_repo "prometheus-client" \
    "https://github.com/prometheus/client_golang.git" \
    "v1.17.0" \
    "prometheus_client"

echo ""
echo "=== 一级依赖 ==="

clone_repo "multierr" \
    "https://github.com/uber-go/multierr.git" \
    "v1.11.0" \
    "multierr"

clone_repo "atomic" \
    "https://github.com/uber-go/atomic.git" \
    "v1.11.0" \
    "atomic"

clone_repo "xxhash" \
    "https://github.com/cespare/xxhash.git" \
    "v2.3.0" \
    "xxhash"

clone_repo "go-farm" \
    "https://github.com/dgryski/go-farm.git" \
    "" \
    "go-farm"

clone_repo "go-humanize" \
    "https://github.com/dustin/go-humanize.git" \
    "v1.0.1" \
    "go-humanize"

clone_repo "go-rendezvous" \
    "https://github.com/dgryski/go-rendezvous.git" \
    "" \
    "go-rendezvous"

echo ""
echo "=========================================="
echo "完成! 已拉取的依赖:"
echo "=========================================="
ls -1
echo ""
echo "共 $(ls -1 | wc -l | tr -d ' ') 个依赖"
