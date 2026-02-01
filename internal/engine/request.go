// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package engine 提供Koala反作弊频率控制系统的核心规则引擎。
//
// 引擎负责加载规则配置、执行规则匹配和限流检查。
// 支持两阶段规则执行：
//   - 阶段1：访问控制（白名单 -> 黑名单）
//   - 阶段2：限流规则（业务 -> 发帖 -> 高级 -> 默认）
package engine

// Request 表示限流检查请求。
// 包含请求的所有上下文信息，用于规则匹配。
type Request struct {
	Act string            // 行为类型（如 login, register, post）
	UID string            // 用户ID
	IP  string            // 客户端IP地址
	DID string            // 设备ID
	Ext map[string]string // 扩展参数（用于自定义匹配条件）
}

// GetField 获取请求中的字段值。
// 支持标准字段（act, uid, ip, did）和扩展字段。
func (r *Request) GetField(field string) string {
	switch field {
	case "act":
		return r.Act
	case "uid":
		return r.UID
	case "ip":
		return r.IP
	case "did":
		return r.DID
	default:
		if r.Ext != nil {
			return r.Ext[field]
		}
		return ""
	}
}

// Response 表示限流检查响应。
type Response struct {
	Allowed  bool   // 是否允许请求
	Code     int    // 响应码（0=允许，其他=拒绝原因）
	Message  string // 响应消息
	RuleName string // 命中的规则名称（如果有）
	AuthType int    // 验证类型（0=无验证，1=滑块验证，2=短信验证等）
}

// NewAllowedResponse 创建一个允许的响应。
func NewAllowedResponse() *Response {
	return &Response{
		Allowed:  true,
		Code:     0,
		Message:  "ok",
		RuleName: "",
		AuthType: 0,
	}
}

// NewDeniedResponse 创建一个拒绝的响应。
func NewDeniedResponse(code int, message, ruleName string, authType int) *Response {
	return &Response{
		Allowed:  false,
		Code:     code,
		Message:  message,
		RuleName: ruleName,
		AuthType: authType,
	}
}
