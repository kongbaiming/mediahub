package handler

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LiveHandler 直播间 HTTP handler
type LiveHandler struct {
	svc *service.LiveService
}

// NewLiveHandler 构造
func NewLiveHandler(svc *service.LiveService) *LiveHandler {
	return &LiveHandler{svc: svc}
}

// List 直播间列表（播放端 + CMS）
func (h *LiveHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	items, total, err := h.svc.List(c.Request.Context(), status, page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"data":  items,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// Get 直播间详情
func (h *LiveHandler) Get(c *gin.Context) {
	id := c.Param("id")
	room, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": room})
}

// Create 创建直播间（CMS）
func (h *LiveHandler) Create(c *gin.Context) {
	var req service.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		respondError(c, apperr.Unauthorized("未登录"))
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondError(c, apperr.Unauthorized("无效用户"))
		return
	}
	room, err := h.svc.Create(c.Request.Context(), req, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": room})
}

// Update 更新直播间（CMS）
func (h *LiveHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req service.UpdateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	room, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": room})
}

// Delete 删除直播间（CMS）
func (h *LiveHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

// Stop 结束直播（CMS 手动标记）
func (h *LiveHandler) Stop(c *gin.Context) {
	id := c.Param("id")
	room, err := h.svc.Stop(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": room})
}

// PublishHook MediaMTX 推流开始 webhook
func (h *LiveHandler) PublishHook(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = c.PostForm("path")
	}
	if err := h.svc.OnPublish(c.Request.Context(), path); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

// UnpublishHook MediaMTX 推流结束 webhook
func (h *LiveHandler) UnpublishHook(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = c.PostForm("path")
	}
	if err := h.svc.OnUnpublish(c.Request.Context(), path); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

// ProxyPlaylist 反代 MediaMTX HLS playlist（播放端使用）
func (h *LiveHandler) ProxyPlaylist(c *gin.Context) {
	id := c.Param("id")
	room, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	srcURL := h.svc.HLSPlaylistURL(room.StreamKey)
	resp, err := http.Get(srcURL)
	if err != nil {
		c.JSON(502, gin.H{"error": "upstream_error", "message": "无法连接推流服务"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := "推流尚未就绪"
		if strings.Contains(string(body), "authentication") {
			msg = "推流服务鉴权失败，请检查 mediamtx.yml 中 authInternalUsers 配置"
		}
		c.JSON(resp.StatusCode, gin.H{"error": "upstream_error", "message": msg, "detail": string(body)})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(502, gin.H{"error": "upstream_error", "message": "读取 playlist 失败"})
		return
	}

	// 重写 playlist 中的相对路径为 API 代理路径
	proxyBase := "/api/v1/live/rooms/" + id + "/"
	content := rewriteM3U8(string(body), proxyBase)

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "no-cache, no-store")
	c.String(200, content)
}

// ProxySegment 反代 HLS 切片
func (h *LiveHandler) ProxySegment(c *gin.Context) {
	id := c.Param("id")
	file := c.Param("file")
	room, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	srcURL := h.svc.HLSPlaylistURL(room.StreamKey)
	base := srcURL[:strings.LastIndex(srcURL, "/")+1]
	segURL := base + file

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, segURL, nil)
	if err != nil {
		c.Status(502)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Status(502)
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	if ct := resp.Header.Get("Content-Type"); ct == "" {
		if strings.HasSuffix(file, ".m4s") || strings.HasSuffix(file, ".mp4") {
			c.Header("Content-Type", "video/iso.segment")
		} else {
			c.Header("Content-Type", "video/mp2t")
		}
	}
	c.Header("Cache-Control", "no-cache")
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

func rewriteM3U8(content, proxyBase string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			if u, err := url.Parse(trimmed); err == nil {
				seg := u.Path[strings.LastIndex(u.Path, "/")+1:]
				lines[i] = proxyBase + seg
			}
		} else {
			lines[i] = proxyBase + trimmed
		}
	}
	return strings.Join(lines, "\n")
}
