package handler

import (
	"context"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

const feedBuildTimeout = 25 * time.Second

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

	ctx, cancel := context.WithTimeout(c.Request.Context(), feedBuildTimeout)
	defer cancel()
	feed, err := h.svc.BuildFeed(ctx, platform, profileID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": feed})
}

// Version 返回 Feed 全局版本号（客户端轮询，v0.4 A6）
func (h *FeedHandler) Version(c *gin.Context) {
	c.JSON(200, gin.H{
		"version": h.svc.GetFeedVersion(c.Request.Context()),
	})
}
