// Package logger 提供基于 log/slog 的结构化日志功能。
//
// 该包提供包级别的全局函数，内部委托给一个默认的 logger 实例。
// 默认 logger 输出到 stderr，使用 INFO 级别。
//
// 使用示例:
//
//	logger.Init("debug", "json")
//	logger.Info("服务启动", "port", 8080)
//	logger.With("request_id", "abc").Info("处理请求")
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// 全局变量：当前 logger 实例及其配置
var (
	// defaultLogger 是包级别函数使用的默认 logger 实例
	defaultLogger *slog.Logger

	// currentLevel 保存当前日志级别，用于动态调整
	currentLevel *slog.LevelVar

	// currentWriter 保存当前输出目标，默认为 stderr
	currentWriter io.Writer

	// currentFormat 保存当前输出格式（"json" 或 "console"）
	currentFormat string

	// mu 保护全局状态的并发安全
	mu sync.RWMutex
)

// init 在包加载时初始化默认 logger。
// 默认输出到 stderr，级别为 INFO，格式为 console。
func init() {
	currentLevel = &slog.LevelVar{}
	currentLevel.Set(slog.LevelInfo)
	currentWriter = os.Stderr
	currentFormat = "console"

	handler := buildHandler(currentWriter, currentLevel, currentFormat)
	defaultLogger = slog.New(handler)
}

// Init 初始化全局 logger。
//
// 参数：
//   - level: 日志级别，支持 "debug"、"info"、"warn"、"error"，不区分大小写。
//     无效值默认使用 "info"。
//   - format: 输出格式，支持 "json" 和 "console"。
//     无效值默认使用 "console"。
func Init(level string, format string) {
	mu.Lock()
	defer mu.Unlock()

	// 解析日志级别
	currentLevel = &slog.LevelVar{}
	currentLevel.Set(parseLevel(level))

	// 解析输出格式，无效格式回退为 console
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "json" && format != "console" {
		format = "console"
	}
	currentFormat = format

	// 保持当前 writer 不变（默认 stderr）
	handler := buildHandler(currentWriter, currentLevel, currentFormat)
	defaultLogger = slog.New(handler)
}

// SetOutput 设置日志输出目标。
// 常用于测试中将日志重定向到 bytes.Buffer 进行断言。
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()

	currentWriter = w
	handler := buildHandler(currentWriter, currentLevel, currentFormat)
	defaultLogger = slog.New(handler)
}

// Info 以 INFO 级别输出日志。
// args 为键值对形式的附加字段，例如 Info("消息", "key", "value")。
func Info(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()

	l.Info(msg, args...)
}

// Warn 以 WARN 级别输出日志。
// args 为键值对形式的附加字段。
func Warn(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()

	l.Warn(msg, args...)
}

// Error 以 ERROR 级别输出日志。
// args 为键值对形式的附加字段。
func Error(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()

	l.Error(msg, args...)
}

// Debug 以 DEBUG 级别输出日志。
// args 为键值对形式的附加字段。
func Debug(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()

	l.Debug(msg, args...)
}

// With 返回一个附带预设字段的 *slog.Logger。
// 适合在请求处理等场景中携带上下文信息，例如：
//
//	l := logger.With("request_id", reqID)
//	l.Info("开始处理请求")
func With(args ...any) *slog.Logger {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()

	return l.With(args...)
}

// ============================================================
// 内部辅助函数
// ============================================================

// parseLevel 将字符串级别转换为 slog.Level。
// 不区分大小写，无效值默认返回 slog.LevelInfo。
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// 无效级别回退为 INFO
		return slog.LevelInfo
	}
}

// buildHandler 根据输出目标、级别和格式构建 slog.Handler。
//   - format 为 "json" 时使用 slog.JSONHandler（结构化 JSON 输出）
//   - 其他情况使用 slog.TextHandler（人类可读的控制台输出）
func buildHandler(w io.Writer, level *slog.LevelVar, format string) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if format == "json" {
		return slog.NewJSONHandler(w, opts)
	}
	// 默认使用 TextHandler（console 格式）
	return slog.NewTextHandler(w, opts)
}
