package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/indexer"
	"github.com/mediahub/api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// IndexerHandler 索引搜索 HTTP handler（CMS 入库用）
type IndexerHandler struct {
	svc *indexer.Service
}

// NewIndexerHandler 构造
func NewIndexerHandler(svc *indexer.Service) *IndexerHandler {
	return &IndexerHandler{svc: svc}
}

type releaseDTO struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Size        int64  `json:"size"`
	Seeders     int    `json:"seeders"`
	Peers       int    `json:"peers"`
	Indexer     string `json:"indexer"`
	PublishDate string `json:"publish_date,omitempty"`
}

func toReleaseDTOs(releases []indexer.Release) []releaseDTO {
	out := make([]releaseDTO, 0, len(releases))
	for _, r := range releases {
		out = append(out, releaseDTO{
			Title:       r.Title,
			Link:        r.Link(),
			Size:        r.Size,
			Seeders:     r.Seeders,
			Peers:       r.Peers,
			Indexer:     r.Indexer,
			PublishDate: r.PublishDate,
		})
	}
	return out
}

// Search 搜索种子资源
func (h *IndexerHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		respondError(c, apperr.Validation("q 不能为空"))
		return
	}
	if h.svc == nil || !h.svc.Enabled() {
		c.JSON(200, gin.H{
			"data":    []releaseDTO{},
			"status":  "unavailable",
			"message": "索引器未配置，请设置 INDEXER_URL 与 INDEXER_API_KEY（Prowlarr）",
		})
		return
	}

	limit := atoi(c.Query("limit"), 20)
	mediaType := c.Query("type")
	releases, err := h.svc.Search(c.Request.Context(), q, mediaType, limit)
	if err != nil {
		logger.Warn("indexer search failed", "q", q, "err", err)
		c.JSON(200, gin.H{
			"data":    []releaseDTO{},
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"data":   toReleaseDTOs(releases),
		"status": "ok",
		"query":  q,
	})
}
