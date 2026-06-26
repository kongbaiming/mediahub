package service

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/internal/scraper"
)

// ScrapeCandidate TMDB 刮削候选
type ScrapeCandidate struct {
	TMDBID        int     `json:"tmdb_id"`
	Type          string  `json:"type"` // movie | tv
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title,omitempty"`
	Year          *int    `json:"year,omitempty"`
	Overview      string  `json:"overview,omitempty"`
	PosterURL     string  `json:"poster_url,omitempty"`
	Runtime       *int    `json:"runtime,omitempty"`
	Rating        float64 `json:"rating,omitempty"`
	MatchScore    float64 `json:"match_score"`
}

// ScrapeMatchService 手动匹配 TMDB（刮削失败时 CMS 点选）
type ScrapeMatchService struct {
	media   *repository.MediaRepo
	tmdb    *scraper.TMDBClient
	catalog *CatalogService
}

func NewScrapeMatchService(media *repository.MediaRepo, tmdb *scraper.TMDBClient, catalog *CatalogService) *ScrapeMatchService {
	return &ScrapeMatchService{media: media, tmdb: tmdb, catalog: catalog}
}

// ListCandidates 列出 TMDB 搜索候选（内嵌元数据 + 路径 + 时长消歧）
func (s *ScrapeMatchService) ListCandidates(ctx context.Context, mediaID string) ([]ScrapeCandidate, error) {
	if s.tmdb == nil {
		return nil, apperr.BadRequest("TMDB 未配置")
	}
	m, err := s.media.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	probePath := probeMediaPath(ctx, s.media, m)
	emb := scanner.EmbeddedMeta{}
	if info, probeErr := scanner.Probe(ctx, "", probePath); probeErr == nil {
		emb = scanner.ExtractEmbeddedMeta(info)
	}
	durationSec := emb.DurationSec
	searchYear := m.Year
	if searchYear == nil && emb.Year != nil {
		searchYear = emb.Year
	}
	if searchYear == nil {
		searchYear = scanner.SeriesFolderYear(filepath.Base(m.StoragePath))
	}

	seen := map[string]struct{}{}
	var out []ScrapeCandidate

	add := func(c ScrapeCandidate) {
		key := c.Type + ":" + strconv.Itoa(c.TMDBID)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}

	if emb.IMDBID != "" {
		if found, err := s.tmdb.FindByIMDBID(ctx, emb.IMDBID); err == nil && found != nil {
			for _, e := range found.MovieResults {
				add(s.candidateFromMovieEntry(ctx, e, durationSec, 100))
			}
			for _, e := range found.TVResults {
				add(s.candidateFromTVEntry(ctx, e, 95))
			}
		}
	}

	for _, q := range scanner.MovieSearchCandidates(m.StoragePath, m.Title, &emb) {
		res, err := s.tmdb.SearchMovie(ctx, q, searchYear)
		if err != nil || len(res.Results) == 0 {
			if searchYear != nil {
				res, err = s.tmdb.SearchMovie(ctx, q, nil)
			}
		}
		if err != nil || len(res.Results) == 0 {
			continue
		}
		limit := len(res.Results)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			add(s.candidateFromMovieEntry(ctx, res.Results[i], durationSec, 80-float64(i*5)))
		}
	}

	for _, q := range scanner.TVSearchCandidates(m.StoragePath, m.Title, &emb) {
		res, err := s.tmdb.SearchTV(ctx, q, searchYear)
		if err != nil || len(res.Results) == 0 {
			if searchYear != nil {
				res, err = s.tmdb.SearchTV(ctx, q, nil)
			}
		}
		if err != nil || len(res.Results) == 0 {
			continue
		}
		limit := len(res.Results)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			add(s.candidateFromTVEntry(ctx, res.Results[i], 70-float64(i*5)))
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].MatchScore > out[j].MatchScore
	})
	if len(out) > 15 {
		out = out[:15]
	}
	return out, nil
}

// ApplyMatch 应用用户选中的 TMDB 条目并标记刮削完成
func (s *ScrapeMatchService) ApplyMatch(ctx context.Context, mediaID string, tmdbID int, mediaType string) error {
	if s.tmdb == nil {
		return apperr.BadRequest("TMDB 未配置")
	}
	if tmdbID <= 0 {
		return apperr.Validation(map[string]string{"tmdb_id": "无效"})
	}
	m, err := s.media.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}

	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	switch mediaType {
	case "movie":
		movie, err := s.tmdb.GetMovie(ctx, tmdbID)
		if err != nil {
			return apperr.Wrap(err, apperr.CodeBadRequest, "TMDB 电影不存在")
		}
		m.Type = common.MediaTypeMovie
		m.Kind = media.MediaKindSingle
		if !media.HasTag(m.Tags, media.TagManualTitle) {
			m.Title = movie.Title
			m.OriginalTitle = movie.OriginalTitle
		}
		m.Overview = movie.Overview
		m.PosterURL = s.tmdb.PosterURL(movie.PosterPath, "w500")
		m.BackdropURL = s.tmdb.BackdropURL(movie.BackdropPath, "w1280")
		m.Rating = movie.VoteAverage
		m.VoteCount = movie.VoteCount
		if y := yearFromDate(movie.ReleaseDate); y != nil {
			m.Year = y
		}
		if movie.Runtime > 0 {
			rt := movie.Runtime
			m.Runtime = &rt
		}
		if len(movie.Genres) > 0 {
			m.Genres = genreNames(movie.Genres)
		}
	case "tv", "tvshow":
		tv, err := s.tmdb.GetTVShow(ctx, tmdbID)
		if err != nil {
			return apperr.Wrap(err, apperr.CodeBadRequest, "TMDB 剧集不存在")
		}
		m.Type = common.MediaTypeTVShow
		m.Kind = media.MediaKindSeries
		if !media.HasTag(m.Tags, media.TagManualTitle) {
			m.Title = tv.Name
			m.OriginalTitle = tv.OriginalName
		}
		m.Overview = tv.Overview
		m.PosterURL = s.tmdb.PosterURL(tv.PosterPath, "w500")
		m.BackdropURL = s.tmdb.BackdropURL(tv.BackdropPath, "w1280")
		m.Rating = tv.VoteAverage
		m.VoteCount = tv.VoteCount
		if y := yearFromDate(tv.FirstAirDate); y != nil {
			m.Year = y
		}
		if len(tv.Genres) > 0 {
			m.Genres = genreNames(tv.Genres)
		}
	default:
		return apperr.Validation(map[string]string{"type": "type 须为 movie 或 tv"})
	}

	id := tmdbID
	m.TMDBID = &id
	m.ScrapeStatus = common.ScrapeStatusDone
	m.ScrapeError = ""
	now := time.Now()
	m.LastScrapeAt = &now

	if err := s.media.ApplyScrapeResult(ctx, m); err != nil {
		return err
	}
	if m.PosterURL == "" || m.BackdropURL == "" {
		s.fillMissingArtwork(ctx, m)
		if m.PosterURL != "" || m.BackdropURL != "" {
			_ = s.media.Update(ctx, m)
		}
	}
	if s.catalog != nil {
		_ = s.catalog.EnrichFromTMDB(ctx, m)
	}
	return nil
}

func (s *ScrapeMatchService) fillMissingArtwork(ctx context.Context, m *media.Media) {
	if s.tmdb == nil || m.TMDBID == nil || *m.TMDBID <= 0 {
		return
	}
	if m.PosterURL != "" && m.BackdropURL != "" {
		return
	}
	if m.IsTV() {
		tv, err := s.tmdb.GetTVShow(ctx, *m.TMDBID)
		if err != nil {
			return
		}
		if m.PosterURL == "" {
			m.PosterURL = s.tmdb.PosterURL(tv.PosterPath, "w500")
		}
		if m.BackdropURL == "" {
			m.BackdropURL = s.tmdb.BackdropURL(tv.BackdropPath, "w1280")
		}
		return
	}
	movie, err := s.tmdb.GetMovie(ctx, *m.TMDBID)
	if err != nil {
		return
	}
	if m.PosterURL == "" {
		m.PosterURL = s.tmdb.PosterURL(movie.PosterPath, "w500")
	}
	if m.BackdropURL == "" {
		m.BackdropURL = s.tmdb.BackdropURL(movie.BackdropPath, "w1280")
	}
}

func probeMediaPath(ctx context.Context, repo *repository.MediaRepo, m *media.Media) string {
	p := m.StoragePath
	if m.IsTV() {
		if ep, err := repo.GetFirstEpisodeFilePath(ctx, m.ID.String()); err == nil && ep != "" {
			p = ep
		}
	}
	return p
}

func (s *ScrapeMatchService) candidateFromMovieEntry(ctx context.Context, e scraper.SearchEntry, durationSec int, baseScore float64) ScrapeCandidate {
	title := e.Title
	if title == "" {
		title = e.Name
	}
	c := ScrapeCandidate{
		TMDBID:        e.ID,
		Type:          "movie",
		Title:         title,
		OriginalTitle: e.OriginalTitle,
		Year:          yearFromDate(e.ReleaseDate),
		Overview:      trimOverview(e.Overview),
		PosterURL:     s.tmdb.PosterURL(e.PosterPath, "w500"),
		Rating:        e.VoteAverage,
		MatchScore:    baseScore,
	}
	if detail, err := s.tmdb.GetMovie(ctx, e.ID); err == nil && detail != nil {
		if detail.Runtime > 0 {
			rt := detail.Runtime
			c.Runtime = &rt
			if durationSec > 0 {
				diff := absInt(rt*60 - durationSec)
				if diff <= 180 {
					c.MatchScore += 20
				} else if diff <= 600 {
					c.MatchScore += 5
				}
			}
		}
		if c.Overview == "" {
			c.Overview = trimOverview(detail.Overview)
		}
	}
	return c
}

func (s *ScrapeMatchService) candidateFromTVEntry(ctx context.Context, e scraper.SearchEntry, baseScore float64) ScrapeCandidate {
	title := e.Name
	if title == "" {
		title = e.Title
	}
	return ScrapeCandidate{
		TMDBID:        e.ID,
		Type:          "tv",
		Title:         title,
		OriginalTitle: e.OriginalName,
		Year:          yearFromDate(e.FirstAirDate),
		Overview:      trimOverview(e.Overview),
		PosterURL:     s.tmdb.PosterURL(e.PosterPath, "w500"),
		Rating:        e.VoteAverage,
		MatchScore:    baseScore,
	}
}

func yearFromDate(s string) *int {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return nil
	}
	y, err := strconv.Atoi(s[:4])
	if err != nil || y < 1900 || y > 2100 {
		return nil
	}
	return &y
}

func trimOverview(s string) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > 160 {
		return string([]rune(s)[:160]) + "…"
	}
	return s
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
