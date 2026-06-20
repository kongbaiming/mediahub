package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/downloader"
	"github.com/mediahub/api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// DownloaderHandler 下载 HTTP handler
type DownloaderHandler struct {
	svc *downloader.Service
}

// NewDownloaderHandler 构造
func NewDownloaderHandler(svc *downloader.Service) *DownloaderHandler {
	return &DownloaderHandler{svc: svc}
}

// Add 添加下载任务
func (h *DownloaderHandler) Add(c *gin.Context) {
	var req downloader.AddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	hash, err := h.svc.Add(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{
		"status": "added",
		"hash":   hash,
	})
}

// List 列出下载任务
//
// 降级策略：qBittorrent 不可用时返回空列表 + status=unavailable，
// 而不是 500。这样 admin 下载管理页能渲染"暂无可用下载器"空状态，
// 不会让用户以为整个 CMS 挂了。家庭场景下 qBit 挂了不应该阻塞主流程。
func (h *DownloaderHandler) List(c *gin.Context) {
	category := c.Query("category")
	items, err := h.svc.List(c.Request.Context(), category)
	if err != nil {
		logger.Warn("downloader list 降级返回空列表", "err", err, "category", category)
		c.JSON(200, gin.H{
			"data":    []any{},
			"total":   0,
			"status":  "unavailable",
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"data":   items,
		"total":  len(items),
		"status": "ok",
	})
}

// Remove 删除任务
func (h *DownloaderHandler) Remove(c *gin.Context) {
	hash := c.Param("hash")
	deleteFiles := c.Query("delete_files") == "true"
	if err := h.svc.Remove(c.Request.Context(), hash, deleteFiles); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "removed"})
}

// Pause 暂停
func (h *DownloaderHandler) Pause(c *gin.Context) {
	hash := c.Param("hash")
	if err := h.svc.Pause(c.Request.Context(), hash); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "paused"})
}

// Resume 恢复
func (h *DownloaderHandler) Resume(c *gin.Context) {
	hash := c.Param("hash")
	if err := h.svc.Resume(c.Request.Context(), hash); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "resumed"})
}

// CheckCompleted 手动触发检查完成入库
func (h *DownloaderHandler) CheckCompleted(c *gin.Context) {
	n, err := h.svc.CheckCompleted(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"status":   "checked",
		"imported": n,
	})
}

// Health 健康检查
func (h *DownloaderHandler) Health(c *gin.Context) {
	if err := h.svc.Health(c.Request.Context()); err != nil {
		c.JSON(503, gin.H{
			"status": "down",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}
