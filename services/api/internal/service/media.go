// Package service 是业务逻辑层（应用层）
//
// 设计原则：
//   - service 编排多个 repo、queue、外部客户端
//   - 不直接返回 GORM 模型，转换成 DTO
//   - 业务规则（如"评分 < 3 不能转码"）放在这里
package service

import (
	"context"
	"sort"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// MediaService 媒资业务
type MediaService struct {
	repo  *repository.MediaRepo
	queue *queue.Queue
}

// NewMediaService 构造
func NewMediaService(repo *repository.MediaRepo, q *queue.Queue) *MediaService {
	return &MediaService{repo: repo, queue: q}
}

// ListDTO 媒资列表响应
type ListDTO struct {
	Items []MediaSummary `json:"items"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"page_size"`
}

// MediaSummary 媒资摘要（列表用）
type MediaSummary struct {
	ID            uuid.UUID        `json:"id"`
	Title         string           `json:"title"`
	OriginalTitle string           `json:"original_title,omitempty"`
	Year          *int             `json:"year,omitempty"`
	Type          common.MediaType `json:"type"`
	Rating        float64          `json:"rating"`
	PosterURL     string           `json:"poster_url,omitempty"`
	BackdropURL   string           `json:"backdrop_url,omitempty"`
	Genres        []string         `json:"genres"`
	HasSubtitle   bool             `json:"has_subtitle"`
	ScrapeStatus  common.ScrapeStatus `json:"scrape_status,omitempty"`
	ScrapeError   string           `json:"scrape_error,omitempty"`
	EpisodeCount  *int             `json:"episode_count,omitempty"`
}

// List 列表
func (s *MediaService) List(ctx context.Context, f repository.MediaFilter, p common.Pagination) (*ListDTO, error) {
	p.Normalize()
	items, total, err := s.repo.List(ctx, f, p.PageSize, p.Offset())
	if err != nil {
		return nil, err
	}

	out := make([]MediaSummary, len(items))
	seriesIDs := make([]uuid.UUID, 0, len(items))
	for i, m := range items {
		out[i] = toSummary(&m)
		if m.IsTV() {
			seriesIDs = append(seriesIDs, m.ID)
		}
	}
	if counts, err := s.repo.CountPlayableEpisodesByMediaIDs(ctx, seriesIDs); err == nil {
		for i := range out {
			if c, ok := counts[out[i].ID]; ok && c > 0 {
				out[i].EpisodeCount = &c
			}
		}
	}
	return &ListDTO{
		Items: out,
		Total: total,
		Page:  p.Page,
		Size:  p.PageSize,
	}, nil
}

// Detail 详情
func (s *MediaService) Detail(ctx context.Context, id string) (*MediaDetail, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDetail(m), nil
}

// CreateManual 手动创建（CMS 后台手动入库）
func (s *MediaService) CreateManual(ctx context.Context, m *media.Media) error {
	if m.Title == "" {
		return apperr.Validation(map[string]string{"title": "标题不能为空"})
	}
	if m.StoragePath == "" {
		return apperr.Validation(map[string]string{"storage_path": "存储路径不能为空"})
	}
	m.ScrapeStatus = common.ScrapeStatusPending
	if err := s.repo.Create(ctx, m); err != nil {
		return err
	}

	// 异步刮削
	if s.queue != nil {
		_ = s.queue.EnqueueScrape(ctx, m.ID.String())
	}
	return nil
}

// Rescan 重新刮削
func (s *MediaService) Rescan(ctx context.Context, id string) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return err
	}
	if err := s.repo.UpdateScrapeStatus(ctx, id, string(common.ScrapeStatusPending), ""); err != nil {
		return err
	}
	if s.queue != nil {
		_ = s.queue.EnqueueScrape(ctx, id)
	}
	return nil
}

// BatchRescanResult 批量重试结果
type BatchRescanResult struct {
	Queued int `json:"queued"`
}

// BatchRescan 批量重新刮削（按 ID 列表或 scrape_status 筛选）
func (s *MediaService) BatchRescan(ctx context.Context, ids []string, scrapeStatus string) (*BatchRescanResult, error) {
	targets := ids
	if len(targets) == 0 {
		if scrapeStatus == "" {
			return nil, apperr.Validation(map[string]string{"ids": "ids 或 scrape_status 至少指定一项"})
		}
		items, _, err := s.repo.List(ctx, repository.MediaFilter{ScrapeStatus: scrapeStatus}, 5000, 0)
		if err != nil {
			return nil, err
		}
		targets = make([]string, len(items))
		for i, m := range items {
			targets[i] = m.ID.String()
		}
	}

	queued := 0
	for _, id := range targets {
		if err := s.Rescan(ctx, id); err != nil {
			continue
		}
		queued++
	}
	return &BatchRescanResult{Queued: queued}, nil
}

// Update 编辑媒资
func (s *MediaService) Update(ctx context.Context, m *media.Media) error {
	if m.Title == "" {
		return apperr.Validation(map[string]string{"title": "标题不能为空"})
	}
	return s.repo.Update(ctx, m)
}

// Delete 删除
func (s *MediaService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Stats 统计
func (s *MediaService) Stats(ctx context.Context) (*repository.Stats, error) {
	return s.repo.Stats(ctx)
}

// ---- DTO 转换 ----

func toSummary(m *media.Media) MediaSummary {
	genres := m.Genres
	if genres == nil {
		genres = media.StringArray{}
	}
	return MediaSummary{
		ID:            m.ID,
		Title:         m.Title,
		OriginalTitle: m.OriginalTitle,
		Year:          m.Year,
		Type:          m.Type,
		Rating:        m.Rating,
		PosterURL:     m.PosterURL,
		BackdropURL:   m.BackdropURL,
		Genres:        genres,
		HasSubtitle:   m.HasSubtitle,
		ScrapeStatus:  m.ScrapeStatus,
		ScrapeError:   m.ScrapeError,
	}
}

// MediaDetail 媒资详情（含季/集）
type MediaDetail struct {
	*media.Media
	Seasons []SeasonDetail `json:"seasons,omitempty"`
	// 同系列（仅电影有值）
	Collection *CollectionInfo `json:"collection,omitempty"`
}

// CollectionInfo 同系列信息
type CollectionInfo struct {
	ID        int              `json:"id"`
	Name      string           `json:"name"`
	PosterURL string           `json:"poster_url"`
	Parts     []CollectionPart `json:"parts,omitempty"`
}

// CollectionPart 同系列中的单部电影
type CollectionPart struct {
	TMDBID    int     `json:"tmdb_id"`
	Title     string  `json:"title"`
	Year      *int    `json:"year,omitempty"`
	PosterURL string  `json:"poster_url,omitempty"`
	Rating    float64 `json:"rating"`
}

// SeasonDetail 季详情
type SeasonDetail struct {
	ID            uuid.UUID `json:"id"`
	SeasonNumber  int       `json:"season_number"`
	Title         string    `json:"title,omitempty"`
	Overview      string    `json:"overview,omitempty"`
	PosterURL     string    `json:"poster_url,omitempty"`
	AirDate       string    `json:"air_date,omitempty"`
	EpisodeCount  int       `json:"episode_count"`
	Episodes      []EpisodeDetail `json:"episodes,omitempty"`
}

// EpisodeDetail 集详情
type EpisodeDetail struct {
	ID            uuid.UUID `json:"id"`
	EpisodeNumber int       `json:"episode_number"`
	Title         string    `json:"title,omitempty"`
	Overview      string    `json:"overview,omitempty"`
	Duration      int       `json:"duration"`
	StillURL      string    `json:"still_url,omitempty"`
	FilePath      string    `json:"file_path,omitempty"`
}

func toDetail(m *media.Media) *MediaDetail {
	if !m.IsTV() && m.PlayablePath() != "" {
		m.StoragePath = m.PlayablePath()
	}
	d := &MediaDetail{Media: m}
	seasons := append([]media.Season(nil), m.Seasons...)
	sort.Slice(seasons, func(i, j int) bool {
		return seasons[i].SeasonNumber < seasons[j].SeasonNumber
	})
	for _, s := range seasons {
		sd := SeasonDetail{
			ID:           s.ID,
			SeasonNumber: s.SeasonNumber,
			Title:        s.Title,
			Overview:     s.Overview,
			PosterURL:    s.PosterURL,
			EpisodeCount: s.EpisodeCount,
		}
		if s.AirDate != nil {
			sd.AirDate = s.AirDate.Format("2006-01-02")
		}
		eps := append([]media.Episode(nil), s.Episodes...)
		sort.Slice(eps, func(i, j int) bool {
			return eps[i].EpisodeNumber < eps[j].EpisodeNumber
		})
		for _, e := range eps {
			sd.Episodes = append(sd.Episodes, EpisodeDetail{
				ID:            e.ID,
				EpisodeNumber: e.EpisodeNumber,
				Title:         e.Title,
				Overview:      e.Overview,
				Duration:      e.Duration,
				StillURL:      e.StillURL,
				FilePath:      e.PlayablePath(),
			})
		}
		d.Seasons = append(d.Seasons, sd)
	}
	return d
}
