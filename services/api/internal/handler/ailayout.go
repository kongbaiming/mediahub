package handler

import (
	"context"
	"io"

	"github.com/mediahub/api/internal/ailayout"
	"github.com/mediahub/api/internal/apperr"

	"github.com/gin-gonic/gin"
)

// LibraryStatsProvider 媒资库统计提供者接口
type LibraryStatsProvider interface {
	GetLibraryStats(ctx context.Context) (*ailayout.LibraryStats, error)
}

// AiLayoutHandler AI 布局生成 HTTP handler
type AiLayoutHandler struct {
	svc      *ailayout.Service
	provider LibraryStatsProvider
}

// NewAiLayoutHandler 构造
func NewAiLayoutHandler(svc *ailayout.Service, provider LibraryStatsProvider) *AiLayoutHandler {
	return &AiLayoutHandler{svc: svc, provider: provider}
}

// Generate 从文字描述生成布局
func (h *AiLayoutHandler) Generate(c *gin.Context) {
	var req ailayout.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	ctx := c.Request.Context()

	// 获取媒资库上下文
	stats, err := h.provider.GetLibraryStats(ctx)
	if err != nil {
		respondError(c, apperr.Internal("获取媒资库统计失败"))
		return
	}

	config, explanation, err := h.svc.GenerateFromText(ctx, req.Prompt, stats)
	if err != nil {
		respondError(c, apperr.ExternalAPI(err, "AI 生成失败"))
		return
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"config":      config,
			"explanation": explanation,
		},
	})
}

// GenerateFromImage 从图片生成布局
func (h *AiLayoutHandler) GenerateFromImage(c *gin.Context) {
	// 读取上传的图片（限制 10MB + 1 字节，超出即拒绝）
	file, _, err := c.Request.FormFile("image")
	if err != nil {
		respondError(c, apperr.Validation("请上传图片文件"))
		return
	}
	defer file.Close()

	const maxSize = 10 * 1024 * 1024 // 10MB
	imageData, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		respondError(c, apperr.Internal("读取图片失败"))
		return
	}
	if len(imageData) > maxSize {
		respondError(c, apperr.Validation("图片大小不能超过 10MB"))
		return
	}

	ctx := c.Request.Context()

	stats, err := h.provider.GetLibraryStats(ctx)
	if err != nil {
		respondError(c, apperr.Internal("获取媒资库统计失败"))
		return
	}

	config, explanation, err := h.svc.GenerateFromImage(ctx, imageData, stats)
	if err != nil {
		respondError(c, apperr.ExternalAPI(err, "AI 图片识别失败"))
		return
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"config":      config,
			"explanation": explanation,
		},
	})
}
