// Package worker 提供 Asynq 任务处理器实现
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/internal/scraper"
	"github.com/mediahub/api/internal/transcoder"
	"github.com/mediahub/api/pkg/logger"

	"github.com/hibiken/asynq"
	"github.com/google/uuid"
)

// Handlers 聚合所有任务处理器
type Handlers struct {
	tmdb       *scraper.TMDBClient
	transcoder *transcoder.Transcoder
	mediaRepo  *repository.MediaRepo
	thumbDir   string
}

// NewHandlers 构造
func NewHandlers(
	tmdb *scraper.TMDBClient,
	t *transcoder.Transcoder,
	repo *repository.MediaRepo,
	thumbDir string,
) *Handlers {
	return &Handlers{
		tmdb:       tmdb,
		transcoder: t,
		mediaRepo:  repo,
		thumbDir:   thumbDir,
	}
}

// Register 注册到 Asynq mux
func (h *Handlers) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(queue.TypeScrapeMedia, h.HandleScrape)
	mux.HandleFunc(queue.TypeGenerateThumb, h.HandleThumb)
	mux.HandleFunc(queue.TypeScanDirectory, h.HandleScan)
}

// ---- 刮削任务 ----

// ScrapePayload 刮削任务 payload
type ScrapePayload struct {
	MediaID string `json:"media_id"`
}

// HandleScrape 处理媒资刮削
func (h *Handlers) HandleScrape(ctx context.Context, t *asynq.Task) error {
	var p ScrapePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("解析 payload: %w", err)
	}

	mid, err := uuid.Parse(p.MediaID)
	if err != nil {
		return fmt.Errorf("无效 media_id: %w", err)
	}

	logger.Info("开始刮削", "media_id", mid)

	// 1. 加载媒资
	m, err := h.mediaRepo.GetByID(ctx, mid.String())
	if err != nil {
		return fmt.Errorf("加载媒资: %w", err)
	}

	// 2. 标记 scraping
	m.ScrapeStatus = common.ScrapeStatusScraping
	_ = h.mediaRepo.Update(ctx, m)

	// 3. 搜索 TMDB
	var tmdbInfo *scraperResult
	if m.IsTV() {
		season, episode := extractSeasonEpisode(m)
		tmdbInfo, err = h.searchTVShow(ctx, m.Title, m.Year, season, episode)
	} else {
		tmdbInfo, err = h.searchMovie(ctx, m.Title, m.Year)
	}

	if err != nil {
		m.ScrapeStatus = common.ScrapeStatusFailed
		m.ScrapeError = err.Error()
		_ = h.mediaRepo.Update(ctx, m)
		return fmt.Errorf("TMDB 刮削: %w", err)
	}

	// 4. 合并元数据
	h.applyTMDB(m, tmdbInfo)

	// 5. 探测文件（ffprobe）
	if info, err := scanner.Probe(ctx, "", m.StoragePath); err == nil {
		mi := info.Extract()
		if m.Runtime == nil && mi.Duration > 0 {
			d := mi.Duration / 60
			m.Runtime = &d
		}
		if mi.VideoCodec != "" {
			vc := mi.VideoCodec
			m.VideoCodec = &vc
		}
		if mi.AudioCodec != "" {
			ac := mi.AudioCodec
			m.AudioCodec = &ac
		}
		if mi.Resolution != "" {
			r := mi.Resolution
			m.Resolution = &r
		}
		if mi.HasSubtitle {
			m.HasSubtitle = true
		}
		if mi.BitRate > 0 {
			m.FileSize = estimateFileSize(mi.BitRate, mi.Duration)
		}
	}

	// 6. 标记成功
	m.ScrapeStatus = common.ScrapeStatusDone
	m.ScrapeError = ""
	now := nowTime()
	m.LastScrapeAt = &now

	if err := h.mediaRepo.Update(ctx, m); err != nil {
		return fmt.Errorf("更新媒资: %w", err)
	}

	logger.Info("刮削成功", "media_id", mid, "title", m.Title)
	return nil
}

// scraperResult 内部中间结构
type scraperResult struct {
	TMDBID     int
	Title      string
	Overview   string
	PosterURL  string
	BackdropURL string
	Rating     float64
	VoteCount  int
	Genres     []string
	Year       *int
	Runtime    *int
}

func (h *Handlers) searchMovie(ctx context.Context, title string, year *int) (*scraperResult, error) {
	res, err := h.tmdb.SearchMovie(ctx, title, year)
	if err != nil {
		return nil, err
	}
	if len(res.Results) == 0 {
		return nil, fmt.Errorf("TMDB 未找到电影: %s", title)
	}
	// 取第一个结果
	first := res.Results[0]
	m, err := h.tmdb.GetMovie(ctx, first.ID)
	if err != nil {
		return nil, err
	}
	return h.movieToResult(m), nil
}

func (h *Handlers) searchTVShow(ctx context.Context, title string, year *int, season, episode *int) (*scraperResult, error) {
	res, err := h.tmdb.SearchTV(ctx, title, year)
	if err != nil {
		return nil, err
	}
	if len(res.Results) == 0 {
		return nil, fmt.Errorf("TMDB 未找到剧集: %s", title)
	}
	tvID := res.Results[0].ID
	t, err := h.tmdb.GetTVShow(ctx, tvID)
	if err != nil {
		return nil, err
	}
	// 季/集元数据
	if season != nil {
		s, err := h.tmdb.GetSeason(ctx, tvID, *season)
		if err == nil && s != nil {
			return h.tvToResult(t, s), nil
		}
	}
	return h.tvToResultNoSeason(t), nil
}

func (h *Handlers) movieToResult(m *scraper.TMDBMovie) *scraperResult {
	r := &scraperResult{
		TMDBID:      m.ID,
		Title:       m.Title,
		Overview:    m.Overview,
		PosterURL:   h.tmdb.PosterURL(m.PosterPath, "w500"),
		BackdropURL: h.tmdb.BackdropURL(m.BackdropPath, "w1280"),
		Rating:      m.VoteAverage,
		VoteCount:   m.VoteCount,
		Genres:      genreNames(m.Genres),
	}
	if y, err := strconvAtoi(m.ReleaseDate); err == nil && y > 0 {
		r.Year = &y
	}
	if m.Runtime > 0 {
		rt := m.Runtime
		r.Runtime = &rt
	}
	return r
}

func (h *Handlers) tvToResult(t *scraper.TMDBTVShow, s *scraper.TMDBSeason) *scraperResult {
	r := &scraperResult{
		TMDBID:      t.ID,
		Title:       t.Name,
		Overview:    t.Overview,
		PosterURL:   h.tmdb.PosterURL(t.PosterPath, "w500"),
		BackdropURL: h.tmdb.BackdropURL(t.BackdropPath, "w1280"),
		Rating:      t.VoteAverage,
		VoteCount:   t.VoteCount,
		Genres:      genreNames(t.Genres),
	}
	if s != nil {
		r.PosterURL = h.tmdb.PosterURL(s.PosterPath, "w500")
	}
	if y, err := strconvAtoi(t.FirstAirDate); err == nil && y > 0 {
		r.Year = &y
	}
	if len(t.EpisodeRunTime) > 0 && t.EpisodeRunTime[0] > 0 {
		rt := t.EpisodeRunTime[0]
		r.Runtime = &rt
	}
	return r
}

func (h *Handlers) tvToResultNoSeason(t *scraper.TMDBTVShow) *scraperResult {
	return h.tvToResult(t, nil)
}

func (h *Handlers) applyTMDB(m *media.Media, info *scraperResult) {
	if info.TMDBID > 0 {
		id := info.TMDBID
		m.TMDBID = &id
	}
	if info.Overview != "" {
		m.Overview = info.Overview
	}
	if info.PosterURL != "" {
		m.PosterURL = info.PosterURL
	}
	if info.BackdropURL != "" {
		m.BackdropURL = info.BackdropURL
	}
	if info.Rating > 0 {
		m.Rating = info.Rating
	}
	if info.VoteCount > 0 {
		m.VoteCount = info.VoteCount
	}
	if len(info.Genres) > 0 {
		m.Genres = info.Genres
	}
	if info.Year != nil && m.Year == nil {
		m.Year = info.Year
	}
	if info.Runtime != nil && m.Runtime == nil {
		m.Runtime = info.Runtime
	}
}

func genreNames(gs []struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}) []string {
	out := make([]string, 0, len(gs))
	for _, g := range gs {
		out = append(out, g.Name)
	}
	return out
}

// ---- 缩略图任务 ----

// ThumbPayload 缩略图任务 payload
type ThumbPayload struct {
	MediaID string `json:"media_id"`
}

// HandleThumb 生成缩略图
func (h *Handlers) HandleThumb(ctx context.Context, t *asynq.Task) error {
	var p ThumbPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("解析 payload: %w", err)
	}

	m, err := h.mediaRepo.GetByID(ctx, p.MediaID)
	if err != nil {
		return err
	}

	outDir := filepath.Join(h.thumbDir, m.ID.String())
	logger.Info("生成缩略图", "media_id", m.ID, "out_dir", outDir)

	thumbs, err := h.transcoder.GenerateThumbnails(ctx, transcoder.ThumbnailOptions{
		Input:     m.StoragePath,
		OutputDir: outDir,
		Count:     3,
		Width:     640,
	})
	if err != nil {
		logger.Warn("缩略图生成失败", "err", err)
		return err
	}

	logger.Info("缩略图生成成功", "count", len(thumbs))
	return nil
}

// ---- 扫描任务 ----

// ScanPayload 扫描任务 payload
type ScanPayload struct {
	Path string `json:"path"`
}

// HandleScan 处理目录扫描
func (h *Handlers) HandleScan(ctx context.Context, t *asynq.Task) error {
	var p ScanPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("解析 payload: %w", err)
	}

	logger.Info("扫描目录", "path", p.Path)

	s, err := scanner.NewScanner([]string{p.Path})
	if err != nil {
		return err
	}
	defer s.Stop()

	count := 0
	s.OnChange(func(ev scanner.FileEvent) {
		if ev.Type == "deleted" {
			return
		}
		if !scanner.IsMediaFile(ev.Path) {
			return
		}
		// 落库（如果不存在）
		existing, _ := h.mediaRepo.GetByStoragePath(ctx, ev.Path)
		if existing != nil {
			return
		}

		parsed := ev.Parsed
		if parsed == nil {
			parsed = scanner.ParseFileName(ev.Path)
		}

		m := &media.Media{
			Type:        common.MediaType(scanner.InferMediaType(parsed, filepath.Dir(ev.Path))),
			Title:       parsed.Title,
			Year:        parsed.Year,
			StoragePath: ev.Path,
			Container:   &parsed.Container,
			VideoCodec:  strPtr(parsed.VideoCodec),
			AudioCodec:  strPtr(parsed.AudioCodec),
			ScrapeStatus: common.ScrapeStatusPending,
			Tags:        []string{parsed.Resolution, parsed.Source, parsed.Group},
		}
		if m.Tags[0] == "" {
			m.Tags = []string{}
		}
		if err := h.mediaRepo.Create(ctx, m); err != nil {
			logger.Warn("入库失败", "path", ev.Path, "err", err)
			return
		}
		count++
		logger.Info("媒资入库", "id", m.ID, "title", m.Title, "path", ev.Path)
	})

	scanCtx, cancel := context.WithTimeout(ctx, 30*60*1e9) // 30 分钟
	defer cancel()
	s.Start(scanCtx)

	logger.Info("扫描完成", "count", count)
	return nil
}

// ---- helpers ----

func extractSeasonEpisode(m *media.Media) (*int, *int) {
	// 从第一个 episode 提取（实际应从 episodes 表查询）
	if len(m.Seasons) == 0 || len(m.Seasons[0].Episodes) == 0 {
		return nil, nil
	}
	ep := m.Seasons[0].Episodes[0]
	s := m.Seasons[0].SeasonNumber
	return &s, &ep.EpisodeNumber
}

func strconvAtoi(s string) (int, error) {
	if len(s) < 4 {
		return 0, fmt.Errorf("too short")
	}
	y, err := atoiYear(s[:4])
	if err != nil {
		return 0, err
	}
	return y, nil
}

func atoiYear(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nowTime() time.Time { return time.Now() }

// FileSize 自动估算（无 ffprobe 时）
func estimateFileSize(bitrate int64, duration int) int64 {
	if bitrate <= 0 || duration <= 0 {
		return 0
	}
	return bitrate / 8 * int64(duration)
}
