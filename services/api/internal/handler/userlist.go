package handler

import (
	"strconv"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserListHandler 用户片单 HTTP handler
type UserListHandler struct {
	svc *service.UserListService
}

// NewUserListHandler 构造
func NewUserListHandler(svc *service.UserListService) *UserListHandler {
	return &UserListHandler{svc: svc}
}

// List 列出当前 Profile 的片单
func (h *UserListHandler) List(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	items, err := h.svc.List(c.Request.Context(), profileID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// Get 获取片单详情（含条目）
func (h *UserListHandler) Get(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, apperr.Validation("invalid id"))
		return
	}
	item, err := h.svc.Get(c.Request.Context(), profileID, id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": item})
}

// Create 创建片单
func (h *UserListHandler) Create(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	item, err := h.svc.Create(c.Request.Context(), profileID, req.Name, req.Description, req.IsPublic)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": item})
}

// Update 更新片单
func (h *UserListHandler) Update(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, apperr.Validation("invalid id"))
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		IsPublic    *bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	if err := h.svc.Update(c.Request.Context(), profileID, id, req.Name, req.Description, req.IsPublic); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

// Delete 删除片单
func (h *UserListHandler) Delete(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, apperr.Validation("invalid id"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), profileID, id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

// AddItem 添加媒资到片单
func (h *UserListHandler) AddItem(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, apperr.Validation("invalid id"))
		return
	}
	var req struct {
		MediaID string `json:"media_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	if err := h.svc.AddItem(c.Request.Context(), profileID, id, req.MediaID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "added"})
}

// RemoveItem 从片单移除媒资
func (h *UserListHandler) RemoveItem(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, apperr.Validation("invalid id"))
		return
	}
	mediaID := c.Param("media_id")
	if err := h.svc.RemoveItem(c.Request.Context(), profileID, id, mediaID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "removed"})
}
