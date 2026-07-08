package service

import (
	"context"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/history"
	"github.com/mediahub/api/internal/domain/user"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// HistoryService 历史与收藏业务
type HistoryService struct {
	repo     *repository.HistoryRepo
	profiles *repository.UserRepo
}

// NewHistoryService 构造
func NewHistoryService(repo *repository.HistoryRepo, profiles *repository.UserRepo) *HistoryService {
	return &HistoryService{repo: repo, profiles: profiles}
}

// RecordProgress 记录播放进度
type RecordProgress struct {
	ProfileID string `json:"profile_id" binding:"required"`
	MediaID   string `json:"media_id" binding:"required"`
	EpisodeID string `json:"episode_id,omitempty"`
	Progress  int    `json:"progress" binding:"gte=0"`
	Duration  int    `json:"duration" binding:"gte=0"`
	Device    string `json:"device"`
}

// RecordProgress 记录进度
func (s *HistoryService) RecordProgress(ctx context.Context, req RecordProgress) error {
	pid, err := uuid.Parse(req.ProfileID)
	if err != nil {
		return apperr.Validation(map[string]string{"profile_id": "格式错误"})
	}
	mid, err := uuid.Parse(req.MediaID)
	if err != nil {
		return apperr.Validation(map[string]string{"media_id": "格式错误"})
	}

	h := &history.History{
		ProfileID: pid,
		MediaID:   mid,
		Progress:  req.Progress,
		Duration:  req.Duration,
		Completed: req.Duration > 0 && req.Progress >= req.Duration*95/100, // 95% 视为完成
		Device:    req.Device,
	}
	if req.EpisodeID != "" {
		eid, err := uuid.Parse(req.EpisodeID)
		if err != nil {
			return apperr.Validation(map[string]string{"episode_id": "格式错误"})
		}
		h.EpisodeID = &eid
	}

	pp := &history.PlaybackProgress{
		ProfileID:   pid,
		MediaID:     mid,
		PositionSec: req.Progress,
		DurationSec: req.Duration,
		Device:      req.Device,
	}
	if h.EpisodeID != nil {
		pp.EpisodeID = h.EpisodeID
	}

	if s.profiles != nil {
		if _, err := s.profiles.GetProfile(ctx, req.ProfileID); err != nil {
			if ae, ok := apperr.As(err); ok && ae.Code == apperr.CodeNotFound {
				return apperr.BadRequest("Profile 不存在，请刷新页面或重新选择成员")
			}
			return err
		}
	}

	if err := s.repo.UpsertHistory(ctx, h); err != nil {
		return err
	}
	return s.repo.UpsertPlaybackProgress(ctx, pp)
}

// GetHistory 获取历史
func (s *HistoryService) GetHistory(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListByProfile(ctx, profileID, limit)
}

// GetContinueWatching 继续观看（未完成）
func (s *HistoryService) GetContinueWatching(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	if limit <= 0 {
		limit = 12
	}
	return s.repo.ListInProgress(ctx, profileID, limit)
}

// GetResumePoint 获取某媒资的续播位置；剧集可传 episodeID 精确到单集
func (s *HistoryService) GetResumePoint(ctx context.Context, profileID, mediaID, episodeID string) (*history.History, error) {
	// 优先返回跨设备续播表 playback_progress，历史表 history 作为兼容兜底。
	if p, err := s.repo.GetPlaybackResumePoint(ctx, profileID, mediaID, episodeID); err != nil {
		return nil, err
	} else if p != nil {
		return p, nil
	}
	if episodeID != "" {
		return s.repo.GetLatestByMediaEpisode(ctx, profileID, mediaID, episodeID)
	}
	return s.repo.GetLatestByMedia(ctx, profileID, mediaID)
}

// DefaultWebProfile 获取 Web 播放端默认 Profile
func (s *HistoryService) DefaultWebProfile(ctx context.Context) (*user.Profile, error) {
	if s.profiles == nil {
		return nil, apperr.Internal("Profile 服务未配置")
	}
	return s.profiles.GetProfile(ctx, DefaultWebProfileID)
}

// ToggleFavorite 切换收藏
func (s *HistoryService) ToggleFavorite(ctx context.Context, profileID, mediaID string, favType common.FavoriteType, rating *float64) (bool, error) {
	if favType == "" {
		favType = common.FavWant
	}
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return false, apperr.Validation(map[string]string{"profile_id": "格式错误"})
	}
	mid, err := uuid.Parse(mediaID)
	if err != nil {
		return false, apperr.Validation(map[string]string{"media_id": "格式错误"})
	}
	_ = pid
	_ = mid
	return s.repo.ToggleFavorite(ctx, profileID, mediaID, favType, rating)
}

// ListFavorites 列出收藏
func (s *HistoryService) ListFavorites(ctx context.Context, profileID, favType string) ([]history.Favorite, error) {
	return s.repo.ListFavorites(ctx, profileID, favType)
}

// AddWantTMDB 加入 TMDB 想看
func (s *HistoryService) AddWantTMDB(ctx context.Context, profileID string, req AddWantTMDBRequest) (bool, error) {
	return s.repo.AddWantTMDB(ctx, profileID, req.TMDBID, req.Type, req.Title, req.PosterURL, req.Year)
}

// ToggleWantTMDB 切换 TMDB 想看
func (s *HistoryService) ToggleWantTMDB(ctx context.Context, profileID string, req AddWantTMDBRequest) (bool, error) {
	return s.repo.ToggleWantTMDB(ctx, profileID, req.TMDBID, req.Type, req.Title, req.PosterURL, req.Year)
}

// IsWantTMDB 是否 TMDB 想看
func (s *HistoryService) IsWantTMDB(ctx context.Context, profileID string, tmdbID int) (bool, error) {
	return s.repo.IsWantTMDB(ctx, profileID, tmdbID)
}

func (s *HistoryService) RemoveWantTMDB(ctx context.Context, profileID string, tmdbID int) error {
	return s.repo.RemoveWantTMDB(ctx, profileID, tmdbID)
}

// ListAllWants 全部想看
func (s *HistoryService) ListAllWants(ctx context.Context, limit int) ([]history.Favorite, error) {
	return s.repo.ListAllWants(ctx, limit)
}

// AddWantTMDBRequest 库外 TMDB 想看
type AddWantTMDBRequest struct {
	TMDBID    int    `json:"tmdb_id" binding:"required"`
	Type      string `json:"type" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Year      *int   `json:"year"`
	PosterURL string `json:"poster_url"`
}
