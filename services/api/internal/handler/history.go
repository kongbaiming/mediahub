package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// HistoryHandler 历史/收藏 HTTP handler
type HistoryHandler struct {
	svc *service.HistoryService
}

// NewHistoryHandler 构造
func NewHistoryHandler(svc *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{svc: svc}
}

// Record 记录播放进度
func (h *HistoryHandler) Record(c *gin.Context) {
	var req service.RecordProgress
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	// profile_id 从 header 取（优先），再从 body 取
	if req.ProfileID == "" {
		req.ProfileID = middleware.GetProfileID(c)
	}
	if req.ProfileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}

	if err := h.svc.RecordProgress(c.Request.Context(), req); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "recorded"})
}

// List 历史
func (h *HistoryHandler) List(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	limit := atoi(c.Query("limit"), 20)
	items, err := h.svc.GetHistory(c.Request.Context(), profileID, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// ContinueWatching 继续观看
func (h *HistoryHandler) ContinueWatching(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	limit := atoi(c.Query("limit"), 12)
	items, err := h.svc.GetContinueWatching(c.Request.Context(), profileID, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// ToggleFavorite 切换收藏
func (h *HistoryHandler) ToggleFavorite(c *gin.Context) {
	var req struct {
		MediaID string         `json:"media_id" binding:"required"`
		Type    common.FavoriteType `json:"type"`
		Rating  *float64       `json:"rating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}

	added, err := h.svc.ToggleFavorite(c.Request.Context(), profileID, req.MediaID, req.Type, req.Rating)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok", "added": added})
}

// ListFavorites 列出收藏
func (h *HistoryHandler) ListFavorites(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	favType := c.Query("type")
	items, err := h.svc.ListFavorites(c.Request.Context(), profileID, favType)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// GetResumePoint 获取续播位置
func (h *HistoryHandler) GetResumePoint(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	mediaID := c.Param("media_id")
	h_, err := h.svc.GetResumePoint(c.Request.Context(), profileID, mediaID)
	if err != nil {
		respondError(c, err)
		return
	}
	if h_ == nil {
		c.JSON(200, gin.H{"data": nil})
		return
	}
	c.JSON(200, gin.H{
		"data": gin.H{
			"progress":  h_.Progress,
			"duration":  h_.Duration,
			"completed": h_.Completed,
			"updated_at": h_.UpdatedAt,
		},
	})
}
