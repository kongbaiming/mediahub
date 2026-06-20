package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mediahub/api/internal/apperr"

	"github.com/gin-gonic/gin"
)

// StreamHandler 流代理
// 路径：/api/v1/stream/*action
//
// 用途：
//   - /api/v1/stream/direct?path=<absolute-path>：直接 ServeFile（适合内网 + 客户端硬解）
//   - /api/v1/stream/hls?path=<absolute-path>&media_id=xxx：HLS 转码流（弱网 / 客户端硬解失败时）
func StreamHandler(mediaRoot string) gin.HandlerFunc {
	hlsCache := "/volume1/docker/mediahub/hls-cache" // DS920+ 标准路径
	return func(c *gin.Context) {
		action := c.Param("action")
		switch action {
		case "direct":
			handleDirect(c, mediaRoot)
		case "hls":
			handleHLS(c, mediaRoot, hlsCache)
		default:
			respondError(c, apperr.NotFound("unknown stream action: "+action))
		}
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

	// 嗅探 Content-Type
	c.Header("Content-Type", sniffVideoMime(cleanPath))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "no-cache")
	c.File(cleanPath)
}

// ---- HLS 转码 ----

var (
	hlsTasks   = map[string]*hlsTask{} // mediaID -> task
	hlsTasksMu sync.RWMutex
)

type hlsTask struct {
	MediaID   string
	Input     string
	OutputDir string
	Status    string // pending | running | done | failed
	Error     string
	StartedAt time.Time
	Cmd       *exec.Cmd
}

// handleHLS 启动 HLS 转码（异步）
func handleHLS(c *gin.Context, mediaRoot, cacheRoot string) {
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

	outDir := filepath.Join(cacheRoot, mediaID)
	playlistPath := filepath.Join(outDir, "playlist.m3u8")

	// 检查是否已转码完成
	if _, err := os.Stat(playlistPath); err == nil {
		c.JSON(200, gin.H{
			"status":   "ready",
			"playlist": fmt.Sprintf("/api/v1/stream/hls/%s/playlist.m3u8", mediaID),
			"cached":   true,
		})
		return
	}

	// 检查是否已在转码
	hlsTasksMu.RLock()
	t, exists := hlsTasks[mediaID]
	hlsTasksMu.RUnlock()
	if exists {
		c.JSON(202, gin.H{
			"status": t.Status,
			"task":   mediaID,
		})
		return
	}

	// 启动转码
	if err := os.MkdirAll(outDir, 0755); err != nil {
		respondError(c, apperr.Wrap(err, apperr.CodeInternal, "创建缓存目录失败"))
		return
	}

	task := &hlsTask{
		MediaID:   mediaID,
		Input:     cleanPath,
		OutputDir: outDir,
		Status:    "running",
		StartedAt: time.Now(),
	}

	// FFmpeg 命令：HLS 切片（Quick Sync 硬转优先）
	args := []string{
		"-hwaccel", "qsv",
		"-i", cleanPath,
		"-c:v", "h264_qsv",
		"-preset", "veryfast",
		"-b:v", "4000k",
		"-c:a", "aac",
		"-b:a", "128k",
		"-hls_time", "6",
		"-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(outDir, "seg_%03d.ts"),
		"-vf", "scale=-2:720",
		"-f", "hls",
		"-y",
		playlistPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	task.Cmd = cmd

	hlsTasksMu.Lock()
	hlsTasks[mediaID] = task
	hlsTasksMu.Unlock()

	if err := cmd.Start(); err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		respondError(c, apperr.Wrap(err, apperr.CodeInternal, "启动转码失败"))
		return
	}

	// 异步等待完成
	go func() {
		err := cmd.Wait()
		hlsTasksMu.Lock()
		defer hlsTasksMu.Unlock()
		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			return
		}
		task.Status = "done"
	}()

	c.JSON(202, gin.H{
		"status": "started",
		"task":   mediaID,
		"poll_url": fmt.Sprintf("/api/v1/stream/hls/%s/status", mediaID),
	})
}

// ServeHLSPlaylist 提供 HLS playlist 和 ts 切片
func ServeHLSPlaylist(cacheRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaID := c.Param("media_id")
		file := c.Param("file") // playlist.m3u8 or seg_001.ts etc.

		path := filepath.Join(cacheRoot, mediaID, file)
		// 安全检查：必须包含 mediaID 子目录
		expectedPrefix := filepath.Join(cacheRoot, mediaID)
		if !strings.HasPrefix(path, expectedPrefix) {
			respondError(c, apperr.Forbidden("invalid path"))
			return
		}
		if _, err := os.Stat(path); err != nil {
			respondError(c, apperr.NotFound("segment not found"))
			return
		}

		// 设置合适的 Content-Type
		if strings.HasSuffix(file, ".m3u8") {
			c.Header("Content-Type", "application/vnd.apple.mpegurl")
		} else if strings.HasSuffix(file, ".ts") {
			c.Header("Content-Type", "video/mp2t")
		}

		c.Header("Cache-Control", "public, max-age=60")
		c.File(path)
	}
}

// GetHLSTaskStatus 查询转码状态
func GetHLSTaskStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaID := c.Param("media_id")
		hlsTasksMu.RLock()
		t, exists := hlsTasks[mediaID]
		hlsTasksMu.RUnlock()
		if !exists {
			// 可能已完成但已清理（简化：直接检查文件存在）
			respondError(c, apperr.NotFound("task not found"))
			return
		}
		c.JSON(200, gin.H{
			"media_id":  t.MediaID,
			"status":    t.Status,
			"error":     t.Error,
			"started":   t.StartedAt,
			"elapsed_s": int(time.Since(t.StartedAt).Seconds()),
		})
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

// 确保引用
var _ = context.Background
