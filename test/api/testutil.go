// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供 Koala API 端到端测试。
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"koala/internal/api"
	"koala/internal/config"
	"koala/internal/engine"
	"koala/internal/engine/matcher"
	"koala/internal/storage"
	"koala/internal/storage/local"
	"koala/internal/storage/redis"
)

// TestServer 封装测试服务器。
type TestServer struct {
	server     *http.Server
	engine     *engine.Engine
	baseURL    string
	httpClient *http.Client
	mu         sync.Mutex
	started    bool
}

// 全局测试服务器实例
var (
	globalServer *TestServer
	serverOnce   sync.Once
	serverMu     sync.Mutex
)

// GetTestServer 获取或创建测试服务器单例。
func GetTestServer(t *testing.T) *TestServer {
	serverMu.Lock()
	defer serverMu.Unlock()

	if globalServer != nil && globalServer.started {
		return globalServer
	}

	serverOnce.Do(func() {
		var err error
		globalServer, err = NewTestServer()
		if err != nil {
			t.Fatalf("创建测试服务器失败: %v", err)
		}
		if err := globalServer.Start(); err != nil {
			t.Fatalf("启动测试服务器失败: %v", err)
		}
	})

	return globalServer
}

// NewTestServer 创建新的测试服务器。
func NewTestServer() (*TestServer, error) {
	// 获取测试数据目录
	testdataDir := getTestdataDir()

	// 加载配置
	configPath := filepath.Join(testdataDir, "koala.toml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 加载规则
	rulesPath := filepath.Join(testdataDir, "rules.toml")
	rulesConfig, err := config.LoadRules(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("加载规则失败: %w", err)
	}

	// 加载字典
	var dictManager *config.DictManager
	if len(rulesConfig.Dicts) > 0 {
		// 将字典路径转换为绝对路径
		dictPaths := make(map[string]string)
		for name, path := range rulesConfig.Dicts {
			dictPaths[name] = filepath.Join(testdataDir, path)
		}
		dictManager, err = config.LoadDicts(dictPaths)
		if err != nil {
			return nil, fmt.Errorf("加载字典失败: %w", err)
		}

		// 将字典注册到全局匹配器
		for name := range rulesConfig.Dicts {
			dict, ok := dictManager.Get(name)
			if ok {
				// 转换为matcher需要的格式
				dictMap := make(map[string]bool)
				for _, entry := range dict.List() {
					dictMap[entry] = true
				}
				matcher.RegisterDict(name, dictMap)
			}
		}
	} else {
		dictManager = config.NewDictManager()
	}

	// 创建存储（支持通过环境变量切换存储类型）
	var store storage.Storage
	storageType := os.Getenv("KOALA_TEST_STORAGE")
	if storageType == "redis" {
		// 使用 Redis 存储
		redisAddr := os.Getenv("KOALA_TEST_REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "127.0.0.1:6379"
		}
		redisCfg := redis.DefaultConfig()
		redisCfg.Addr = redisAddr
		redisCfg.KeyPrefix = "koala_test:" // 测试专用前缀
		store, err = redis.New(redisCfg)
		if err != nil {
			return nil, fmt.Errorf("创建Redis存储失败: %w", err)
		}
	} else {
		// 默认使用本地存储
		store, err = local.New(local.Config{
			MaxCost:     64 * 1024 * 1024,
			NumCounters: 10000,
			BufferItems: 64,
		})
		if err != nil {
			return nil, fmt.Errorf("创建本地存储失败: %w", err)
		}
	}

	// 创建引擎
	eng := engine.New(
		engine.WithStorage(store),
		engine.WithDicts(dictManager),
	)

	// 加载规则
	if err := eng.LoadRules(rulesConfig); err != nil {
		return nil, fmt.Errorf("加载规则到引擎失败: %w", err)
	}

	// 创建引擎适配器
	adapter := &EngineAdapter{engine: eng}

	// 创建 Handler 和 Router
	handler := api.NewHandler(adapter)
	routerConfig := &api.RouterConfig{
		RequestTimeout: cfg.Server.ReadTimeout,
		EnableCORS:     true,
		EnableMetrics:  true,
	}
	router := api.NewRouter(handler, routerConfig)

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: router,
	}

	return &TestServer{
		server:  server,
		engine:  eng,
		baseURL: "http://localhost" + cfg.Server.Listen,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// EngineAdapter 将 engine.Engine 适配为 api.Engine 接口。
type EngineAdapter struct {
	engine *engine.Engine
}

// Browse 实现 api.Engine 接口。
func (a *EngineAdapter) Browse(ctx context.Context, req *api.EngineRequest) (*api.EngineResponse, error) {
	engineReq := &engine.Request{
		Act: req.Act,
		UID: req.UID,
		IP:  req.IP,
		DID: req.DID,
		Ext: req.Ext,
	}

	resp, err := a.engine.Browse(ctx, engineReq)
	if err != nil {
		return nil, err
	}

	// 如果需要自动更新计数器
	if req.Update && resp.Allowed {
		_, _ = a.engine.Check(ctx, engineReq)
	}

	return &api.EngineResponse{
		Allowed:  resp.Allowed,
		RuleName: resp.RuleName,
		Code:     resp.Code,
		Message:  resp.Message,
		AuthType: resp.AuthType,
	}, nil
}

// Update 实现 api.Engine 接口。
func (a *EngineAdapter) Update(ctx context.Context, req *api.EngineRequest) error {
	engineReq := &engine.Request{
		Act: req.Act,
		UID: req.UID,
		IP:  req.IP,
		DID: req.DID,
		Ext: req.Ext,
	}

	_, err := a.engine.Check(ctx, engineReq)
	return err
}

// Start 启动测试服务器。
func (s *TestServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("服务器错误: %v\n", err)
		}
	}()

	// 等待服务器启动
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := s.httpClient.Get(s.baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				s.started = true
				return nil
			}
		}
	}

	return fmt.Errorf("服务器启动超时")
}

// Stop 停止测试服务器。
func (s *TestServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	s.started = false
	return nil
}

// BaseURL 返回服务器基础URL。
func (s *TestServer) BaseURL() string {
	return s.baseURL
}

// Client 返回 HTTP 客户端。
func (s *TestServer) Client() *http.Client {
	return s.httpClient
}

// Engine 返回引擎实例（用于重置状态等）。
func (s *TestServer) Engine() *engine.Engine {
	return s.engine
}

// ========== HTTP 请求辅助方法 ==========

// APIRequest 表示 API 请求体。
type APIRequest struct {
	Act    string            `json:"act"`
	UID    string            `json:"uid,omitempty"`
	IP     string            `json:"ip,omitempty"`
	DID    string            `json:"did,omitempty"`
	Ext    map[string]string `json:"ext,omitempty"`
	Update bool              `json:"update,omitempty"`
}

// APIResponse 表示 API 响应体。
type APIResponse struct {
	Allowed  bool   `json:"allowed"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	RuleName string `json:"rule_name,omitempty"`
	AuthType int    `json:"auth_type,omitempty"`
}

// BatchRequest 表示批量请求体。
type BatchRequest struct {
	Requests []BatchItem `json:"requests"`
}

// BatchItem 表示批量请求中的单项。
type BatchItem struct {
	ID  string            `json:"id"`
	Act string            `json:"act"`
	UID string            `json:"uid,omitempty"`
	IP  string            `json:"ip,omitempty"`
	DID string            `json:"did,omitempty"`
	Ext map[string]string `json:"ext,omitempty"`
}

// BatchResponse 表示批量响应体。
type BatchResponse struct {
	Results []BatchResult `json:"results"`
}

// BatchResult 表示批量响应中的单项结果。
type BatchResult struct {
	ID       string `json:"id"`
	Allowed  bool   `json:"allowed"`
	Code     int    `json:"code"`
	Message  string `json:"message"`
	RuleName string `json:"rule_name,omitempty"`
	AuthType int    `json:"auth_type,omitempty"`
}

// HealthResponse 表示健康检查响应。
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ReadyResponse 表示就绪检查响应。
type ReadyResponse struct {
	Ready     bool   `json:"ready"`
	Timestamp string `json:"timestamp,omitempty"`
}

// PostJSON 发送 POST JSON 请求。
func (s *TestServer) PostJSON(path string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return s.httpClient.Do(req)
}

// Get 发送 GET 请求。
func (s *TestServer) Get(path string) (*http.Response, error) {
	return s.httpClient.Get(s.baseURL + path)
}

// Browse 调用 Browse API。
func (s *TestServer) Browse(req APIRequest) (*APIResponse, *http.Response, error) {
	resp, err := s.PostJSON("/api/v1/browse", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, resp, err
	}

	return &apiResp, resp, nil
}

// Update 调用 Update API。
func (s *TestServer) Update(req APIRequest) (*APIResponse, *http.Response, error) {
	resp, err := s.PostJSON("/api/v1/update", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, resp, err
	}

	return &apiResp, resp, nil
}

// Batch 调用 Batch API。
func (s *TestServer) Batch(req BatchRequest) (*BatchResponse, *http.Response, error) {
	resp, err := s.PostJSON("/api/v1/batch", req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	var batchResp BatchResponse
	if err := json.Unmarshal(body, &batchResp); err != nil {
		return nil, resp, err
	}

	return &batchResp, resp, nil
}

// Health 调用健康检查 API。
func (s *TestServer) Health() (*HealthResponse, *http.Response, error) {
	resp, err := s.Get("/health")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(body, &healthResp); err != nil {
		return nil, resp, err
	}

	return &healthResp, resp, nil
}

// Ready 调用就绪检查 API。
func (s *TestServer) Ready() (*ReadyResponse, *http.Response, error) {
	resp, err := s.Get("/ready")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}

	var readyResp ReadyResponse
	if err := json.Unmarshal(body, &readyResp); err != nil {
		return nil, resp, err
	}

	return &readyResp, resp, nil
}

// Metrics 调用指标 API。
func (s *TestServer) Metrics() (string, *http.Response, error) {
	resp, err := s.Get("/metrics")
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp, err
	}

	return string(body), resp, nil
}

// PostRaw 发送原始请求体。
func (s *TestServer) PostRaw(path string, body []byte, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, s.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return s.httpClient.Do(req)
}

// ========== 辅助函数 ==========

// getTestdataDir 获取测试数据目录。
func getTestdataDir() string {
	// 尝试从环境变量获取
	if dir := os.Getenv("KOALA_TESTDATA_DIR"); dir != "" {
		return dir
	}

	// 尝试找到 testdata 目录
	// 首先尝试当前目录
	if _, err := os.Stat("testdata"); err == nil {
		return "testdata"
	}

	// 尝试相对于测试文件的路径
	if _, err := os.Stat("test/api/testdata"); err == nil {
		return "test/api/testdata"
	}

	// 尝试向上查找
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		testdata := filepath.Join(dir, "test", "api", "testdata")
		if _, err := os.Stat(testdata); err == nil {
			return testdata
		}
		dir = filepath.Dir(dir)
	}

	// 默认返回
	return "testdata"
}

// UniqueID 生成唯一ID用于测试。
func UniqueID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// RepeatString 重复字符串。
func RepeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
