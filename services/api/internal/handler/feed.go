package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// FeedHandler Feed HTTP handler
type FeedHandler struct {
	svc *service.FeedService
}

// NewFeedHandler 构造
func NewFeedHandler(svc *service.FeedService) *FeedHandler {
	return &FeedHandler{svc: svc}
}

// Get 播放端拉取 Feed
func (h *FeedHandler) Get(c *gin.Context) {
	platform := c.Param("platform")
	// 校验 platform
	switch common.Platform(platform) {
	case common.PlatformWeb, common.PlatformAndroidTV, common.PlatformTVOS:
	default:
		respondError(c, apperr.Validation(map[string]string{
			"platform": "必须是 web / android-tv / tvos",
		}))
		return
	}

	// profile_id 从 header 取（家庭多 Profile）
	profileID := middleware.GetProfileID(c)
	// 没传就用 "anonymous"，继续观看数据源会自然为空
	if profileID == "" {
		profileID = "anonymous"
	}

	feed, err := h.svc.BuildFeed(c.Request.Context(), platform, profileID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": feed})
}
