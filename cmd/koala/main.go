// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package main 是 Koala 反作弊频率控制系统的入口点。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"koala/internal/api"
	"koala/internal/config"
	"koala/internal/engine"
	"koala/internal/storage"
	"koala/internal/storage/local"
	"koala/internal/storage/manager"
	"koala/internal/storage/redis"
	"koala/pkg/logger"
)

var (
	configFile = flag.String("config", "conf/koala.toml", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
)

// 版本信息
const (
	Version   = "2.0.0"
	BuildTime = "unknown"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("Koala 反作弊频率控制系统 v%s (构建时间: %s)\n", Version, BuildTime)
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化存储
	store, err := initStorage(cfg)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}
	defer store.Close()

	// 加载规则配置
	rulesConfig, err := config.LoadRules(cfg.Rules.File)
	if err != nil {
		log.Fatalf("加载规则配置失败: %v", err)
	}

	// 加载字典
	var dictManager *config.DictManager
	if len(rulesConfig.Dicts) > 0 {
		dictManager, err = config.LoadDicts(rulesConfig.Dicts)
		if err != nil {
			log.Fatalf("加载字典失败: %v", err)
		}
	} else {
		dictManager = config.NewDictManager()
	}

	// 将字典同步到 matcher 的 DictMatcher
	engine.SyncDictsToMatcher(dictManager)

	// 创建规则引擎
	eng := engine.New(
		engine.WithStorage(store),
		engine.WithDicts(dictManager),
	)

	// 加载规则到引擎
	if err := eng.LoadRules(rulesConfig); err != nil {
		log.Fatalf("加载规则失败: %v", err)
	}

	// 创建引擎适配器（将 engine.Engine 适配为 api.Engine 接口）
	engineAdapter := &EngineAdapter{engine: eng}

	// 创建 HTTP Handler
	handler := api.NewHandler(engineAdapter)

	// 创建路由配置
	routerConfig := &api.RouterConfig{
		RequestTimeout: cfg.Server.ReadTimeout,
		EnableCORS:     true,
		EnableMetrics:  cfg.Metrics.Enabled,
	}

	// 创建服务器
	server := api.NewServer(handler, cfg.Server.Listen, routerConfig)

	// 启动配置热重载
	if cfg.Rules.ReloadInterval > 0 {
		watcher, err := startConfigWatcher(cfg, eng, dictManager)
		if err != nil {
			log.Printf("警告: 启动配置监听失败: %v", err)
		} else {
			defer watcher.Stop()
		}
	}

	// 启动服务器
	go func() {
		logger.Info("Koala 服务启动", "addr", cfg.Server.Listen)
		if err := server.Run(); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 使用配置的超时时间优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭错误", "error", err)
	}

	logger.Info("服务器已关闭")
}

// EngineAdapter 将 engine.Engine 适配为 api.Engine 接口。
type EngineAdapter struct {
	engine *engine.Engine
}

// Browse 实现 api.Engine 接口的 Browse 方法。
func (a *EngineAdapter) Browse(ctx context.Context, req *api.EngineRequest) (*api.EngineResponse, error) {
	// 转换请求类型
	engineReq := &engine.Request{
		Act: req.Act,
		UID: req.UID,
		IP:  req.IP,
		DID: req.DID,
		Ext: req.Ext,
	}

	// 调用引擎的 Browse 方法
	resp, err := a.engine.Browse(ctx, engineReq)
	if err != nil {
		return nil, err
	}

	// 如果需要自动更新计数器
	if req.Update && resp.Allowed {
		_, _ = a.engine.Check(ctx, engineReq)
	}

	// 转换响应类型
	return &api.EngineResponse{
		Allowed:  resp.Allowed,
		RuleName: resp.RuleName,
		Code:     resp.Code,
		Message:  resp.Message,
		AuthType: resp.AuthType,
	}, nil
}

// Update 实现 api.Engine 接口的 Update 方法。
func (a *EngineAdapter) Update(ctx context.Context, req *api.EngineRequest) error {
	// 转换请求类型
	engineReq := &engine.Request{
		Act: req.Act,
		UID: req.UID,
		IP:  req.IP,
		DID: req.DID,
		Ext: req.Ext,
	}

	// 调用引擎的 Check 方法（Check 方法会更新计数器）
	_, err := a.engine.Check(ctx, engineReq)
	return err
}

// initStorage 根据配置初始化存储
func initStorage(cfg *config.Config) (storage.Storage, error) {
	// 解析本地存储最大容量
	maxCost, err := cfg.Storage.Local.GetMaxSizeBytes()
	if err != nil {
		maxCost = 512 * 1024 * 1024 // 默认 512MB
	}

	// 设置计数器数量
	numCounters := cfg.Storage.Local.NumCounters
	if numCounters == 0 {
		numCounters = 100000 // 默认 10 万
	}

	// 创建本地存储（始终需要作为备用）
	localStorage, err := local.New(local.Config{
		MaxCost:     maxCost,
		NumCounters: int64(numCounters),
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("创建本地存储失败: %w", err)
	}

	// 如果配置使用 Redis
	if cfg.Storage.Type == "redis" {
		redisStorage, err := redis.New(redis.Config{
			Addr:         cfg.Storage.Redis.Addr,
			Password:     cfg.Storage.Redis.Password,
			DB:           cfg.Storage.Redis.DB,
			PoolSize:     cfg.Storage.Redis.PoolSize,
			DialTimeout:  cfg.Storage.Redis.DialTimeout,
			ReadTimeout:  cfg.Storage.Redis.ReadTimeout,
			WriteTimeout: cfg.Storage.Redis.WriteTimeout,
		})
		if err != nil {
			log.Printf("警告: 创建 Redis 存储失败，使用本地存储: %v", err)
			return localStorage, nil
		}

		// 使用存储管理器实现自动故障转移
		return manager.NewWithOptions(manager.Options{
			Primary:             redisStorage,
			Fallback:            localStorage,
			MaxFailures:         3,
			HealthCheckInterval: 5 * time.Second,
			RecoveryInterval:    10 * time.Second,
		}), nil
	}

	return localStorage, nil
}

// startConfigWatcher 启动配置文件监听
func startConfigWatcher(cfg *config.Config, eng *engine.Engine, dictManager *config.DictManager) (*config.ConfigWatcher, error) {
	watcher, err := config.NewConfigWatcher(*configFile, func(event config.ConfigChangeEvent) {
		if event.Error != nil {
			log.Printf("配置变更错误: %v", event.Error)
			return
		}

		log.Printf("检测到配置变更: %s (%s)", event.Path, event.Type)

		// 如果是规则变更，重新加载规则
		if event.Type == config.ConfigChangeTypeRules {
			rulesConfig, err := config.LoadRules(cfg.Rules.File)
			if err != nil {
				log.Printf("重新加载规则失败: %v", err)
				return
			}

			// 更新引擎规则
			if err := eng.LoadRules(rulesConfig); err != nil {
				log.Printf("更新引擎规则失败: %v", err)
				return
			}

			log.Printf("规则热重载成功")
		}

		// 如果是字典变更，将字典同步到 matcher
		if event.Type == config.ConfigChangeTypeDict {
			engine.SyncDictsToMatcher(dictManager)
			log.Printf("字典同步到 matcher 成功")
		}
	})
	if err != nil {
		return nil, err
	}

	// 加载初始配置
	if err := watcher.Load(); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 开始监听
	if err := watcher.StartWatching(); err != nil {
		return nil, fmt.Errorf("启动监听失败: %w", err)
	}

	return watcher, nil
}
