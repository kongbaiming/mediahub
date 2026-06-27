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

// CatalogEnricher 刮削后扩展目录元数据
type CatalogEnricher interface {
	EnrichFromTMDB(ctx context.Context, m *media.Media) error
}

// Handlers 聚合所有任务处理器
type Handlers struct {
	tmdb            *scraper.TMDBClient
	transcoder      *transcoder.Transcoder
	mediaRepo       *repository.MediaRepo
	queue           *queue.Queue
	thumbDir        string
	enricher        CatalogEnricher
	feedInvalidator func(ctx context.Context, platform string) error
}

// SetFeedInvalidator 刮削完成后失效 Feed 缓存（v0.4 A6）
func (h *Handlers) SetFeedInvalidator(fn func(ctx context.Context, platform string) error) {
	h.feedInvalidator = fn
}

func (h *Handlers) invalidateFeedAfterScrape(ctx context.Context) {
	if h.feedInvalidator == nil {
		return
	}
	_ = h.feedInvalidator(ctx, "web")
	_ = h.feedInvalidator(ctx, "android-tv")
}

// NewHandlers 构造
func NewHandlers(
	tmdb *scraper.TMDBClient,
	t *transcoder.Transcoder,
	repo *repository.MediaRepo,
	q *queue.Queue,
	thumbDir string,
	enricher CatalogEnricher,
) *Handlers {
	return &Handlers{
		tmdb:       tmdb,
		transcoder: t,
		mediaRepo:  repo,
		queue:      q,
		thumbDir:   thumbDir,
		enricher:   enricher,
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

	// 2. 标记 scraping（仅更新状态，不清空已有元数据）
	_ = h.mediaRepo.UpdateScrapeStatus(ctx, mid.String(), string(common.ScrapeStatusScraping), "")

	// 2b. ffprobe：读取内嵌元数据与时长（用于 IMDB 直查 / 搜索候选 / 时长消歧）
	probePath := h.probePathForMedia(ctx, mid.String(), m)
	var probeResult *scanner.ProbeResult
	emb := scanner.EmbeddedMeta{}
	if info, probeErr := scanner.Probe(ctx, "", probePath); probeErr == nil {
		probeResult = info
		emb = scanner.ExtractEmbeddedMeta(info)
	}
	durationSec := emb.DurationSec
	if emb.Year != nil && m.Year == nil {
		m.Year = emb.Year
	}
	savedPoster := m.PosterURL
	savedBackdrop := m.BackdropURL
	manualTitle := media.HasTag(m.Tags, media.TagManualTitle)
	searchOpts := &scanner.SearchCandidateOpts{
		PreferFolderOverEmbedded: manualTitle,
		ReferenceTitle:           m.Title,
	}

	// 3. 搜索 TMDB
	var tmdbInfo *scraperResult
	if emb.IMDBID != "" {
		if info, imdbErr := h.scrapeByIMDB(ctx, emb.IMDBID, m.IsTV()); imdbErr == nil {
			tmdbInfo = info
		}
	}
	if tmdbInfo == nil && manualTitle && m.TMDBID != nil && *m.TMDBID > 0 {
		season := emb.Season
		if season == nil {
			parsed := scanner.ParseFilePath(probePath)
			season = parsed.Season
		}
		if info, refreshErr := h.scrapeByTMDBID(ctx, *m.TMDBID, m.IsTV(), season); refreshErr == nil {
			tmdbInfo = info
		}
	}
	if tmdbInfo == nil && m.IsTV() {
		folder := scanner.AlbumFolderName(m.StoragePath)
		searchYear := m.Year
		if searchYear == nil {
			searchYear = scanner.SeriesFolderYear(folder)
		}
		if searchYear == nil && emb.Year != nil {
			searchYear = emb.Year
		}
		season, episode := emb.Season, emb.Episode
		var lastErr error
		for _, searchTitle := range scanner.TVSearchCandidates(m.StoragePath, m.Title, &emb, searchOpts) {
			tmdbInfo, err = h.searchTVShow(ctx, searchTitle, searchYear, season, episode)
			if err == nil {
				break
			}
			lastErr = err
			if searchYear != nil {
				tmdbInfo, err = h.searchTVShow(ctx, searchTitle, nil, season, episode)
				if err == nil {
					break
				}
				lastErr = err
			}
		}
		if tmdbInfo == nil && lastErr != nil {
			err = lastErr
		}
		// 误将电影单文件按剧集入库（如 冰河世纪4/053.mp4）时，回退按电影刮削
		if err != nil {
			if info, movieErr := h.searchMovieCandidates(ctx, m, &emb, durationSec); movieErr == nil {
				tmdbInfo = info
				err = nil
				m.Type = common.MediaTypeMovie
				m.Kind = media.MediaKindSingle
				logger.Info("剧集刮削失败，已按电影匹配", "media_id", mid, "title", m.Title)
			}
		}
	} else if tmdbInfo == nil {
		searchYear := m.Year
		if searchYear == nil && emb.Year != nil {
			searchYear = emb.Year
		}
		var lastErr error
		for _, candidate := range scanner.MovieSearchCandidates(m.StoragePath, m.Title, &emb) {
			tmdbInfo, err = h.searchMovie(ctx, candidate, searchYear, durationSec)
			if err == nil {
				break
			}
			lastErr = err
			if searchYear != nil {
				tmdbInfo, err = h.searchMovie(ctx, candidate, nil, durationSec)
				if err == nil {
					break
				}
				lastErr = err
			}
		}
		if tmdbInfo == nil && lastErr != nil {
			err = lastErr
		}
	} else {
		err = nil
	}

	if err != nil {
		_ = h.mediaRepo.UpdateScrapeStatus(ctx, mid.String(), string(common.ScrapeStatusFailed), err.Error())
		return fmt.Errorf("TMDB 刮削: %w", err)
	}

	// 4. 合并元数据
	h.applyTMDB(m, tmdbInfo)
	h.fillMissingArtwork(ctx, m, emb.Season)
	if m.PosterURL == "" && savedPoster != "" {
		m.PosterURL = savedPoster
	}
	if m.BackdropURL == "" && savedBackdrop != "" {
		m.BackdropURL = savedBackdrop
	}

	// 5. 写入 ffprobe 结果
	if probeResult != nil {
		mi := probeResult.Extract()
		width, height := probeVideoSize(probeResult)
		_ = h.mediaRepo.ApplyProbeToFile(ctx, probePath, repository.FileProbeInfo{
			Duration:    mi.Duration,
			VideoCodec:  mi.VideoCodec,
			AudioCodec:  mi.AudioCodec,
			Resolution:  mi.Resolution,
			HasSubtitle: mi.HasSubtitle,
			BitRate:     mi.BitRate,
		}, width, height)
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

	if err := h.mediaRepo.ApplyScrapeResult(ctx, m); err != nil {
		return fmt.Errorf("更新媒资: %w", err)
	}

	if h.enricher != nil {
		if err := h.enricher.EnrichFromTMDB(ctx, m); err != nil {
			logger.Warn("目录元数据扩展失败", "media_id", mid, "err", err)
		}
	}

	logger.Info("刮削成功", "media_id", mid, "title", m.Title)
	h.invalidateFeedAfterScrape(ctx)
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

func (h *Handlers) probePathForMedia(ctx context.Context, mediaID string, m *media.Media) string {
	probePath := m.StoragePath
	if m.IsTV() {
		if epPath, err := h.mediaRepo.GetFirstEpisodeFilePath(ctx, mediaID); err == nil && epPath != "" {
			probePath = epPath
		}
	}
	return probePath
}

func (h *Handlers) scrapeByIMDB(ctx context.Context, imdbID string, preferTV bool) (*scraperResult, error) {
	found, err := h.tmdb.FindByIMDBID(ctx, imdbID)
	if err != nil {
		return nil, err
	}
	if preferTV && len(found.TVResults) > 0 {
		t, err := h.tmdb.GetTVShow(ctx, found.TVResults[0].ID)
		if err != nil {
			return nil, err
		}
		return h.tvToResultNoSeason(t), nil
	}
	if len(found.MovieResults) > 0 {
		m, err := h.tmdb.GetMovie(ctx, found.MovieResults[0].ID)
		if err != nil {
			return nil, err
		}
		return h.movieToResult(m), nil
	}
	if len(found.TVResults) > 0 {
		t, err := h.tmdb.GetTVShow(ctx, found.TVResults[0].ID)
		if err != nil {
			return nil, err
		}
		return h.tvToResultNoSeason(t), nil
	}
	return nil, fmt.Errorf("TMDB 未找到 IMDB: %s", imdbID)
}

func (h *Handlers) scrapeByTMDBID(ctx context.Context, tmdbID int, preferTV bool, season *int) (*scraperResult, error) {
	if preferTV {
		t, err := h.tmdb.GetTVShow(ctx, tmdbID)
		if err != nil {
			return nil, err
		}
		if season != nil && *season > 0 {
			if s, err := h.tmdb.GetSeason(ctx, tmdbID, *season); err == nil && s != nil {
				return h.tvToResult(t, s), nil
			}
		}
		return h.tvToResultNoSeason(t), nil
	}
	m, err := h.tmdb.GetMovie(ctx, tmdbID)
	if err != nil {
		return nil, err
	}
	return h.movieToResult(m), nil
}

func (h *Handlers) searchMovie(ctx context.Context, title string, year *int, durationSec int) (*scraperResult, error) {
	res, err := h.tmdb.SearchMovie(ctx, title, year)
	if err != nil {
		return nil, err
	}
	if len(res.Results) == 0 {
		return nil, fmt.Errorf("TMDB 未找到电影: %s", title)
	}
	pick := h.pickMovieResult(ctx, res.Results, durationSec)
	m, err := h.tmdb.GetMovie(ctx, pick)
	if err != nil {
		return nil, err
	}
	return h.movieToResult(m), nil
}

func (h *Handlers) pickMovieResult(ctx context.Context, results []scraper.SearchEntry, durationSec int) int {
	if len(results) == 0 {
		return 0
	}
	if durationSec <= 0 || len(results) == 1 {
		return results[0].ID
	}
	limit := len(results)
	if limit > 5 {
		limit = 5
	}
	bestID := results[0].ID
	bestDiff := durationSec + 1
	for i := 0; i < limit; i++ {
		m, err := h.tmdb.GetMovie(ctx, results[i].ID)
		if err != nil || m.Runtime <= 0 {
			continue
		}
		diff := absInt(m.Runtime*60 - durationSec)
		if diff < bestDiff {
			bestDiff = diff
			bestID = results[i].ID
		}
	}
	if bestDiff <= 180 {
		return bestID
	}
	return results[0].ID
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (h *Handlers) searchMovieCandidates(ctx context.Context, m *media.Media, emb *scanner.EmbeddedMeta, durationSec int) (*scraperResult, error) {
	folder := filepath.Base(m.StoragePath)
	title := scanner.SeriesFolderTitle(folder)
	if title == "" {
		title = m.Title
	}
	searchYear := m.Year
	if searchYear == nil {
		searchYear = scanner.SeriesFolderYear(folder)
	}
	if searchYear == nil && emb != nil && emb.Year != nil {
		searchYear = emb.Year
	}
	var lastErr error
	for _, candidate := range scanner.MovieSearchCandidates(m.StoragePath, title, emb) {
		if info, err := h.searchMovie(ctx, candidate, searchYear, durationSec); err == nil {
			return info, nil
		} else {
			lastErr = err
		}
		if searchYear != nil {
			if info, err := h.searchMovie(ctx, candidate, nil, durationSec); err == nil {
				return info, nil
			} else {
				lastErr = err
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("TMDB 未找到电影")
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
		if sp := h.tmdb.PosterURL(s.PosterPath, "w500"); sp != "" {
			r.PosterURL = sp
		}
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

// fillMissingArtwork 海报/背景为空时回退季图或主剧条目（如 TMDB 分季条目无图）
func (h *Handlers) fillMissingArtwork(ctx context.Context, m *media.Media, season *int) {
	if m.TMDBID == nil || *m.TMDBID <= 0 {
		return
	}
	if m.PosterURL != "" && m.BackdropURL != "" {
		return
	}
	sn := season
	if sn == nil && m.IsTV() {
		parsed := scanner.ParseFilePath(h.probePathForMedia(ctx, m.ID.String(), m))
		sn = parsed.Season
	}
	poster, backdrop := h.tmdb.ResolveMissingArtwork(ctx, m.IsTV(), *m.TMDBID, m.Title, sn, m.Year)
	if m.PosterURL == "" {
		m.PosterURL = poster
	}
	if m.BackdropURL == "" {
		m.BackdropURL = backdrop
	}
}

func (h *Handlers) applyTMDB(m *media.Media, info *scraperResult) {
	if info.TMDBID > 0 {
		id := info.TMDBID
		m.TMDBID = &id
	}
	if !media.HasTag(m.Tags, media.TagManualTitle) {
		if info.Title != "" {
			m.Title = info.Title
		}
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
		deps := scanner.IngestDeps{MediaRepo: h.mediaRepo, Queue: h.queue}
		if _, err := scanner.IngestMediaFile(ctx, deps, ev.Path); err != nil {
			logger.Warn("入库失败", "path", ev.Path, "err", err)
			return
		}
		count++
		logger.Info("媒资入库", "path", ev.Path)
	})

	scanCtx, cancel := context.WithTimeout(ctx, 30*60*1e9) // 30 分钟
	defer cancel()
	s.Start(scanCtx)

	logger.Info("扫描完成", "count", count)
	return nil
}

// ---- helpers ----

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

func probeVideoSize(result *scanner.ProbeResult) (width, height int) {
	if result == nil {
		return 0, 0
	}
	for _, s := range result.Streams {
		if s.CodecType == "video" {
			return s.Width, s.Height
		}
	}
	return 0, 0
}

// FileSize 自动估算（无 ffprobe 时）
func estimateFileSize(bitrate int64, duration int) int64 {
	if bitrate <= 0 || duration <= 0 {
		return 0
	}
	return bitrate / 8 * int64(duration)
}
