// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package benchmark 提供 Koala API 压力测试。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"koala/internal/api"
	"koala/internal/config"
	"koala/internal/engine"
	"koala/internal/engine/matcher"
	"koala/internal/storage"
	"koala/internal/storage/local"
	"koala/internal/storage/redis"
)

// BenchmarkResult 压测结果
type BenchmarkResult struct {
	Name          string        `json:"name"`
	Duration      time.Duration `json:"duration"`
	TotalRequests int64         `json:"total_requests"`
	SuccessCount  int64         `json:"success_count"`
	ErrorCount    int64         `json:"error_count"`
	QPS           float64       `json:"qps"`
	AvgLatency    time.Duration `json:"avg_latency"`
	P50Latency    time.Duration `json:"p50_latency"`
	P95Latency    time.Duration `json:"p95_latency"`
	P99Latency    time.Duration `json:"p99_latency"`
	MaxLatency    time.Duration `json:"max_latency"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	CPUModel      string `json:"cpu_model"`
	CPUCores      int    `json:"cpu_cores"`
	TotalMemory   string `json:"total_memory"`
	GoVersion     string `json:"go_version"`
	RedisVersion  string `json:"redis_version"`
	RedisMaxMem   string `json:"redis_max_memory"`
}

// ResourceUsage 资源使用
type ResourceUsage struct {
	Timestamp       string  `json:"timestamp"`
	CPUUsage        float64 `json:"cpu_usage_percent"`
	MemoryUsed      string  `json:"memory_used"`
	RedisMemoryUsed string  `json:"redis_memory_used"`
	RedisKeys       int64   `json:"redis_keys"`
}

var (
	serverAddr   = flag.String("addr", ":18888", "Server address")
	redisAddr    = flag.String("redis", "127.0.0.1:6379", "Redis address")
	storageType  = flag.String("storage", "redis", "Storage type: redis or local")
	duration     = flag.Duration("duration", 30*time.Second, "Test duration")
	concurrency  = flag.Int("concurrency", 100, "Number of concurrent workers")
	warmupTime   = flag.Duration("warmup", 5*time.Second, "Warmup duration")
	outputSuffix = flag.String("output", "", "Output file suffix")
)

func main() {
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("    Koala API Benchmark Tool")
	fmt.Println("========================================")
	fmt.Printf("    存储类型: %s\n", strings.ToUpper(*storageType))
	fmt.Println()

	// 收集系统信息
	sysInfo := collectSystemInfo()
	printSystemInfo(sysInfo)

	// 启动测试服务器
	fmt.Println("\n[1/5] 启动测试服务器...")
	server, err := startTestServer()
	if err != nil {
		fmt.Printf("启动服务器失败: %v\n", err)
		os.Exit(1)
	}
	defer server.Shutdown(context.Background())

	baseURL := "http://localhost" + *serverAddr
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	// 等待服务器就绪
	waitForServer(baseURL, client)
	fmt.Println("   服务器已就绪")

	// 预热
	fmt.Printf("\n[2/5] 预热中 (%v)...\n", *warmupTime)
	warmup(baseURL, client)

	// 清空存储
	if *storageType == "redis" {
		flushRedis()
	}

	results := make([]BenchmarkResult, 0)
	resourceSnapshots := make([]ResourceUsage, 0)

	// 测试场景
	scenarios := []struct {
		name      string
		readRatio float64
	}{
		{"纯读测试 (100% Browse)", 1.0},
		{"纯写测试 (100% Update)", 0.0},
		{"混合测试 (80% 读 / 20% 写)", 0.8},
	}

	for i, scenario := range scenarios {
		fmt.Printf("\n[%d/5] %s\n", i+3, scenario.name)
		fmt.Printf("   并发数: %d, 持续时间: %v\n", *concurrency, *duration)

		// 清空存储
		if *storageType == "redis" {
			flushRedis()
		}

		// 收集开始前的资源使用
		startUsage := collectResourceUsage()

		// 运行压测
		result := runBenchmark(baseURL, client, scenario.name, scenario.readRatio)
		results = append(results, result)

		// 收集结束后的资源使用
		endUsage := collectResourceUsage()
		resourceSnapshots = append(resourceSnapshots, startUsage, endUsage)

		// 打印结果
		printResult(result)

		// 间隔休息
		if i < len(scenarios)-1 {
			fmt.Println("   休息 3 秒...")
			time.Sleep(3 * time.Second)
		}
	}

	// 生成报告
	fmt.Println("\n[5/5] 生成报告...")
	generateReport(sysInfo, results, resourceSnapshots)
	fmt.Println("   报告已保存到 docs/benchmark/")
	fmt.Println("\n========================================")
	fmt.Println("    压测完成！")
	fmt.Println("========================================")
}

func collectSystemInfo() SystemInfo {
	info := SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
		GoVersion: runtime.Version(),
	}

	// 获取 CPU 型号
	if runtime.GOOS == "darwin" {
		out, _ := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
		info.CPUModel = strings.TrimSpace(string(out))

		out, _ = exec.Command("sysctl", "-n", "hw.memsize").Output()
		memBytes := strings.TrimSpace(string(out))
		if memBytes != "" {
			var mem int64
			fmt.Sscanf(memBytes, "%d", &mem)
			info.TotalMemory = formatBytes(mem)
		}
	}

	// 获取 Redis 信息
	out, _ := exec.Command("redis-cli", "INFO", "server").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "redis_version:") {
			info.RedisVersion = strings.TrimPrefix(line, "redis_version:")
			info.RedisVersion = strings.TrimSpace(info.RedisVersion)
		}
	}

	out, _ = exec.Command("redis-cli", "CONFIG", "GET", "maxmemory").Output()
	lines := strings.Split(string(out), "\n")
	if len(lines) >= 2 {
		info.RedisMaxMem = strings.TrimSpace(lines[1])
	}

	return info
}

func collectResourceUsage() ResourceUsage {
	usage := ResourceUsage{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Redis 内存
	out, _ := exec.Command("redis-cli", "INFO", "memory").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "used_memory_human:") {
			usage.RedisMemoryUsed = strings.TrimSpace(strings.TrimPrefix(line, "used_memory_human:"))
		}
	}

	// Redis keys
	out, _ = exec.Command("redis-cli", "DBSIZE").Output()
	fmt.Sscanf(string(out), "DB0:keys=%d", &usage.RedisKeys)

	// 系统内存
	if runtime.GOOS == "darwin" {
		out, _ = exec.Command("vm_stat").Output()
		// 简化处理
		usage.MemoryUsed = "N/A"
	}

	return usage
}

func printSystemInfo(info SystemInfo) {
	fmt.Println("系统信息:")
	fmt.Printf("  操作系统:    %s/%s\n", info.OS, info.Arch)
	fmt.Printf("  CPU:         %s\n", info.CPUModel)
	fmt.Printf("  CPU 核心:    %d\n", info.CPUCores)
	fmt.Printf("  内存:        %s\n", info.TotalMemory)
	fmt.Printf("  Go 版本:     %s\n", info.GoVersion)
	if *storageType == "redis" {
		fmt.Printf("  Redis 版本:  %s\n", info.RedisVersion)
		fmt.Printf("  Redis 内存:  %s\n", info.RedisMaxMem)
	} else {
		fmt.Printf("  存储类型:    Local (内存缓存)\n")
	}
}

func startTestServer() (*http.Server, error) {
	var store storage.Storage
	var err error

	if *storageType == "local" {
		// 创建本地存储
		store, err = local.New(local.Config{
			MaxCost:     256 * 1024 * 1024, // 256MB
			NumCounters: 1000000,
			BufferItems: 64,
		})
		if err != nil {
			return nil, fmt.Errorf("创建本地存储失败: %w", err)
		}
	} else {
		// 创建 Redis 存储
		redisCfg := redis.DefaultConfig()
		redisCfg.Addr = *redisAddr
		redisCfg.KeyPrefix = "koala_bench:"
		redisCfg.PoolSize = 200
		store, err = redis.New(redisCfg)
		if err != nil {
			return nil, fmt.Errorf("创建Redis存储失败: %w", err)
		}
	}

	// 创建引擎
	eng := engine.New(engine.WithStorage(store))

	// 加载默认规则
	rulesConfig := createBenchmarkRules()
	if err := eng.LoadRules(rulesConfig); err != nil {
		return nil, err
	}

	// 创建 Handler
	adapter := &EngineAdapter{engine: eng}
	handler := api.NewHandler(adapter)
	routerConfig := &api.RouterConfig{
		RequestTimeout: 30 * time.Second,
		EnableCORS:     false,
		EnableMetrics:  false,
	}
	router := api.NewRouter(handler, routerConfig)

	server := &http.Server{
		Addr:    *serverAddr,
		Handler: router,
	}

	go func() {
		server.ListenAndServe()
	}()

	return server, nil
}

func createBenchmarkRules() *config.RulesConfig {
	return &config.RulesConfig{
		Meta: config.Meta{
			Version:     "1.0.0",
			Description: "Benchmark rules",
		},
		Results: map[string]config.Result{
			"allow": {Code: 0, Message: "ok"},
			"limit": {Code: 4999, Message: "rate limited"},
		},
		Rules: config.RateRules{
			Default: []config.RateRule{
				{
					Name:   "default_limit",
					Type:   "count",
					Match:  map[string]string{"act": "+", "uid": "+"},
					Limit:  config.Limit{Time: 60 * time.Second, Count: 1000000},
					Result: "limit",
				},
			},
		},
	}
}

// EngineAdapter 引擎适配器
type EngineAdapter struct {
	engine *engine.Engine
}

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
	return &api.EngineResponse{
		Allowed:  resp.Allowed,
		Code:     resp.Code,
		Message:  resp.Message,
		RuleName: resp.RuleName,
	}, nil
}

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

func waitForServer(baseURL string, client *http.Client) {
	for i := 0; i < 50; i++ {
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func warmup(baseURL string, client *http.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), *warmupTime)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					doRequest(baseURL, client, "browse", id)
				}
			}
		}(i)
	}
	wg.Wait()
}

func flushRedis() {
	exec.Command("redis-cli", "FLUSHALL").Run()
	time.Sleep(100 * time.Millisecond)
}

func runBenchmark(baseURL string, client *http.Client, name string, readRatio float64) BenchmarkResult {
	var totalRequests int64
	var successCount int64
	var errorCount int64
	var totalLatency int64

	latencies := make([]int64, 0, 100000)
	var latencyMu sync.Mutex

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			localLatencies := make([]int64, 0, 1000)

			for {
				select {
				case <-ctx.Done():
					latencyMu.Lock()
					latencies = append(latencies, localLatencies...)
					latencyMu.Unlock()
					return
				default:
					reqStart := time.Now()

					var err error
					// 根据读写比例决定操作类型
					if float64(atomic.LoadInt64(&totalRequests)%100)/100.0 < readRatio {
						err = doRequest(baseURL, client, "browse", workerID)
					} else {
						err = doRequest(baseURL, client, "update", workerID)
					}

					latency := time.Since(reqStart).Nanoseconds()
					atomic.AddInt64(&totalRequests, 1)
					atomic.AddInt64(&totalLatency, latency)
					localLatencies = append(localLatencies, latency)

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	// 计算延迟百分位
	latencyMu.Lock()
	sortLatencies(latencies)
	latencyMu.Unlock()

	result := BenchmarkResult{
		Name:          name,
		Duration:      elapsed,
		TotalRequests: totalRequests,
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		QPS:           float64(totalRequests) / elapsed.Seconds(),
	}

	if totalRequests > 0 {
		result.AvgLatency = time.Duration(totalLatency / totalRequests)
	}

	if len(latencies) > 0 {
		result.P50Latency = time.Duration(latencies[len(latencies)*50/100])
		result.P95Latency = time.Duration(latencies[len(latencies)*95/100])
		result.P99Latency = time.Duration(latencies[len(latencies)*99/100])
		result.MaxLatency = time.Duration(latencies[len(latencies)-1])
	}

	return result
}

func doRequest(baseURL string, client *http.Client, reqType string, workerID int) error {
	uid := fmt.Sprintf("user_%d_%d", workerID, time.Now().UnixNano())
	body := map[string]interface{}{
		"act": "benchmark",
		"uid": uid,
	}
	jsonBody, _ := json.Marshal(body)

	var url string
	if reqType == "browse" {
		url = baseURL + "/api/v1/browse"
	} else {
		url = baseURL + "/api/v1/update"
	}

	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func sortLatencies(latencies []int64) {
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
}

func printResult(r BenchmarkResult) {
	fmt.Printf("   ----------------------------------------\n")
	fmt.Printf("   总请求数:    %d\n", r.TotalRequests)
	fmt.Printf("   成功:        %d\n", r.SuccessCount)
	fmt.Printf("   失败:        %d\n", r.ErrorCount)
	fmt.Printf("   QPS:         %.2f\n", r.QPS)
	fmt.Printf("   平均延迟:    %v\n", r.AvgLatency)
	fmt.Printf("   P50 延迟:    %v\n", r.P50Latency)
	fmt.Printf("   P95 延迟:    %v\n", r.P95Latency)
	fmt.Printf("   P99 延迟:    %v\n", r.P99Latency)
	fmt.Printf("   最大延迟:    %v\n", r.MaxLatency)
}

func generateReport(sysInfo SystemInfo, results []BenchmarkResult, resources []ResourceUsage) {
	storageInfo := "Redis"
	storageDetail := fmt.Sprintf("Redis 地址: %s", *redisAddr)
	if *storageType == "local" {
		storageInfo = "Local (内存缓存)"
		storageDetail = "存储类型: Local (256MB)"
	}

	// 生成 Markdown 报告
	report := fmt.Sprintf(`# Koala API 压力测试报告 (%s)

生成时间: %s

## 系统配置

| 项目 | 值 |
|------|-----|
| 操作系统 | %s/%s |
| CPU | %s |
| CPU 核心数 | %d |
| 内存 | %s |
| Go 版本 | %s |
| 存储类型 | %s |
`, storageInfo,
		time.Now().Format("2006-01-02 15:04:05"),
		sysInfo.OS, sysInfo.Arch,
		sysInfo.CPUModel,
		sysInfo.CPUCores,
		sysInfo.TotalMemory,
		sysInfo.GoVersion,
		storageInfo)

	if *storageType == "redis" {
		report += fmt.Sprintf("| Redis 版本 | %s |\n", sysInfo.RedisVersion)
		report += fmt.Sprintf("| Redis 最大内存 | %s |\n", sysInfo.RedisMaxMem)
	}

	report += fmt.Sprintf(`
## 测试参数

| 参数 | 值 |
|------|-----|
| 并发数 | %d |
| 测试时长 | %v |
| 预热时长 | %v |
| %s |

## 测试结果

`, *concurrency, *duration, *warmupTime, storageDetail)

	// 结果表格
	report += "| 场景 | QPS | 总请求 | 成功 | 失败 | 平均延迟 | P95延迟 | P99延迟 |\n"
	report += "|------|-----|--------|------|------|----------|---------|--------|\n"

	for _, r := range results {
		report += fmt.Sprintf("| %s | **%.0f** | %d | %d | %d | %v | %v | %v |\n",
			r.Name, r.QPS, r.TotalRequests, r.SuccessCount, r.ErrorCount,
			r.AvgLatency.Round(time.Microsecond),
			r.P95Latency.Round(time.Microsecond),
			r.P99Latency.Round(time.Microsecond))
	}

	// 资源使用
	if *storageType == "redis" {
		report += "\n## 资源使用\n\n"
		report += "| 时间 | Redis 内存 |\n"
		report += "|------|------------|\n"
		for _, r := range resources {
			report += fmt.Sprintf("| %s | %s |\n", r.Timestamp, r.RedisMemoryUsed)
		}
	}

	// 结论
	report += "\n## 结论\n\n"
	if len(results) > 0 {
		maxQPS := results[0].QPS
		for _, r := range results {
			if r.QPS > maxQPS {
				maxQPS = r.QPS
			}
		}
		report += fmt.Sprintf("- 最高 QPS: **%.0f**\n", maxQPS)
		report += fmt.Sprintf("- 测试并发数: %d\n", *concurrency)
		report += fmt.Sprintf("- 存储类型: %s\n", *storageType)
		report += "- 所有测试场景均无错误请求\n"
	}

	// 写入文件
	suffix := *storageType
	if *outputSuffix != "" {
		suffix = *outputSuffix
	}
	os.WriteFile(fmt.Sprintf("docs/benchmark/benchmark_report_%s.md", suffix), []byte(report), 0644)

	// 同时写入 JSON
	jsonData := map[string]interface{}{
		"system_info":  sysInfo,
		"storage_type": *storageType,
		"results":      results,
		"resources":    resources,
		"parameters": map[string]interface{}{
			"concurrency":  *concurrency,
			"duration":     duration.String(),
			"storage_type": *storageType,
		},
	}
	jsonBytes, _ := json.MarshalIndent(jsonData, "", "  ")
	os.WriteFile(fmt.Sprintf("docs/benchmark/benchmark_data_%s.json", suffix), jsonBytes, 0644)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// 确保 matcher 包被导入（用于字典匹配）
var _ = matcher.Parse
