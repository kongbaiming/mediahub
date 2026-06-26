package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/hlsstore"
	"github.com/mediahub/api/internal/mediafile"
	"github.com/mediahub/api/internal/scanner"
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
	PreferCopy  bool
}

// StreamHandler 流代理
func StreamHandler(allowedRoots []string, hlsCacheRoot string, tc HLSTranscodeSettings, store *hlsstore.Store) gin.HandlerFunc {
	roots := normalizeAllowedRoots(allowedRoots)
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasSuffix(p, "/direct") {
			handleDirect(c, roots)
			return
		}
		if strings.HasSuffix(p, "/hls") {
			handleHLS(c, roots, hlsCacheRoot, tc, store)
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

	if ok, reason := mediafile.IsPlayable(cleanPath); !ok {
		respondError(c, apperr.BadRequest(reason))
		return
	}

	c.Header("Content-Type", sniffVideoMime(cleanPath))
	c.Header("Accept-Ranges", "bytes")
	c.Header("Cache-Control", "no-cache")
	c.File(cleanPath)
}

// ---- HLS 转码 ----

type hlsTask struct {
	MediaID   string
	Input     string
	OutputDir string
	Status    string // pending | running | done | failed
	Error     string
	StartedAt time.Time
	Opts      transcoder.HLSOptions
}

func hlsTaskFromRecord(rec *hlsstore.TaskRecord) *hlsTask {
	if rec == nil {
		return nil
	}
	return &hlsTask{
		MediaID:   rec.MediaID,
		Input:     rec.Input,
		OutputDir: rec.OutputDir,
		Status:    rec.Status,
		Error:     rec.Error,
		StartedAt: rec.StartedAt,
		Opts: transcoder.HLSOptions{
			CopyVideo: rec.CopyVideo,
			Height:    rec.Height,
		},
	}
}

func hlsTaskToRecord(t *hlsTask) *hlsstore.TaskRecord {
	if t == nil {
		return nil
	}
	return &hlsstore.TaskRecord{
		MediaID:   t.MediaID,
		Input:     t.Input,
		OutputDir: t.OutputDir,
		Status:    t.Status,
		Error:     t.Error,
		StartedAt: t.StartedAt,
		CopyVideo: t.Opts.CopyVideo,
		Height:    t.Opts.Height,
	}
}

func persistHLSTask(ctx context.Context, store *hlsstore.Store, t *hlsTask) {
	if store == nil || t == nil {
		return
	}
	store.Set(ctx, hlsTaskToRecord(t))
}

func loadHLSTask(ctx context.Context, store *hlsstore.Store, mediaID string) (*hlsTask, bool) {
	if store == nil {
		return nil, false
	}
	rec, ok := store.Get(ctx, mediaID)
	if !ok {
		return nil, false
	}
	return hlsTaskFromRecord(rec), true
}

type hlsCacheProfile struct {
	CopyVideo bool   `json:"copy_video"`
	Height    int    `json:"height"`
	Bitrate   string `json:"bitrate"`
}

func hlsProfilePath(outDir string) string {
	return filepath.Join(outDir, ".profile.json")
}

func readHLSProfile(outDir string) (*hlsCacheProfile, error) {
	data, err := os.ReadFile(hlsProfilePath(outDir))
	if err != nil {
		return nil, err
	}
	var p hlsCacheProfile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func writeHLSProfile(outDir string, opts transcoder.HLSOptions) error {
	p := hlsCacheProfile{
		CopyVideo: opts.CopyVideo,
		Height:    opts.Height,
		Bitrate:   opts.Bitrate,
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return os.WriteFile(hlsProfilePath(outDir), data, 0644)
}

func hlsProfileMatches(outDir string, opts transcoder.HLSOptions) bool {
	p, err := readHLSProfile(outDir)
	if err != nil {
		return false
	}
	return p.CopyVideo == opts.CopyVideo && p.Height == opts.Height && p.Bitrate == opts.Bitrate
}

func resolveHLSOptions(ctx context.Context, input string, tc HLSTranscodeSettings) transcoder.HLSOptions {
	opts := transcoder.HLSOptions{
		Input:        input,
		SegmentTime:  tc.SegmentTime,
		Preset:       tc.Preset,
		AudioBitrate: "128k",
	}

	if tc.MaxBitrate == "" {
		tc.MaxBitrate = "2500k"
	}
	if tc.Preset == "" {
		tc.Preset = "ultrafast"
	}
	if tc.SegmentTime <= 0 {
		tc.SegmentTime = 4
	}
	opts.Bitrate = tc.MaxBitrate
	opts.Preset = tc.Preset
	opts.SegmentTime = tc.SegmentTime

	result, err := scanner.Probe(ctx, "", input)
	if err != nil {
		if tc.MaxHeight <= 0 {
			opts.Height = 1080
		} else {
			opts.Height = tc.MaxHeight
		}
		return opts
	}

	hint := result.PlaybackHint(input)
	sourceHeight := hint.Height

	if tc.PreferCopy && hint.HLSCopyable {
		opts.CopyVideo = true
		return opts
	}

	height := tc.MaxHeight
	if height <= 0 {
		height = sourceHeight
	}
	if sourceHeight > 0 && (height <= 0 || sourceHeight < height) {
		height = sourceHeight
	}
	if height <= 0 {
		height = 1080
	}
	opts.Height = height
	return opts
}

// StreamProbeHandler 探测文件编码与推荐播放方式
func StreamProbeHandler(allowedRoots []string) gin.HandlerFunc {
	roots := normalizeAllowedRoots(allowedRoots)
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			respondError(c, apperr.BadRequest("missing path"))
			return
		}
		cleanPath := filepath.Clean(path)
		if !isPathUnderRoots(cleanPath, roots) {
			respondError(c, apperr.Forbidden("path outside media root"))
			return
		}
		if info, err := os.Stat(cleanPath); err != nil || info.IsDir() {
			respondError(c, apperr.NotFound("file not found"))
			return
		}

		result, err := scanner.Probe(c.Request.Context(), "", cleanPath)
		if err != nil {
			respondError(c, apperr.Internal("probe failed: "+err.Error()))
			return
		}
		hint := result.PlaybackHint(cleanPath)
		c.JSON(200, hint)
	}
}

func hlsPlaylistURL(mediaID string) string {
	return fmt.Sprintf("/api/v1/stream/hls/%s/playlist.m3u8", mediaID)
}

func hlsStatusResponse(c *gin.Context, t *hlsTask) {
	resp := gin.H{
		"status":     t.Status,
		"task":       t.MediaID,
		"poll_url":   fmt.Sprintf("/api/v1/stream/hls/%s/status", t.MediaID),
		"media_id":   t.MediaID,
		"started":    t.StartedAt,
		"elapsed_s":  int(time.Since(t.StartedAt).Seconds()),
		"copy_video": t.Opts.CopyVideo,
		"height":     t.Opts.Height,
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
func handleHLS(c *gin.Context, allowedRoots []string, cacheRoot string, tc HLSTranscodeSettings, store *hlsstore.Store) {
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

	if ok, reason := mediafile.IsPlayable(cleanPath); !ok {
		respondError(c, apperr.BadRequest(reason))
		return
	}

	outDir := filepath.Join(cacheRoot, mediaID)
	playlistPath := filepath.Join(outDir, "playlist.m3u8")
	hlsOpts := resolveHLSOptions(c.Request.Context(), cleanPath, tc)
	hlsOpts.OutputDir = outDir

	if _, err := os.Stat(playlistPath); err == nil {
		if isHLSPlaylistComplete(playlistPath) && hlsProfileMatches(outDir, hlsOpts) {
			c.JSON(200, gin.H{
				"status":     "ready",
				"playlist":   hlsPlaylistURL(mediaID),
				"cached":     true,
				"copy_video": hlsOpts.CopyVideo,
				"height":     hlsOpts.Height,
			})
			return
		}
		if runningTask, ok := loadHLSTask(c.Request.Context(), store, mediaID); ok && runningTask.Status == "running" {
			hlsStatusResponse(c, runningTask)
			return
		}
		_ = os.RemoveAll(outDir)
	}

	if t, exists := loadHLSTask(c.Request.Context(), store, mediaID); exists {
		if t.Status == "failed" {
			store.Delete(c.Request.Context(), mediaID)
		} else {
			hlsStatusResponse(c, t)
			return
		}
	}

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
		Opts:      hlsOpts,
	}

	persistHLSTask(c.Request.Context(), store, task)
	go runHLSTranscode(task, tc, store)

	c.JSON(202, gin.H{
		"status":     "started",
		"task":       mediaID,
		"poll_url":   fmt.Sprintf("/api/v1/stream/hls/%s/status", mediaID),
		"copy_video": hlsOpts.CopyVideo,
		"height":     hlsOpts.Height,
	})
}

func runHLSTranscode(task *hlsTask, tc HLSTranscodeSettings, store *hlsstore.Store) {
	tr := transcoder.NewTranscoder("ffmpeg", tc.HWAccel)
	opts := task.Opts
	opts.Input = task.Input
	opts.OutputDir = task.OutputDir

	_, err := tr.TranscodeHLSWithFallback(context.Background(), opts)

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		_ = os.RemoveAll(task.OutputDir)
	} else {
		_ = writeHLSProfile(task.OutputDir, opts)
		task.Status = "done"
	}
	persistHLSTask(context.Background(), store, task)
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
func GetHLSTaskStatus(cacheRoot string, store *hlsstore.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaID := c.Param("media_id")
		outDir := filepath.Join(cacheRoot, mediaID)

		if t, exists := loadHLSTask(c.Request.Context(), store, mediaID); exists {
			resp := gin.H{
				"media_id":   t.MediaID,
				"status":     t.Status,
				"error":      t.Error,
				"started":    t.StartedAt,
				"elapsed_s":  int(time.Since(t.StartedAt).Seconds()),
				"copy_video": t.Opts.CopyVideo,
				"height":     t.Opts.Height,
			}
			if t.OutputDir == "" {
				t.OutputDir = outDir
			}
			enrichHLSPlayable(resp, t.OutputDir, t.MediaID)
			if resp["status"] == "done" || t.Status == "done" {
				resp["playlist"] = hlsPlaylistURL(t.MediaID)
			}
			c.JSON(200, resp)
			return
		}

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
			if hlsPlaylistHasSegments(playlistPath) {
				c.JSON(200, gin.H{
					"media_id": mediaID,
					"status":   "running",
					"playable": true,
					"playlist": hlsPlaylistURL(mediaID),
				})
				return
			}
			_ = os.RemoveAll(outDir)
		}

		respondError(c, apperr.NotFound("task not found"))
	}
}

// RecoverHLSTasks API 重启后恢复 Redis 中仍为 running 的 HLS 任务
func RecoverHLSTasks(ctx context.Context, cacheRoot string, tc HLSTranscodeSettings, store *hlsstore.Store) {
	if store == nil {
		return
	}
	for _, rec := range store.ListAll(ctx) {
		if rec.Status != "running" {
			continue
		}
		outDir := rec.OutputDir
		if outDir == "" {
			outDir = filepath.Join(cacheRoot, rec.MediaID)
		}
		playlistPath := filepath.Join(outDir, "playlist.m3u8")
		if isHLSPlaylistComplete(playlistPath) {
			rec.Status = "done"
			store.Set(ctx, rec)
			continue
		}
		if rec.Input == "" {
			rec.Status = "failed"
			rec.Error = "missing input after restart"
			store.Set(ctx, rec)
			continue
		}
		if _, err := os.Stat(rec.Input); err != nil {
			rec.Status = "failed"
			rec.Error = "source file missing"
			store.Set(ctx, rec)
			_ = os.RemoveAll(outDir)
			continue
		}
		task := hlsTaskFromRecord(rec)
		if task.OutputDir == "" {
			task.OutputDir = outDir
		}
		if task.Opts.OutputDir == "" {
			task.Opts.OutputDir = outDir
		}
		task.Status = "running"
		task.StartedAt = time.Now()
		persistHLSTask(ctx, store, task)
		go runHLSTranscode(task, tc, store)
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
