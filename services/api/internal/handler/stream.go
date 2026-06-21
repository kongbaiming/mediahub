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

// HLSTranscodeSettings HLS 转码参数（来自环境变量）
type HLSTranscodeSettings struct {
	HWAccel     string
	MaxBitrate  string
	MaxHeight   int
	Preset      string
	SegmentTime int
}

// StreamHandler 流代理
// 路径（具体路由，不再用 catch-all，因为 Gin 不允许 wildcard 和 static 段共存）：
//   - /api/v1/stream/direct?path=<absolute-path>       直接 ServeFile（适合内网 + 客户端硬解）
//   - /api/v1/stream/hls?path=<absolute-path>&media_id=xxx  启动 HLS 转码流（弱网 / 客户端硬解失败时）
//
// allowedRoots 通常为 MEDIA_ROOT 与 DOWNLOAD_ROOT（下载目录中的媒资也需可播）。
func StreamHandler(allowedRoots []string, hlsCacheRoot string, tc HLSTranscodeSettings) gin.HandlerFunc {
	roots := normalizeAllowedRoots(allowedRoots)
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasSuffix(p, "/direct") {
			handleDirect(c, roots)
			return
		}
		if strings.HasSuffix(p, "/hls") {
			handleHLS(c, roots, hlsCacheRoot, tc)
			return
		}
		respondError(c, apperr.NotFound("unknown stream action: "+p))
	}
}

func normalizeAllowedRoots(roots []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(roots))
	for _, r := range roots {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, filepath.Clean(r))
	}
	return out
}

func isPathUnderRoots(path string, roots []string) bool {
	cleanPath := filepath.Clean(path)
	for _, root := range roots {
		if cleanPath == root || strings.HasPrefix(cleanPath, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// handleDirect 直接 ServeFile
func handleDirect(c *gin.Context, allowedRoots []string) {
	path := c.Query("path")
	if path == "" {
		respondError(c, apperr.BadRequest("missing path"))
		return
	}

	cleanPath := filepath.Clean(path)
	if !isPathUnderRoots(cleanPath, allowedRoots) {
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
func handleHLS(c *gin.Context, allowedRoots []string, cacheRoot string, tc HLSTranscodeSettings) {
	mediaID := c.Query("media_id")
	path := c.Query("path")
	if path == "" || mediaID == "" {
		respondError(c, apperr.BadRequest("missing path or media_id"))
		return
	}

	cleanPath := filepath.Clean(path)
	if !isPathUnderRoots(cleanPath, allowedRoots) {
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

	if tc.MaxBitrate == "" {
		tc.MaxBitrate = "2500k"
	}
	if tc.MaxHeight <= 0 {
		tc.MaxHeight = 480
	}
	if tc.Preset == "" {
		tc.Preset = "ultrafast"
	}
	if tc.SegmentTime <= 0 {
		tc.SegmentTime = 4
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

	go runHLSTranscode(task, tc)

	c.JSON(202, gin.H{
		"status":   "started",
		"task":     mediaID,
		"poll_url": fmt.Sprintf("/api/v1/stream/hls/%s/status", mediaID),
	})
}

func runHLSTranscode(task *hlsTask, tc HLSTranscodeSettings) {
	tr := transcoder.NewTranscoder("ffmpeg", tc.HWAccel)
	opts := transcoder.HLSOptions{
		Input:        task.Input,
		OutputDir:    task.OutputDir,
		Height:       tc.MaxHeight,
		Bitrate:      tc.MaxBitrate,
		AudioBitrate: "128k",
		SegmentTime:  tc.SegmentTime,
		Preset:       tc.Preset,
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
