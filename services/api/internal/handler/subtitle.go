package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/subtitle"

	"github.com/gin-gonic/gin"
)

// SubtitleHandler 字幕 HTTP handler
type SubtitleHandler struct {
	svc *subtitle.Service
}

// NewSubtitleHandler 构造
func NewSubtitleHandler(svc *subtitle.Service) *SubtitleHandler {
	return &SubtitleHandler{svc: svc}
}

// Search 搜索字幕
func (h *SubtitleHandler) Search(c *gin.Context) {
	var req subtitle.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	subs, err := h.svc.Search(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"data":  subs,
		"total": len(subs),
	})
}

// Download 下载字幕
func (h *SubtitleHandler) Download(c *gin.Context) {
	mediaID := c.Param("id")
	var req subtitle.DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	if err := h.svc.Download(c.Request.Context(), mediaID, req.Subtitle); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "downloaded"})
}
