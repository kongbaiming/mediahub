package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/recommend"

	"github.com/gin-gonic/gin"
)

// RecommendHandler 推荐 HTTP handler
type RecommendHandler struct {
	svc *recommend.Service
}

// NewRecommendHandler 构造
func NewRecommendHandler(svc *recommend.Service) *RecommendHandler {
	return &RecommendHandler{svc: svc}
}

// Hot 全局热门
func (h *RecommendHandler) Hot(c *gin.Context) {
	limit := atoi(c.Query("limit"), 20)
	items, err := h.svc.Hot(c.Request.Context(), limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// ForProfile 个人推荐
func (h *RecommendHandler) ForProfile(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		respondError(c, apperr.BadRequest("缺少 profile_id"))
		return
	}
	limit := atoi(c.Query("limit"), 20)
	items, err := h.svc.ForProfile(c.Request.Context(), profileID, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// SimilarTo 相似媒资
func (h *RecommendHandler) SimilarTo(c *gin.Context) {
	id := c.Param("id")
	limit := atoi(c.Query("limit"), 20)
	items, err := h.svc.SimilarTo(c.Request.Context(), id, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}
