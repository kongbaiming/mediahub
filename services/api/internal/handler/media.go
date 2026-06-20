package handler

import (
	"strconv"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// MediaHandler 媒资 HTTP handler
type MediaHandler struct {
	svc *service.MediaService
}

// NewMediaHandler 构造
func NewMediaHandler(svc *service.MediaService) *MediaHandler {
	return &MediaHandler{svc: svc}
}

// List 列表
func (h *MediaHandler) List(c *gin.Context) {
	p := common.Pagination{
		Page:     atoi(c.Query("page"), 1),
		PageSize: atoi(c.Query("page_size"), 20),
	}

	f := repository.MediaFilter{
		Type:         c.Query("type"),
		Genre:        c.Query("genre"),
		Search:       c.Query("q"),
		Sort:         c.Query("sort"),
		SortDesc:     c.Query("order") != "asc",
		ScrapeStatus: c.Query("scrape_status"),
	}
	if v := c.Query("year"); v != "" {
		if y, err := strconv.Atoi(v); err == nil {
			f.Year = &y
		}
	}
	if v := c.Query("min_rating"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinRating = &r
		}
	}

	result, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, result)
}

// Get 详情
func (h *MediaHandler) Get(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

// Create 手动创建
func (h *MediaHandler) Create(c *gin.Context) {
	var req struct {
		Type          common.MediaType `json:"type" binding:"required"`
		Title         string           `json:"title" binding:"required"`
		OriginalTitle string           `json:"original_title"`
		Year          *int             `json:"year"`
		StoragePath   string           `json:"storage_path" binding:"required"`
		Overview      string           `json:"overview"`
		PosterURL     string           `json:"poster_url"`
		BackdropURL   string           `json:"backdrop_url"`
		Genres        []string         `json:"genres"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	m := &media.Media{
		Type:          req.Type,
		Title:         req.Title,
		OriginalTitle: req.OriginalTitle,
		Year:          req.Year,
		StoragePath:   req.StoragePath,
		Overview:      req.Overview,
		PosterURL:     req.PosterURL,
		BackdropURL:   req.BackdropURL,
		Genres:        req.Genres,
		ScrapeStatus:  common.ScrapeStatusDone, // 手动入库视为已刮削
	}
	if err := h.svc.CreateManual(c.Request.Context(), m); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": m})
}

// Update 更新
func (h *MediaHandler) Update(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	if v, ok := req["title"].(string); ok {
		existing.Title = v
	}
	if v, ok := req["original_title"].(string); ok {
		existing.OriginalTitle = v
	}
	if v, ok := req["overview"].(string); ok {
		existing.Overview = v
	}
	if v, ok := req["poster_url"].(string); ok {
		existing.PosterURL = v
	}
	if v, ok := req["backdrop_url"].(string); ok {
		existing.BackdropURL = v
	}
	if v, ok := req["year"].(float64); ok {
		y := int(v)
		existing.Year = &y
	}
	if v, ok := req["rating"].(float64); ok {
		existing.Rating = v
	}
	if v, ok := req["genres"].([]any); ok {
		gs := make(media.StringArray, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				gs = append(gs, s)
			}
		}
		existing.Genres = gs
	}
	if existing.Genres == nil {
		existing.Genres = media.StringArray{}
	}
	if existing.Tags == nil {
		existing.Tags = media.StringArray{}
	}

	if err := h.svc.Update(c.Request.Context(), existing.Media); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": existing})
}

// Delete 删除
func (h *MediaHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

// Rescan 重新刮削
func (h *MediaHandler) Rescan(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Rescan(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(202, gin.H{"status": "queued", "media_id": id})
}

// Stats 统计
func (h *MediaHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": stats})
}

// Search 关键字搜索（为 TV / Android 客户端返回简洁的扁平结果）
func (h *MediaHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(200, gin.H{"data": []any{}})
		return
	}

	typeFilter := c.Query("type")
	limit := atoi(c.Query("limit"), 30)
	if limit > 100 {
		limit = 100
	}

	// 直接复用 List 服务（page_size=limit，只取 items）
	f := repository.MediaFilter{
		Type:   typeFilter,
		Search: q,
	}
	p := common.Pagination{Page: 1, PageSize: limit}

	result, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		respondError(c, err)
		return
	}

	// 转换为轻量级输出（去掉 is_adult 等敏感字段，方便客户端直接用）
	out := make([]gin.H, 0, len(result.Items))
	for _, m := range result.Items {
		out = append(out, gin.H{
			"media_id":     m.ID,
			"title":        m.Title,
			"year":         m.Year,
			"poster_url":   m.PosterURL,
			"backdrop_url": m.BackdropURL,
			"rating":       m.Rating,
			"type":         string(m.Type),
			"genres":       m.Genres,
		})
	}

	c.JSON(200, gin.H{
		"data":  out,
		"total": result.Total,
		"q":     q,
	})
}
