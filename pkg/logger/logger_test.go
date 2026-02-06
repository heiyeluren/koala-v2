package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// 测试：Init() 配置正确的日志级别
// ============================================================

func TestInit_DebugLevel(t *testing.T) {
	// 初始化为 debug 级别，json 格式
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	// debug 级别应该能输出 debug 日志
	Debug("调试消息")
	assert.Contains(t, buf.String(), "调试消息", "debug 级别下 Debug() 应输出消息")
}

func TestInit_InfoLevel(t *testing.T) {
	// 初始化为 info 级别
	Init("info", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	// info 级别下 Debug 不应输出
	Debug("不应出现的调试消息")
	assert.NotContains(t, buf.String(), "不应出现的调试消息",
		"info 级别下 Debug() 不应输出消息")

	// info 级别下 Info 应输出
	buf.Reset()
	Info("信息消息")
	assert.Contains(t, buf.String(), "信息消息", "info 级别下 Info() 应输出消息")
}

func TestInit_WarnLevel(t *testing.T) {
	// 初始化为 warn 级别
	Init("warn", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	// warn 级别下 Info 不应输出
	Info("不应出现的信息消息")
	assert.NotContains(t, buf.String(), "不应出现的信息消息",
		"warn 级别下 Info() 不应输出消息")

	// warn 级别下 Warn 应输出
	buf.Reset()
	Warn("警告消息")
	assert.Contains(t, buf.String(), "警告消息", "warn 级别下 Warn() 应输出消息")
}

func TestInit_ErrorLevel(t *testing.T) {
	// 初始化为 error 级别
	Init("error", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	// error 级别下 Warn 不应输出
	Warn("不应出现的警告消息")
	assert.NotContains(t, buf.String(), "不应出现的警告消息",
		"error 级别下 Warn() 不应输出消息")

	// error 级别下 Error 应输出
	buf.Reset()
	Error("错误消息")
	assert.Contains(t, buf.String(), "错误消息", "error 级别下 Error() 应输出消息")
}

// ============================================================
// 测试：各级别日志函数输出包含指定消息
// ============================================================

func TestInfo_OutputContainsMessage(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Info("测试信息消息", "key", "value")
	output := buf.String()
	assert.Contains(t, output, "测试信息消息", "Info() 输出应包含指定消息")
	assert.Contains(t, output, "key", "Info() 输出应包含附加字段的键")
	assert.Contains(t, output, "value", "Info() 输出应包含附加字段的值")
}

func TestWarn_OutputContainsMessage(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Warn("测试警告消息", "code", 42)
	output := buf.String()
	assert.Contains(t, output, "测试警告消息", "Warn() 输出应包含指定消息")
}

func TestError_OutputContainsMessage(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Error("测试错误消息", "err", "something failed")
	output := buf.String()
	assert.Contains(t, output, "测试错误消息", "Error() 输出应包含指定消息")
}

func TestDebug_OutputContainsMessage(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Debug("测试调试消息", "detail", "verbose info")
	output := buf.String()
	assert.Contains(t, output, "测试调试消息", "Debug() 输出应包含指定消息")
}

// ============================================================
// 测试：SetOutput 将日志输出重定向到自定义 writer
// ============================================================

func TestSetOutput_RedirectsOutput(t *testing.T) {
	Init("debug", "json")

	// 使用自定义 buffer 捕获输出
	var buf bytes.Buffer
	SetOutput(&buf)

	Info("重定向测试消息")

	// 确认消息写入了自定义 writer
	require.NotEmpty(t, buf.String(), "SetOutput 后日志应写入指定 writer")
	assert.Contains(t, buf.String(), "重定向测试消息",
		"自定义 writer 应包含日志消息")
}

func TestSetOutput_MultipleWriters(t *testing.T) {
	Init("debug", "json")

	// 第一个 writer
	var buf1 bytes.Buffer
	SetOutput(&buf1)
	Info("第一个writer")
	assert.Contains(t, buf1.String(), "第一个writer")

	// 切换到第二个 writer
	var buf2 bytes.Buffer
	SetOutput(&buf2)
	Info("第二个writer")
	assert.Contains(t, buf2.String(), "第二个writer")
	// 第一个 writer 不应收到新消息
	assert.NotContains(t, buf1.String(), "第二个writer",
		"切换 writer 后旧 writer 不应再收到日志")
}

// ============================================================
// 测试：各级别输出正确的级别字符串
// ============================================================

func TestLevelString_INFO(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Info("级别测试")
	output := buf.String()
	assert.True(t, strings.Contains(output, "INFO"),
		"Info() 输出应包含级别字符串 INFO，实际输出: %s", output)
}

func TestLevelString_WARN(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Warn("级别测试")
	output := buf.String()
	assert.True(t, strings.Contains(output, "WARN"),
		"Warn() 输出应包含级别字符串 WARN，实际输出: %s", output)
}

func TestLevelString_ERROR(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Error("级别测试")
	output := buf.String()
	assert.True(t, strings.Contains(output, "ERROR"),
		"Error() 输出应包含级别字符串 ERROR，实际输出: %s", output)
}

func TestLevelString_DEBUG(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Debug("级别测试")
	output := buf.String()
	assert.True(t, strings.Contains(output, "DEBUG"),
		"Debug() 输出应包含级别字符串 DEBUG，实际输出: %s", output)
}

// ============================================================
// 测试：console 格式输出
// ============================================================

func TestInit_ConsoleFormat(t *testing.T) {
	Init("debug", "console")

	var buf bytes.Buffer
	SetOutput(&buf)

	Info("控制台格式测试")
	output := buf.String()
	assert.Contains(t, output, "控制台格式测试",
		"console 格式应正常输出消息")
	assert.Contains(t, output, "INFO",
		"console 格式应包含级别字符串 INFO")
}

// ============================================================
// 测试：With() 返回带有预设字段的 logger
// ============================================================

func TestWith_ReturnsLoggerWithFields(t *testing.T) {
	Init("debug", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	// 使用 With 创建带有预设字段的 logger，返回 *slog.Logger
	var l *slog.Logger
	l = With("request_id", "abc-123", "user_id", 456)
	require.NotNil(t, l, "With() 应返回非 nil 的 *slog.Logger")

	l.Info("带字段的消息")
	output := buf.String()
	assert.Contains(t, output, "带字段的消息", "With logger 应输出消息")
	assert.Contains(t, output, "abc-123", "With logger 输出应包含预设字段值 request_id")
	assert.Contains(t, output, "456", "With logger 输出应包含预设字段值 user_id")
}

// ============================================================
// 测试：默认配置（未调用 Init 时的行为）
// ============================================================

func TestDefaultLogger(t *testing.T) {
	// 重新初始化为默认状态用于测试
	Init("info", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	Info("默认logger测试")
	assert.Contains(t, buf.String(), "默认logger测试",
		"默认 logger 应正常工作")
}

// ============================================================
// 测试：无效的日志级别默认使用 INFO
// ============================================================

func TestInit_InvalidLevel(t *testing.T) {
	// 传入无效级别，应该默认使用 info
	Init("invalid_level", "json")

	var buf bytes.Buffer
	SetOutput(&buf)

	// Info 应该输出
	Info("无效级别回退测试")
	assert.Contains(t, buf.String(), "无效级别回退测试",
		"无效级别应回退为 info，Info() 应输出消息")

	// Debug 不应输出（因为回退到 info 级别）
	buf.Reset()
	Debug("不应出现")
	assert.NotContains(t, buf.String(), "不应出现",
		"无效级别回退为 info 后，Debug() 不应输出消息")
}

// ============================================================
// 测试：无效的输出格式默认使用 console
// ============================================================

func TestInit_InvalidFormat(t *testing.T) {
	// 传入无效格式，应该默认使用 console（TextHandler）
	Init("debug", "invalid_format")

	var buf bytes.Buffer
	SetOutput(&buf)

	Debug("无效格式回退测试")
	output := buf.String()
	assert.Contains(t, output, "无效格式回退测试",
		"无效格式应回退为 console，日志应正常输出")
}
