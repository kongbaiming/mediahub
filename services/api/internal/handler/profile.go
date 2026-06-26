package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// ProfileHandler Profile HTTP handler
type ProfileHandler struct {
	svc *service.ProfileService
}

// NewProfileHandler 构造
func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

// List 列出当前用户的所有 Profile
func (h *ProfileHandler) List(c *gin.Context) {
	uid := middleware.GetUserID(c)
	items, err := h.svc.ListMyProfiles(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// Create 创建 Profile
func (h *ProfileHandler) Create(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req service.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	p, err := h.svc.Create(c.Request.Context(), uid, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": p})
}

// Update 更新 Profile
func (h *ProfileHandler) Update(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id := c.Param("id")
	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	p, err := h.svc.Update(c.Request.Context(), uid, id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": p})
}

// Delete 删除 Profile
func (h *ProfileHandler) Delete(c *gin.Context) {
	uid := middleware.GetUserID(c)
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), uid, id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

// ListWebPlayer 列出 Web 播放端 Profile（无需登录）
func (h *ProfileHandler) ListWebPlayer(c *gin.Context) {
	items, err := h.svc.ListForWebPlayer(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// CreateWebPlayer 创建 Web 播放端 Profile（无需登录）
func (h *ProfileHandler) CreateWebPlayer(c *gin.Context) {
	var req service.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	p, err := h.svc.CreateForWebPlayer(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": p})
}

// VerifyPin 验证 PIN（儿童 Profile 切换）
func (h *ProfileHandler) VerifyPin(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Pin string `json:"pin" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	if err := h.svc.VerifyPin(c.Request.Context(), id, req.Pin); err != nil {
		respondError(c, apperr.Unauthorized("PIN 错误"))
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}
