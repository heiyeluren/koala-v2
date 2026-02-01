// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供Koala反作弊频率控制系统的HTTP API实现。
//
// Koala是一个高性能的限流系统，支持多种限流策略和访问控制。
package api

import (
	"context"
)

// ========== 引擎接口定义 ==========

// Engine 定义限流引擎的接口。
// 所有限流检查和更新操作都通过此接口进行。
type Engine interface {
	// Browse 检查请求是否允许通过。
	// 返回检查结果，包括是否允许、匹配的规则名称等信息。
	Browse(ctx context.Context, req *EngineRequest) (*EngineResponse, error)

	// Update 更新计数器。
	// 在请求被允许执行后调用，用于递增限流计数器。
	Update(ctx context.Context, req *EngineRequest) error
}

// EngineRequest 表示引擎请求参数。
type EngineRequest struct {
	Act    string            // 行为/动作类型
	UID    string            // 用户ID
	IP     string            // 客户端IP地址
	DID    string            // 设备ID
	Ext    map[string]string // 扩展字段
	Update bool              // 是否在Browse后自动更新计数器
}

// EngineResponse 表示引擎响应。
type EngineResponse struct {
	Allowed  bool   // 是否允许通过
	RuleName string // 匹配的规则名称
	Code     int    // 结果代码
	Message  string // 结果消息
	AuthType int    // 认证类型（0=无需认证, 1=需要验证码等）
}

// ========== API请求/响应类型 ==========

// APIRequest 表示Browse/Update API请求体。
type APIRequest struct {
	Act    string            `json:"act" binding:"required"`    // 行为/动作类型（必填）
	UID    string            `json:"uid"`                       // 用户ID
	IP     string            `json:"ip"`                        // 客户端IP地址
	DID    string            `json:"did"`                       // 设备ID
	Ext    map[string]string `json:"ext"`                       // 扩展字段
	Update bool              `json:"update"`                    // Browse时是否自动更新计数器
}

// APIResponse 表示API响应体。
type APIResponse struct {
	Allowed  bool   `json:"allowed"`             // 是否允许通过
	Code     int    `json:"code"`                // 结果代码（0=成功）
	Message  string `json:"message"`             // 结果消息
	RuleName string `json:"rule_name,omitempty"` // 匹配的规则名称
	AuthType int    `json:"auth_type,omitempty"` // 认证类型
}

// BatchRequest 表示批量检查请求体。
type BatchRequest struct {
	Requests []BatchItem `json:"requests" binding:"required,min=1,max=100"` // 请求列表
}

// BatchItem 表示批量请求中的单个请求项。
type BatchItem struct {
	ID  string            `json:"id" binding:"required"`  // 请求ID（用于关联响应）
	Act string            `json:"act" binding:"required"` // 行为/动作类型
	UID string            `json:"uid"`                    // 用户ID
	IP  string            `json:"ip"`                     // 客户端IP地址
	DID string            `json:"did"`                    // 设备ID
	Ext map[string]string `json:"ext"`                    // 扩展字段
}

// BatchResponse 表示批量检查响应体。
type BatchResponse struct {
	Results []BatchResult `json:"results"` // 结果列表
}

// BatchResult 表示批量响应中的单个结果。
type BatchResult struct {
	ID       string `json:"id"`                  // 请求ID
	Allowed  bool   `json:"allowed"`             // 是否允许通过
	Code     int    `json:"code"`                // 结果代码
	Message  string `json:"message"`             // 结果消息
	RuleName string `json:"rule_name,omitempty"` // 匹配的规则名称
	AuthType int    `json:"auth_type,omitempty"` // 认证类型
}

// ========== 健康检查类型 ==========

// HealthResponse 表示健康检查响应。
type HealthResponse struct {
	Status    string `json:"status"`    // 健康状态（ok/error）
	Timestamp string `json:"timestamp"` // 时间戳
}

// ReadyResponse 表示就绪检查响应。
type ReadyResponse struct {
	Ready     bool   `json:"ready"`               // 是否就绪
	Message   string `json:"message,omitempty"`   // 消息
	Timestamp string `json:"timestamp,omitempty"` // 时间戳
}
