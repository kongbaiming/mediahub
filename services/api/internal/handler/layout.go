package handler

import (
	"encoding/json"
	"io"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// LayoutHandler 布局 HTTP handler
type LayoutHandler struct {
	svc     *service.LayoutService
	feedSvc *service.FeedService
}

// NewLayoutHandler 构造
func NewLayoutHandler(svc *service.LayoutService, feedSvc *service.FeedService) *LayoutHandler {
	return &LayoutHandler{svc: svc, feedSvc: feedSvc}
}

// List 列表
func (h *LayoutHandler) List(c *gin.Context) {
	var isTemplate *bool
	if v := c.Query("is_template"); v != "" {
		b := v == "true" || v == "1"
		isTemplate = &b
	}
	status := c.Query("status")

	items, err := h.svc.List(c.Request.Context(), isTemplate, status)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"data":  items,
		"total": len(items),
		"page":  1,
		"size":  len(items),
	})
}

// Get 详情
func (h *LayoutHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var (
		l   any
		err error
	)
	if c.Query("editor") == "1" {
		l, err = h.svc.GetForEditor(c.Request.Context(), id)
	} else {
		l, err = h.svc.Get(c.Request.Context(), id)
	}
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": l})
}

// Create 创建
func (h *LayoutHandler) Create(c *gin.Context) {
	var req service.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	l, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": l})
}

// Update 更新
func (h *LayoutHandler) Update(c *gin.Context) {
	id := c.Param("id")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, apperr.Validation("无法读取请求体"))
		return
	}

	var req service.UpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		if _, ok := raw["parent_id"]; ok {
			req.ParentIDSet = true
		}
	}

	l, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": l})
}

// Delete 删除
func (h *LayoutHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

// Publish 发布
func (h *LayoutHandler) Publish(c *gin.Context) {
	id := c.Param("id")
	var req service.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	// 校验 platform
	switch req.TargetPlatform {
	case common.PlatformWeb, common.PlatformAndroidTV, common.PlatformTVOS:
	default:
		respondError(c, apperr.Validation(map[string]string{
			"target_platform": "必须是 web / android-tv / tvos",
		}))
		return
	}
	l, err := h.svc.Publish(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": l})
}

// ListPublications 列出布局的所有发布
func (h *LayoutHandler) ListPublications(c *gin.Context) {
	id := c.Param("id")
	pubs, err := h.svc.ListPublications(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": pubs, "total": len(pubs)})
}

// Preview 编辑器预览 Feed（合并继承 + 真实媒资数据）
func (h *LayoutHandler) Preview(c *gin.Context) {
	id := c.Param("id")
	platform := c.Query("platform")
	if platform == "" {
		platform = "web"
	}
	switch common.Platform(platform) {
	case common.PlatformWeb, common.PlatformAndroidTV, common.PlatformTVOS:
	default:
		respondError(c, apperr.Validation(map[string]string{
			"platform": "必须是 web / android-tv / tvos",
		}))
		return
	}

	merged, err := h.svc.GetForEditor(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		profileID = "anonymous"
	}

	feed, err := h.feedSvc.BuildFeedFromLayout(c.Request.Context(), merged, platform, profileID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": feed})
}

// DisablePublication 禁用某个发布
func (h *LayoutHandler) DisablePublication(c *gin.Context) {
	id := c.Param("pub_id")
	if err := h.svc.DisablePublication(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "disabled"})
}
