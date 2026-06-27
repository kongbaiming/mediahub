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

// GuessYouLike 猜你喜欢
func (h *RecommendHandler) GuessYouLike(c *gin.Context) {
	profileID := middleware.GetProfileID(c)
	if profileID == "" {
		profileID = "anonymous"
	}
	limit := atoi(c.Query("limit"), 20)
	discover := atoi(c.Query("discover_limit"), 6)
	items, err := h.svc.GuessYouLike(c.Request.Context(), profileID, limit, discover)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}

// LibraryMissing CMS：猜你喜欢库外推荐（v0.4 B1）
func (h *RecommendHandler) LibraryMissing(c *gin.Context) {
	profileID := c.Query("profile_id")
	if profileID == "" {
		profileID = middleware.GetProfileID(c)
	}
	limit := atoi(c.Query("limit"), 30)
	discover := atoi(c.Query("discover_limit"), 12)
	items, err := h.svc.LibraryMissing(c.Request.Context(), profileID, limit, discover)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items, "total": len(items)})
}
