package repository

import (
	"context"
	"errors"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"

	"gorm.io/gorm"
)

// M3UPlaylistView M3U 来源及同步配置
type M3UPlaylistView struct {
	URL             string     `json:"url"`
	Count           int64      `json:"count"`
	SyncEnabled     bool       `json:"sync_enabled"`
	IntervalMinutes int        `json:"interval_minutes"`
	LastSyncAt      *time.Time `json:"last_sync_at,omitempty"`
	LastSyncStatus  string     `json:"last_sync_status,omitempty"`
	LastSyncMessage string     `json:"last_sync_message,omitempty"`
}

// LiveM3USyncRepo M3U 同步配置仓储
type LiveM3USyncRepo struct {
	db *gorm.DB
}

func NewLiveM3USyncRepo(db *gorm.DB) *LiveM3USyncRepo {
	return &LiveM3USyncRepo{db: db}
}

func (r *LiveM3USyncRepo) GetByPlaylistURL(ctx context.Context, url string) (*live.M3USyncJob, error) {
	var job live.M3USyncJob
	err := r.db.WithContext(ctx).First(&job, "playlist_url = ?", url).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("同步配置不存在")
	}
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询同步配置失败")
	}
	return &job, nil
}

func (r *LiveM3USyncRepo) Upsert(ctx context.Context, job *live.M3USyncJob) error {
	var existing live.M3USyncJob
	err := r.db.WithContext(ctx).First(&existing, "playlist_url = ?", job.PlaylistURL).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(job).Error
	}
	if err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "查询同步配置失败")
	}
	existing.Enabled = job.Enabled
	existing.IntervalMinutes = job.IntervalMinutes
	return r.db.WithContext(ctx).Save(&existing).Error
}

func (r *LiveM3USyncRepo) EnsureDefault(ctx context.Context, playlistURL string) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&live.M3USyncJob{}).
		Where("playlist_url = ?", playlistURL).Count(&count).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "查询同步配置失败")
	}
	if count > 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&live.M3USyncJob{
		PlaylistURL:     playlistURL,
		Enabled:         true,
		IntervalMinutes: 1440,
	}).Error
}

func (r *LiveM3USyncRepo) UpdateSyncResult(ctx context.Context, playlistURL string, status live.M3USyncStatus, message string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&live.M3USyncJob{}).
		Where("playlist_url = ?", playlistURL).
		Updates(map[string]any{
			"last_sync_at":      now,
			"last_sync_status":  status,
			"last_sync_message": message,
			"updated_at":        now,
		}).Error
}

func (r *LiveM3USyncRepo) ListDue(ctx context.Context, now time.Time) ([]live.M3USyncJob, error) {
	var jobs []live.M3USyncJob
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&jobs).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询同步任务失败")
	}
	out := make([]live.M3USyncJob, 0, len(jobs))
	for _, job := range jobs {
		if job.LastSyncAt == nil {
			out = append(out, job)
			continue
		}
		next := job.LastSyncAt.Add(time.Duration(job.IntervalMinutes) * time.Minute)
		if !now.Before(next) {
			out = append(out, job)
		}
	}
	return out, nil
}

func (r *LiveRepo) ListPlaylistsWithSync(ctx context.Context, syncRepo *LiveM3USyncRepo) ([]M3UPlaylistView, error) {
	stats, err := r.ListPlaylists(ctx)
	if err != nil {
		return nil, err
	}
	jobMap := make(map[string]live.M3USyncJob)
	var jobs []live.M3USyncJob
	if err := syncRepo.db.WithContext(ctx).Find(&jobs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询同步配置失败")
	}
	for _, j := range jobs {
		jobMap[j.PlaylistURL] = j
	}
	out := make([]M3UPlaylistView, len(stats))
	for i, s := range stats {
		v := M3UPlaylistView{
			URL:             s.URL,
			Count:           s.Count,
			IntervalMinutes: 1440,
		}
		if job, ok := jobMap[s.URL]; ok {
			v.SyncEnabled = job.Enabled
			v.IntervalMinutes = job.IntervalMinutes
			v.LastSyncAt = job.LastSyncAt
			v.LastSyncStatus = string(job.LastSyncStatus)
			v.LastSyncMessage = job.LastSyncMessage
		}
		out[i] = v
	}
	return out, nil
}
