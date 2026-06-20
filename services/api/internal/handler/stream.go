package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/transcoder"

	"github.com/gin-gonic/gin"
)

// StreamHandler 流代理
// 路径（具体路由，不再用 catch-all，因为 Gin 不允许 wildcard 和 static 段共存）：
//   - /api/v1/stream/direct?path=<absolute-path>       直接 ServeFile（适合内网 + 客户端硬解）
//   - /api/v1/stream/hls?path=<absolute-path>&media_id=xxx  启动 HLS 转码流（弱网 / 客户端硬解失败时）
//
// 实现：按 request URL path 后缀判断走 direct 还是 hls，避免依赖 c.Param("action")。
func StreamHandler(mediaRoot, hlsCacheRoot, hwAccel, maxBitrate string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasSuffix(p, "/direct") {
			handleDirect(c, mediaRoot)
			return
		}
		if strings.HasSuffix(p, "/hls") {
			handleHLS(c, mediaRoot, hlsCacheRoot, hwAccel, maxBitrate)
			return
		}
		respondError(c, apperr.NotFound("unknown stream action: "+p))
	}
}

// handleDirect 直接 ServeFile
func handleDirect(c *gin.Context, mediaRoot string) {
	path := c.Query("path")
	if path == "" {
		respondError(c, apperr.BadRequest("missing path"))
		return
	}

	cleanRoot := filepath.Clean(mediaRoot)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanRoot) {
		respondError(c, apperr.Forbidden("path outside media root"))
		return
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(c, apperr.NotFound("file not found"))
			return
		}
		respondError(c, apperr.Internal(err.Error()))
		return
	}
	if info.IsDir() {
		respondError(c, apperr.BadRequest("path is a directory"))
		return
	}

	c.Header("Content-Type", sniffVideoMime(cleanPath))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "no-cache")
	c.File(cleanPath)
}

// ---- HLS 转码 ----

var (
	hlsTasks   = map[string]*hlsTask{}
	hlsTasksMu sync.RWMutex
)

type hlsTask struct {
	MediaID   string
	Input     string
	OutputDir string
	Status    string // pending | running | done | failed
	Error     string
	StartedAt time.Time
}

func hlsPlaylistURL(mediaID string) string {
	return fmt.Sprintf("/api/v1/stream/hls/%s/playlist.m3u8", mediaID)
}

func hlsStatusResponse(c *gin.Context, t *hlsTask) {
	resp := gin.H{
		"status":    t.Status,
		"task":      t.MediaID,
		"poll_url":  fmt.Sprintf("/api/v1/stream/hls/%s/status", t.MediaID),
		"media_id":  t.MediaID,
		"started":   t.StartedAt,
		"elapsed_s": int(time.Since(t.StartedAt).Seconds()),
	}
	if t.Error != "" {
		resp["error"] = t.Error
	}
	enrichHLSPlayable(resp, t.OutputDir, t.MediaID)
	if t.Status == "done" {
		resp["playlist"] = hlsPlaylistURL(t.MediaID)
		c.JSON(200, resp)
		return
	}
	c.JSON(202, resp)
}

func enrichHLSPlayable(resp gin.H, outputDir, mediaID string) {
	playlistPath := filepath.Join(outputDir, "playlist.m3u8")
	if isHLSPlaylistComplete(playlistPath) {
		resp["status"] = "done"
		resp["playable"] = true
		resp["playlist"] = hlsPlaylistURL(mediaID)
		return
	}
	if hlsPlaylistHasSegments(playlistPath) {
		resp["playable"] = true
		resp["playlist"] = hlsPlaylistURL(mediaID)
	}
}

func isHLSPlaylistComplete(playlistPath string) bool {
	data, err := os.ReadFile(playlistPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "#EXT-X-ENDLIST")
}

func hlsPlaylistHasSegments(playlistPath string) bool {
	data, err := os.ReadFile(playlistPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "#EXTINF:")
}

// handleHLS 启动 HLS 转码（异步）
func handleHLS(c *gin.Context, mediaRoot, cacheRoot, hwAccel, maxBitrate string) {
	mediaID := c.Query("media_id")
	path := c.Query("path")
	if path == "" || mediaID == "" {
		respondError(c, apperr.BadRequest("missing path or media_id"))
		return
	}

	cleanRoot := filepath.Clean(mediaRoot)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, cleanRoot) {
		respondError(c, apperr.Forbidden("path outside media root"))
		return
	}

	if info, err := os.Stat(cleanPath); err != nil {
		if os.IsNotExist(err) {
			respondError(c, apperr.NotFound("file not found"))
			return
		}
		respondError(c, apperr.Internal(err.Error()))
		return
	} else if info.IsDir() {
		respondError(c, apperr.BadRequest("path is a directory; tvshow must use episode file_path"))
		return
	}

	outDir := filepath.Join(cacheRoot, mediaID)
	playlistPath := filepath.Join(outDir, "playlist.m3u8")

	if _, err := os.Stat(playlistPath); err == nil {
		if isHLSPlaylistComplete(playlistPath) {
			c.JSON(200, gin.H{
				"status":   "ready",
				"playlist": hlsPlaylistURL(mediaID),
				"cached":   true,
			})
			return
		}
		hlsTasksMu.RLock()
		runningTask, ok := hlsTasks[mediaID]
		hlsTasksMu.RUnlock()
		if ok && runningTask.Status == "running" {
			hlsStatusResponse(c, runningTask)
			return
		}
		_ = os.RemoveAll(outDir)
	}

	hlsTasksMu.Lock()
	t, exists := hlsTasks[mediaID]
	if exists && t.Status == "failed" {
		delete(hlsTasks, mediaID)
		exists = false
	}
	hlsTasksMu.Unlock()

	if exists {
		hlsStatusResponse(c, t)
		return
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		respondError(c, apperr.Wrap(err, apperr.CodeInternal, "创建缓存目录失败"))
		return
	}

	if maxBitrate == "" {
		maxBitrate = "4000k"
	}

	task := &hlsTask{
		MediaID:   mediaID,
		Input:     cleanPath,
		OutputDir: outDir,
		Status:    "running",
		StartedAt: time.Now(),
	}

	hlsTasksMu.Lock()
	hlsTasks[mediaID] = task
	hlsTasksMu.Unlock()

	go runHLSTranscode(task, hwAccel, maxBitrate)

	c.JSON(202, gin.H{
		"status":   "started",
		"task":     mediaID,
		"poll_url": fmt.Sprintf("/api/v1/stream/hls/%s/status", mediaID),
	})
}

func runHLSTranscode(task *hlsTask, hwAccel, maxBitrate string) {
	tr := transcoder.NewTranscoder("ffmpeg", hwAccel)
	opts := transcoder.HLSOptions{
		Input:        task.Input,
		OutputDir:    task.OutputDir,
		Height:       720,
		Bitrate:      maxBitrate,
		AudioBitrate: "128k",
		SegmentTime:  6,
	}

	_, err := tr.TranscodeHLSWithFallback(context.Background(), opts)

	hlsTasksMu.Lock()
	defer hlsTasksMu.Unlock()
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		_ = os.RemoveAll(task.OutputDir)
		return
	}
	task.Status = "done"
}

// ServeHLSPlaylist 提供 HLS playlist 和 ts 切片
func ServeHLSPlaylist(cacheRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaID := c.Param("media_id")
		file := c.Param("file")
		if file == "" && strings.HasSuffix(c.Request.URL.Path, "playlist.m3u8") {
			file = "playlist.m3u8"
		}
		if file == "" {
			respondError(c, apperr.BadRequest("missing segment file"))
			return
		}

		path := filepath.Join(cacheRoot, mediaID, file)
		expectedPrefix := filepath.Join(cacheRoot, mediaID)
		if !strings.HasPrefix(path, expectedPrefix) {
			respondError(c, apperr.Forbidden("invalid path"))
			return
		}
		if _, err := os.Stat(path); err != nil {
			respondError(c, apperr.NotFound("segment not found"))
			return
		}

		if strings.HasSuffix(file, ".m3u8") {
			c.Header("Content-Type", "application/vnd.apple.mpegurl")
			c.Header("Cache-Control", "no-cache, no-store")
		} else if strings.HasSuffix(file, ".ts") {
			c.Header("Content-Type", "video/mp2t")
			c.Header("Cache-Control", "public, max-age=86400")
		}

		c.File(path)
	}
}

// GetHLSTaskStatus 查询转码状态
func GetHLSTaskStatus(cacheRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaID := c.Param("media_id")

		hlsTasksMu.RLock()
		t, exists := hlsTasks[mediaID]
		hlsTasksMu.RUnlock()

		if exists {
			resp := gin.H{
				"media_id":  t.MediaID,
				"status":    t.Status,
				"error":     t.Error,
				"started":   t.StartedAt,
				"elapsed_s": int(time.Since(t.StartedAt).Seconds()),
			}
			enrichHLSPlayable(resp, t.OutputDir, t.MediaID)
			if resp["status"] == "done" || t.Status == "done" {
				resp["playlist"] = hlsPlaylistURL(t.MediaID)
			}
			c.JSON(200, resp)
			return
		}

		outDir := filepath.Join(cacheRoot, mediaID)
		playlistPath := filepath.Join(outDir, "playlist.m3u8")
		if _, err := os.Stat(playlistPath); err == nil {
			if isHLSPlaylistComplete(playlistPath) {
				c.JSON(200, gin.H{
					"media_id": mediaID,
					"status":   "done",
					"playlist": hlsPlaylistURL(mediaID),
				})
				return
			}
			_ = os.RemoveAll(outDir)
		}

		respondError(c, apperr.NotFound("task not found"))
	}
}

// sniffVideoMime 根据扩展名嗅探 MIME
func sniffVideoMime(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mkv", ".webm":
		return "video/webm"
	case ".mp4", ".m4v", ".mov":
		return "video/mp4"
	case ".ts", ".m2ts":
		return "video/mp2t"
	case ".avi":
		return "video/x-msvideo"
	default:
		return "application/octet-stream"
	}
}

// parseStreamPath 安全解析
func parseStreamPath(raw string) (string, error) {
	u, err := url.QueryUnescape(raw)
	if err != nil {
		return "", err
	}
	return filepath.Clean(u), nil
}

// hashKey 简单哈希（用于缓存 key）
func hashKey(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])[:12]
}
