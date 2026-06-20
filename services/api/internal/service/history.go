package service

import (
	"context"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/history"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// HistoryService 历史与收藏业务
type HistoryService struct {
	repo *repository.HistoryRepo
}

// NewHistoryService 构造
func NewHistoryService(repo *repository.HistoryRepo) *HistoryService {
	return &HistoryService{repo: repo}
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
	return s.repo.UpsertHistory(ctx, h)
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

// GetResumePoint 获取某媒资的续播位置
func (s *HistoryService) GetResumePoint(ctx context.Context, profileID, mediaID string) (*history.History, error) {
	return s.repo.GetLatestByMedia(ctx, profileID, mediaID)
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
