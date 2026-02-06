package errors

import (
	"errors"
	"fmt"
	"testing"
)

// TestNewInternal 测试 NewInternal 能正确创建包含内部和外部消息的错误
func TestNewInternal(t *testing.T) {
	internal := "数据库连接超时: dial tcp 10.0.0.1:3306 timeout"
	external := "服务暂时不可用，请稍后重试"

	err := NewInternal(internal, external)

	if err == nil {
		t.Fatal("NewInternal 不应返回 nil")
	}
	if err.internal != internal {
		t.Errorf("内部消息不匹配: 期望 %q, 得到 %q", internal, err.internal)
	}
	if err.external != external {
		t.Errorf("外部消息不匹配: 期望 %q, 得到 %q", external, err.external)
	}
}

// TestError 测试 Error() 方法返回内部详细错误信息
func TestError(t *testing.T) {
	internal := "redis SET 失败: connection refused"
	external := "操作失败"

	err := NewInternal(internal, external)

	if got := err.Error(); got != internal {
		t.Errorf("Error() 应返回内部消息: 期望 %q, 得到 %q", internal, got)
	}
}

// TestSafeMessage 测试 SafeMessage() 方法返回对外安全消息
func TestSafeMessage(t *testing.T) {
	internal := "用户表查询失败: SQL syntax error near 'SELECT * FROM users WHER'"
	external := "查询失败，请稍后重试"

	err := NewInternal(internal, external)

	if got := err.SafeMessage(); got != external {
		t.Errorf("SafeMessage() 应返回外部消息: 期望 %q, 得到 %q", external, got)
	}
}

// TestWrap 测试 Wrap() 能正确包装已有错误并添加安全消息
func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("原始错误: file not found")
	safeMsg := "请求的资源不存在"

	wrappedErr := Wrap(originalErr, safeMsg)

	if wrappedErr == nil {
		t.Fatal("Wrap 不应返回 nil")
	}

	// 验证 Error() 返回原始错误信息
	if got := wrappedErr.Error(); got != originalErr.Error() {
		t.Errorf("Wrap 后 Error() 应返回原始错误信息: 期望 %q, 得到 %q", originalErr.Error(), got)
	}

	// 验证 SafeMessage() 返回安全消息
	if got := wrappedErr.SafeMessage(); got != safeMsg {
		t.Errorf("Wrap 后 SafeMessage() 应返回安全消息: 期望 %q, 得到 %q", safeMsg, got)
	}

	// 验证 Unwrap() 能获取原始错误
	if unwrapped := wrappedErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("Unwrap() 应返回原始错误: 期望 %v, 得到 %v", originalErr, unwrapped)
	}
}

// TestWrapNilError 测试 Wrap() 在传入 nil 错误时的行为
func TestWrapNilError(t *testing.T) {
	wrappedErr := Wrap(nil, "安全消息")

	if wrappedErr != nil {
		t.Error("Wrap(nil, ...) 应返回 nil")
	}
}

// TestSafeMessageFunc_WithInternalError 测试 SafeMessage 函数对 InternalError 返回安全消息
func TestSafeMessageFunc_WithInternalError(t *testing.T) {
	err := NewInternal("内部详情: panic at disk.go:42", "系统错误，请联系管理员")
	defaultMsg := "未知错误"

	got := SafeMsg(err, defaultMsg)
	expected := "系统错误，请联系管理员"

	if got != expected {
		t.Errorf("SafeMsg 对 InternalError 应返回安全消息: 期望 %q, 得到 %q", expected, got)
	}
}

// TestSafeMessageFunc_WithRegularError 测试 SafeMessage 函数对普通错误返回默认消息
func TestSafeMessageFunc_WithRegularError(t *testing.T) {
	err := fmt.Errorf("一个普通的错误")
	defaultMsg := "服务器内部错误"

	got := SafeMsg(err, defaultMsg)

	if got != defaultMsg {
		t.Errorf("SafeMsg 对普通错误应返回默认消息: 期望 %q, 得到 %q", defaultMsg, got)
	}
}

// TestSafeMessageFunc_WithNilError 测试 SafeMessage 函数对 nil 错误返回默认消息
func TestSafeMessageFunc_WithNilError(t *testing.T) {
	defaultMsg := "服务器内部错误"

	got := SafeMsg(nil, defaultMsg)

	if got != defaultMsg {
		t.Errorf("SafeMsg 对 nil 错误应返回默认消息: 期望 %q, 得到 %q", defaultMsg, got)
	}
}

// TestErrorInterface 测试 InternalError 实现了 error 接口
func TestErrorInterface(t *testing.T) {
	var err error = NewInternal("内部错误", "外部错误")

	if err == nil {
		t.Fatal("InternalError 应实现 error 接口")
	}

	// 确保可以通过 error 接口调用 Error()
	if got := err.Error(); got != "内部错误" {
		t.Errorf("通过 error 接口调用 Error() 应返回内部消息: 期望 %q, 得到 %q", "内部错误", got)
	}
}

// TestErrorsIs 测试 errors.Is 能通过 Unwrap 链找到原始错误
func TestErrorsIs(t *testing.T) {
	originalErr := fmt.Errorf("原始错误")
	wrappedErr := Wrap(originalErr, "安全消息")

	if !errors.Is(wrappedErr, originalErr) {
		t.Error("errors.Is 应能通过 Unwrap 链找到原始错误")
	}
}

// TestErrorsAs 测试 errors.As 能将错误转换为 InternalError 类型
func TestErrorsAs(t *testing.T) {
	err := NewInternal("内部详情", "对外消息")

	var ie *InternalError
	if !errors.As(err, &ie) {
		t.Error("errors.As 应能将错误转换为 *InternalError 类型")
	}

	if ie.SafeMessage() != "对外消息" {
		t.Errorf("转换后 SafeMessage() 期望 %q, 得到 %q", "对外消息", ie.SafeMessage())
	}
}

// TestUnwrap_WithoutWrappedError 测试没有包装错误时 Unwrap 返回 nil
func TestUnwrap_WithoutWrappedError(t *testing.T) {
	err := NewInternal("内部错误", "外部错误")

	if unwrapped := err.Unwrap(); unwrapped != nil {
		t.Errorf("未包装错误时 Unwrap() 应返回 nil, 得到 %v", unwrapped)
	}
}
