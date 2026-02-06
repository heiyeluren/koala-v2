// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package api 提供Koala反作弊频率控制系统的HTTP Handler实现。
package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"koala/pkg/logger"
)

// Handler 处理所有HTTP请求。
type Handler struct {
	engine Engine // 限流引擎
}

// NewHandler 创建新的Handler实例。
func NewHandler(engine Engine) *Handler {
	return &Handler{
		engine: engine,
	}
}

// Browse 处理频率检查请求。
// POST /api/v1/browse
// 检查指定行为是否被限流，可选地自动更新计数器。
func (h *Handler) Browse(c *gin.Context) {
	var req APIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Browse请求参数绑定失败", "error", err)
		c.JSON(http.StatusBadRequest, APIResponse{
			Allowed: false,
			Code:    -1,
			Message: "请求参数错误",
		})
		return
	}

	// 构造引擎请求
	engineReq := &EngineRequest{
		Act:    req.Act,
		UID:    req.UID,
		IP:     req.IP,
		DID:    req.DID,
		Ext:    req.Ext,
		Update: req.Update,
	}

	// 调用引擎进行检查
	resp, err := h.engine.Browse(c.Request.Context(), engineReq)
	if err != nil {
		logger.Error("Browse引擎处理失败", "error", err, "act", req.Act, "uid", req.UID)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Allowed: false,
			Code:    -2,
			Message: "服务器内部错误",
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, APIResponse{
		Allowed:  resp.Allowed,
		Code:     resp.Code,
		Message:  resp.Message,
		RuleName: resp.RuleName,
		AuthType: resp.AuthType,
	})
}

// Update 处理计数器更新请求。
// POST /api/v1/update
// 在行为执行成功后更新限流计数器。
func (h *Handler) Update(c *gin.Context) {
	var req APIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Update请求参数绑定失败", "error", err)
		c.JSON(http.StatusBadRequest, APIResponse{
			Allowed: false,
			Code:    -1,
			Message: "请求参数错误",
		})
		return
	}

	// 构造引擎请求
	engineReq := &EngineRequest{
		Act: req.Act,
		UID: req.UID,
		IP:  req.IP,
		DID: req.DID,
		Ext: req.Ext,
	}

	// 调用引擎更新计数器
	err := h.engine.Update(c.Request.Context(), engineReq)
	if err != nil {
		logger.Error("Update引擎处理失败", "error", err, "act", req.Act, "uid", req.UID)
		c.JSON(http.StatusInternalServerError, APIResponse{
			Allowed: false,
			Code:    -2,
			Message: "服务器内部错误",
		})
		return
	}

	// 返回成功
	c.JSON(http.StatusOK, APIResponse{
		Allowed: true,
		Code:    0,
		Message: "ok",
	})
}

// Batch 处理批量检查请求。
// POST /api/v1/batch
// 批量检查多个请求是否被限流。
func (h *Handler) Batch(c *gin.Context) {
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Batch请求参数绑定失败", "error", err)
		c.JSON(http.StatusBadRequest, APIResponse{
			Allowed: false,
			Code:    -1,
			Message: "请求参数错误",
		})
		return
	}

	// 验证请求数量
	if len(req.Requests) == 0 {
		c.JSON(http.StatusBadRequest, APIResponse{
			Allowed: false,
			Code:    -1,
			Message: "请求列表不能为空",
		})
		return
	}
	if len(req.Requests) > 100 {
		c.JSON(http.StatusBadRequest, APIResponse{
			Allowed: false,
			Code:    -1,
			Message: "请求数量超过限制（最大100）",
		})
		return
	}

	// 并行处理每个请求
	results := make([]BatchResult, len(req.Requests))
	var wg sync.WaitGroup
	for i, item := range req.Requests {
		// 验证请求项（在主协程中完成，避免竞态）
		if item.ID == "" || item.Act == "" {
			results[i] = BatchResult{
				ID:      item.ID,
				Allowed: false,
				Code:    -1,
				Message: "请求ID和Act不能为空",
			}
			continue
		}

		wg.Add(1)
		go func(idx int, batchItem BatchItem) {
			defer wg.Done()

			engineReq := &EngineRequest{
				Act: batchItem.Act,
				UID: batchItem.UID,
				IP:  batchItem.IP,
				DID: batchItem.DID,
				Ext: batchItem.Ext,
			}

			resp, err := h.engine.Browse(c.Request.Context(), engineReq)
			if err != nil {
				logger.Error("Batch引擎处理失败", "error", err, "act", batchItem.Act, "uid", batchItem.UID, "item_id", batchItem.ID)
				results[idx] = BatchResult{
					ID:      batchItem.ID,
					Allowed: false,
					Code:    -2,
					Message: "服务器内部错误",
				}
				return
			}

			results[idx] = BatchResult{
				ID:       batchItem.ID,
				Allowed:  resp.Allowed,
				Code:     resp.Code,
				Message:  resp.Message,
				RuleName: resp.RuleName,
				AuthType: resp.AuthType,
			}
		}(i, item)
	}
	wg.Wait()

	c.JSON(http.StatusOK, BatchResponse{
		Results: results,
	})
}

// Health 处理健康检查请求。
// GET /health
// 返回服务健康状态。
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready 处理就绪检查请求。
// GET /ready
// 返回服务就绪状态，检查依赖组件是否可用。
func (h *Handler) Ready(c *gin.Context) {
	// 如果引擎不存在，仍然返回就绪（用于测试场景）
	// 在生产环境中，可以添加更多检查逻辑
	c.JSON(http.StatusOK, ReadyResponse{
		Ready:     true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
