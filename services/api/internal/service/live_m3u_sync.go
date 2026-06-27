package service

import (
	"context"
	"sync"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"

	"github.com/google/uuid"
)

// UpdateM3USyncConfigRequest 更新 M3U 同步配置
type UpdateM3USyncConfigRequest struct {
	PlaylistURL     string `json:"playlist_url" binding:"required"`
	Enabled         bool   `json:"enabled"`
	IntervalMinutes int    `json:"interval_minutes"`
}


func (s *LiveService) ListPlaylistsWithSync(ctx context.Context) ([]repository.M3UPlaylistView, error) {
	if s.syncRepo == nil {
		return nil, apperr.Validation("同步功能未初始化")
	}
	return s.repo.ListPlaylistsWithSync(ctx, s.syncRepo)
}

func (s *LiveService) UpdateM3USyncConfig(ctx context.Context, req UpdateM3USyncConfigRequest) (*repository.M3UPlaylistView, error) {
	if !s.config.Enabled {
		return nil, apperr.Validation("直播功能未启用")
	}
	playlistURL, err := validateM3UPlaylistURL(req.PlaylistURL)
	if err != nil {
		return nil, err
	}
	stats, err := s.repo.ListPlaylists(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	var count int64
	for _, st := range stats {
		if st.URL == playlistURL {
			found = true
			count = st.Count
			break
		}
	}
	if !found {
		return nil, apperr.Validation("该 M3U 来源尚未导入频道，请先导入")
	}

	interval := live.NormalizeSyncInterval(req.IntervalMinutes)
	job := &live.M3USyncJob{
		PlaylistURL:     playlistURL,
		Enabled:         req.Enabled,
		IntervalMinutes: interval,
	}
	if err := s.syncRepo.Upsert(ctx, job); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "保存同步配置失败")
	}
	return &repository.M3UPlaylistView{
		URL:             playlistURL,
		Count:           count,
		SyncEnabled:     req.Enabled,
		IntervalMinutes: interval,
	}, nil
}

var m3uSyncMu sync.Map

func (s *LiveService) runScheduledM3USync(ctx context.Context, job live.M3USyncJob) {
	key := job.PlaylistURL
	if _, loaded := m3uSyncMu.LoadOrStore(key, true); loaded {
		return
	}
	defer m3uSyncMu.Delete(key)

	logger.Info("开始自动同步 M3U", "url", job.PlaylistURL)
	result, err := s.SyncM3U(ctx, job.PlaylistURL, uuid.Nil)
	if err != nil {
		logger.Warn("M3U 自动同步失败", "url", job.PlaylistURL, "err", err)
		return
	}
	logger.Info("M3U 自动同步完成", "url", job.PlaylistURL, "created", result.Created, "skipped", result.Skipped)
}

func (s *LiveService) tickM3USync(ctx context.Context) {
	if s.syncRepo == nil || !s.config.Enabled {
		return
	}
	jobs, err := s.syncRepo.ListDue(ctx, time.Now())
	if err != nil {
		logger.Debug("查询 M3U 同步任务失败", "err", err)
		return
	}
	for _, job := range jobs {
		s.runScheduledM3USync(ctx, job)
	}
}

// StartM3USyncWatcher 启动 M3U 自动同步（每分钟检查一次）
func (s *LiveService) StartM3USyncWatcher(ctx context.Context) {
	if s.syncRepo == nil {
		return
	}
	logger.Info("M3U 自动同步已启动", "check_interval", "1m")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickM3USync(ctx)
		}
	}
}
