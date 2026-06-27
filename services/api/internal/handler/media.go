package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// MediaHandler 媒资 HTTP handler
type MediaHandler struct {
	svc          *service.MediaService
	match        *service.ScrapeMatchService
	allowedRoots []string
	pathAliases  []PathAlias
	mediaRoot    string
}

// NewMediaHandler 构造
func NewMediaHandler(svc *service.MediaService, match *service.ScrapeMatchService, mediaRoot, downloadRoot string, pathAliases []PathAlias) *MediaHandler {
	roots := []string{mediaRoot}
	if downloadRoot != "" && downloadRoot != mediaRoot {
		roots = append(roots, downloadRoot)
	}
	return &MediaHandler{
		svc:          svc,
		match:        match,
		allowedRoots: roots,
		pathAliases:  pathAliases,
		mediaRoot:    mediaRoot,
	}
}

// List 列表
func (h *MediaHandler) List(c *gin.Context) {
	p := common.Pagination{
		Page:     atoi(c.Query("page"), 1),
		PageSize: atoi(c.Query("page_size"), 20),
	}

	f := repository.MediaFilter{
		Type:         c.Query("type"),
		Genre:        c.Query("genre"),
		Search:       c.Query("q"),
		Sort:         c.Query("sort"),
		SortDesc:     c.Query("order") != "asc",
		ScrapeStatus: c.Query("scrape_status"),
	}
	if v := c.Query("year"); v != "" {
		if y, err := strconv.Atoi(v); err == nil {
			f.Year = &y
		}
	}
	if v := c.Query("min_rating"); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinRating = &r
		}
	}

	result, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, result)
}

// Get 详情
func (h *MediaHandler) Get(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

// Probe 媒资 ffprobe 详情（v0.4 A3）
func (h *MediaHandler) Probe(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	probePath := detail.StoragePath
	if detail.Type == common.MediaTypeTVShow || detail.Type == common.MediaTypeAnime {
		for _, season := range detail.Seasons {
			for _, ep := range season.Episodes {
				if ep.FilePath != "" {
					probePath = ep.FilePath
					break
				}
			}
			if probePath != detail.StoragePath {
				break
			}
		}
	}

	cleanPath := resolveStreamPath(probePath, h.pathAliases)
	if !isPathUnderRoots(cleanPath, h.allowedRoots) {
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
	c.JSON(200, gin.H{"data": result.ToMediaProbe(cleanPath)})
}

// Create 手动创建
func (h *MediaHandler) Create(c *gin.Context) {
	var req struct {
		Type          common.MediaType `json:"type" binding:"required"`
		Title         string           `json:"title" binding:"required"`
		OriginalTitle string           `json:"original_title"`
		Year          *int             `json:"year"`
		StoragePath   string           `json:"storage_path" binding:"required"`
		Overview      string           `json:"overview"`
		PosterURL     string           `json:"poster_url"`
		BackdropURL   string           `json:"backdrop_url"`
		Genres        []string         `json:"genres"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	m := &media.Media{
		Type:          req.Type,
		Title:         req.Title,
		OriginalTitle: req.OriginalTitle,
		Year:          req.Year,
		StoragePath:   req.StoragePath,
		Overview:      req.Overview,
		PosterURL:     req.PosterURL,
		BackdropURL:   req.BackdropURL,
		Genres:        req.Genres,
		ScrapeStatus:  common.ScrapeStatusDone, // 手动入库视为已刮削
	}
	if err := h.svc.CreateManual(c.Request.Context(), m); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": m})
}

// Update 更新
func (h *MediaHandler) Update(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	prevTitle := existing.Title
	if v, ok := req["title"].(string); ok {
		existing.Title = strings.TrimSpace(v)
		if existing.Title != prevTitle {
			media.EnsureTag(&existing.Tags, media.TagManualTitle)
		}
	}
	if v, ok := req["original_title"].(string); ok {
		existing.OriginalTitle = strings.TrimSpace(v)
	}
	if v, ok := req["overview"].(string); ok {
		existing.Overview = v
	}
	if v, ok := req["storage_path"].(string); ok {
		v = strings.TrimSpace(v)
		if v == "" {
			respondError(c, apperr.Validation("storage_path 不能为空"))
			return
		}
		resolved, err := ValidateStoragePath(v, h.allowedRoots, h.pathAliases)
		if err != nil {
			respondError(c, err)
			return
		}
		existing.StoragePath = resolved
	}
	if v, ok := req["poster_url"].(string); ok {
		existing.PosterURL = v
	}
	if v, ok := req["backdrop_url"].(string); ok {
		existing.BackdropURL = v
	}
	if v, ok := req["year"]; ok {
		switch y := v.(type) {
		case nil:
			existing.Year = nil
		case float64:
			yr := int(y)
			existing.Year = &yr
		}
	}
	if v, ok := req["rating"].(float64); ok {
		existing.Rating = v
	}
	if v, ok := req["genres"].([]any); ok {
		gs := make(media.StringArray, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				gs = append(gs, s)
			}
		}
		existing.Genres = gs
	}
	if existing.Genres == nil {
		existing.Genres = media.StringArray{}
	}
	if existing.Tags == nil {
		existing.Tags = media.StringArray{}
	}

	if err := h.svc.Update(c.Request.Context(), existing.Media); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": existing})
}

// Delete 删除
func (h *MediaHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

// Rescan 重新刮削
func (h *MediaHandler) Rescan(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Rescan(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(202, gin.H{"status": "queued", "media_id": id})
}

// BatchRescan 批量重新刮削
func (h *MediaHandler) BatchRescan(c *gin.Context) {
	var req struct {
		IDs          []string `json:"ids"`
		ScrapeStatus string   `json:"scrape_status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	result, err := h.svc.BatchRescan(c.Request.Context(), req.IDs, req.ScrapeStatus)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(202, gin.H{"status": "queued", "queued": result.Queued})
}

// ScrapeCandidates 列出 TMDB 刮削候选（失败时 CMS 点选）
func (h *MediaHandler) ScrapeCandidates(c *gin.Context) {
	if h.match == nil {
		respondError(c, apperr.BadRequest("刮削匹配未启用"))
		return
	}
	id := c.Param("id")
	items, err := h.match.ListCandidates(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

// ApplyScrapeMatch 应用选中的 TMDB 条目
func (h *MediaHandler) ApplyScrapeMatch(c *gin.Context) {
	if h.match == nil {
		respondError(c, apperr.BadRequest("刮削匹配未启用"))
		return
	}
	id := c.Param("id")
	var req struct {
		TMDBID int    `json:"tmdb_id" binding:"required"`
		Type   string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	if err := h.match.ApplyMatch(c.Request.Context(), id, req.TMDBID, req.Type); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "applied", "media_id": id})
}

// Stats 统计
func (h *MediaHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": stats})
}

// Search 关键字搜索（为 TV / Android 客户端返回简洁的扁平结果）
func (h *MediaHandler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(200, gin.H{"data": []any{}})
		return
	}

	typeFilter := c.Query("type")
	limit := atoi(c.Query("limit"), 30)
	if limit > 100 {
		limit = 100
	}

	// 直接复用 List 服务（page_size=limit，只取 items）
	f := repository.MediaFilter{
		Type:   typeFilter,
		Search: q,
	}
	p := common.Pagination{Page: 1, PageSize: limit}

	result, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		respondError(c, err)
		return
	}

	// 转换为轻量级输出（去掉 is_adult 等敏感字段，方便客户端直接用）
	out := make([]gin.H, 0, len(result.Items))
	for _, m := range result.Items {
		out = append(out, gin.H{
			"media_id":     m.ID,
			"title":        m.Title,
			"year":         m.Year,
			"poster_url":   m.PosterURL,
			"backdrop_url": m.BackdropURL,
			"rating":       m.Rating,
			"type":         string(m.Type),
			"genres":       m.Genres,
		})
	}

	c.JSON(200, gin.H{
		"data":  out,
		"total": result.Total,
		"q":     q,
	})
}

const maxPosterBytes = 10 << 20 // 10MB

var allowedPosterTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func (h *MediaHandler) posterDir() string {
	return filepath.Join(h.mediaRoot, ".mediahub", "posters")
}

func (h *MediaHandler) posterFilePath(mediaID, ext string) string {
	return filepath.Join(h.posterDir(), mediaID+ext)
}

func (h *MediaHandler) findPosterFile(mediaID string) (string, bool) {
	for _, ext := range allowedPosterTypes {
		p := h.posterFilePath(mediaID, ext)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// GetPoster 返回自定义上传的海报
func (h *MediaHandler) GetPoster(c *gin.Context) {
	id := c.Param("id")
	path, ok := h.findPosterFile(id)
	if !ok {
		respondError(c, apperr.NotFound("poster not found"))
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.File(path)
}

// UploadPoster 上传/替换自定义海报
func (h *MediaHandler) UploadPoster(c *gin.Context) {
	id := c.Param("id")
	if _, err := h.svc.Detail(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	file, err := c.FormFile("poster")
	if err != nil {
		respondError(c, apperr.Validation("请上传 poster 字段的图片文件"))
		return
	}
	if file.Size > maxPosterBytes {
		respondError(c, apperr.Validation("海报文件不能超过 10MB"))
		return
	}

	src, err := file.Open()
	if err != nil {
		respondError(c, apperr.Internal(err.Error()))
		return
	}
	defer src.Close()

	buf := make([]byte, 512)
	n, _ := src.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	ext, ok := allowedPosterTypes[contentType]
	if !ok {
		respondError(c, apperr.Validation("仅支持 JPEG、PNG、WebP 格式"))
		return
	}

	if err := os.MkdirAll(h.posterDir(), 0o755); err != nil {
		respondError(c, apperr.Internal(err.Error()))
		return
	}

	destPath := h.posterFilePath(id, ext)
	// 删除其他扩展名的旧文件
	for _, otherExt := range allowedPosterTypes {
		if otherExt == ext {
			continue
		}
		_ = os.Remove(h.posterFilePath(id, otherExt))
	}

	dest, err := os.Create(destPath)
	if err != nil {
		respondError(c, apperr.Internal(err.Error()))
		return
	}
	defer dest.Close()

	if _, err := dest.Write(buf[:n]); err != nil {
		respondError(c, apperr.Internal(err.Error()))
		return
	}
	if _, err := io.Copy(dest, src); err != nil {
		respondError(c, apperr.Internal(err.Error()))
		return
	}

	posterURL := fmt.Sprintf("/api/v1/media/%s/poster?t=%d", id, time.Now().Unix())
	m, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	m.PosterURL = posterURL
	if err := h.svc.Update(c.Request.Context(), m.Media); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"poster_url": posterURL}})
}
