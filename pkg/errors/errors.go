// Package errors 提供区分内部错误信息和对外安全信息的错误类型。
//
// 在生产系统中，内部错误详情（如数据库连接字符串、SQL语句等）不应暴露给客户端。
// InternalError 类型将内部调试信息与对外安全消息分离，
// 确保日志中记录完整的错误上下文，同时只向客户端返回安全的提示信息。
package errors

import "errors"

// InternalError 区分内部错误信息和对外安全信息
type InternalError struct {
	internal string // 内部详细错误信息（用于日志记录）
	external string // 对外安全信息（返回给客户端）
	wrapped  error  // 被包装的原始错误（支持错误链）
}

// NewInternal 创建新的内部错误
func NewInternal(internal, external string) *InternalError {
	return &InternalError{
		internal: internal,
		external: external,
	}
}

// Error 返回内部详细错误信息（实现 error 接口）
func (e *InternalError) Error() string {
	return e.internal
}

// SafeMessage 返回可以安全暴露给客户端的消息
func (e *InternalError) SafeMessage() string {
	return e.external
}

// Unwrap 返回被包装的原始错误，支持 errors.Is/As 错误链查找
func (e *InternalError) Unwrap() error {
	return e.wrapped
}

// Wrap 包装一个已有错误，添加安全消息。
// 原始错误的 Error() 信息作为内部消息保留，safeMessage 作为对外安全消息。
// 如果 err 为 nil，返回 nil。
func Wrap(err error, safeMessage string) *InternalError {
	if err == nil {
		return nil
	}
	return &InternalError{
		internal: err.Error(),
		external: safeMessage,
		wrapped:  err,
	}
}

// SafeMsg 检查错误是否是 InternalError 类型。
// 如果是，返回安全消息；否则返回默认消息。
// 适用于在 HTTP 响应中安全地返回错误信息。
func SafeMsg(err error, defaultMsg string) string {
	if err == nil {
		return defaultMsg
	}
	var ie *InternalError
	if errors.As(err, &ie) {
		return ie.SafeMessage()
	}
	return defaultMsg
}
