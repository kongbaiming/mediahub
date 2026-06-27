package handler

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"
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

// ProxyPlaylist 反代 MediaMTX 主 HLS playlist
func (h *LiveHandler) ProxyPlaylist(c *gin.Context) {
	h.proxyMedia(c, "index.m3u8")
}

// ProxySegment 反代 HLS 子 playlist / 切片（保留 query，如 LL-HLS 的 session）
func (h *LiveHandler) ProxySegment(c *gin.Context) {
	h.proxyMedia(c, c.Param("file"))
}

// ProxyUpstream 反代 IPTV 上游资源（子 playlist / 切片）
func (h *LiveHandler) ProxyUpstream(c *gin.Context) {
	id := c.Param("id")
	rawURL := c.Query("u")
	if rawURL == "" {
		c.JSON(400, gin.H{"error": "missing_url", "message": "缺少上游地址"})
		return
	}
	targetURL, err := url.QueryUnescape(rawURL)
	if err != nil || targetURL == "" {
		c.JSON(400, gin.H{"error": "invalid_url", "message": "上游地址无效"})
		return
	}

	room, err := h.svc.GetRoomRaw(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if !room.IsIPTV() {
		c.JSON(404, gin.H{"error": "not_iptv", "message": "非 IPTV 直播间"})
		return
	}

	h.fetchAndProxyURL(c, room, targetURL)
}

func (h *LiveHandler) proxyMedia(c *gin.Context, file string) {
	id := c.Param("id")
	room, err := h.svc.GetRoomRaw(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	if room.IsIPTV() {
		if file != "index.m3u8" {
			c.JSON(404, gin.H{"error": "not_found", "message": "资源不存在"})
			return
		}
		if room.Status == live.StatusEnded {
			c.JSON(503, gin.H{"error": "stream_ended", "message": "直播已结束"})
			return
		}
		if room.SourceURL == "" {
			c.JSON(503, gin.H{"error": "no_source", "message": "未配置 IPTV 源地址"})
			return
		}
		h.fetchAndProxyURL(c, room, room.SourceURL)
		return
	}

	fileName := strings.SplitN(file, "?", 2)[0]
	if fileName == "index.m3u8" && !h.svc.IsPathOnline(c.Request.Context(), room.StreamKey) {
		c.JSON(503, gin.H{
			"error":   "not_streaming",
			"message": "主播尚未推流，请在 OBS 中点击「开始直播」",
		})
		return
	}

	upstream := h.svc.HLSMediaURL(room.StreamKey, file, c.Request.URL.RawQuery)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		c.JSON(502, gin.H{"error": "upstream_error", "message": "构建上游请求失败"})
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(502, gin.H{"error": "upstream_error", "message": "无法连接推流服务"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		msg := "推流尚未就绪"
		errCode := "upstream_error"
		switch {
		case resp.StatusCode == http.StatusUnauthorized || strings.Contains(bodyStr, "authentication"):
			msg = "推流服务鉴权失败，请检查 mediamtx.yml 中 authInternalUsers 配置"
		case strings.Contains(bodyStr, "no stream is available"):
			msg = "主播尚未推流或推流已中断，请确认 OBS 正在直播"
			errCode = "not_streaming"
		}
		status := resp.StatusCode
		if errCode == "not_streaming" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": errCode, "message": msg, "detail": bodyStr})
		return
	}

	if strings.HasSuffix(fileName, ".m3u8") {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(502, gin.H{"error": "upstream_error", "message": "读取 playlist 失败"})
			return
		}
		proxyBase := "/api/v1/live/rooms/" + id + "/"
		content := rewriteM3U8(string(body), proxyBase)
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.Header("Cache-Control", "no-cache, no-store")
		c.String(200, content)
		return
	}

	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	if ct := resp.Header.Get("Content-Type"); ct == "" {
		if strings.HasSuffix(fileName, ".m4s") || strings.HasSuffix(fileName, ".mp4") {
			c.Header("Content-Type", "video/iso.segment")
		} else {
			c.Header("Content-Type", "video/mp2t")
		}
	}
	c.Header("Cache-Control", "no-cache")
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

func (h *LiveHandler) fetchAndProxyURL(c *gin.Context, room *live.Room, targetURL string) {
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		c.JSON(400, gin.H{"error": "invalid_url", "message": "上游地址无效"})
		return
	}
	if !service.IsSafePublicURL(parsed) {
		c.JSON(403, gin.H{"error": "forbidden", "message": "不允许访问该地址"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		c.JSON(502, gin.H{"error": "upstream_error", "message": "构建上游请求失败"})
		return
	}
	req.Header.Set("User-Agent", "MediaHub-Live-Proxy/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(502, gin.H{"error": "upstream_error", "message": "无法连接 IPTV 源"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{
			"error":   "upstream_error",
			"message": "IPTV 源返回错误",
			"detail":  string(body),
		})
		return
	}

	pathLower := strings.ToLower(parsed.Path)
	ct := resp.Header.Get("Content-Type")
	isPlaylist := strings.HasSuffix(pathLower, ".m3u8") ||
		strings.Contains(strings.ToLower(ct), "mpegurl") ||
		strings.Contains(strings.ToLower(ct), "m3u8")

	if isPlaylist {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(502, gin.H{"error": "upstream_error", "message": "读取 playlist 失败"})
			return
		}
		content, err := service.RewriteIPTVM3U8(string(body), room.ID.String(), targetURL)
		if err != nil {
			c.JSON(502, gin.H{"error": "upstream_error", "message": "解析 playlist 失败"})
			return
		}
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.Header("Cache-Control", "no-cache, no-store")
		c.String(200, content)
		return
	}

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	} else if strings.HasSuffix(pathLower, ".m4s") || strings.HasSuffix(pathLower, ".mp4") {
		c.Header("Content-Type", "video/iso.segment")
	} else if strings.HasSuffix(pathLower, ".ts") {
		c.Header("Content-Type", "video/mp2t")
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
				ref := u.Path[strings.LastIndex(u.Path, "/")+1:]
				if u.RawQuery != "" {
					ref += "?" + u.RawQuery
				}
				lines[i] = proxyBase + ref
			}
		} else {
			lines[i] = proxyBase + trimmed
		}
	}
	return strings.Join(lines, "\n")
}
