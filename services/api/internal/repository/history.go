package repository

import (
	"context"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/history"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HistoryRepo 历史仓储
type HistoryRepo struct {
	db *gorm.DB
}

// NewHistoryRepo 构造历史仓储
func NewHistoryRepo(db *gorm.DB) *HistoryRepo {
	return &HistoryRepo{db: db}
}

// UpsertHistory 更新或插入历史
func (r *HistoryRepo) UpsertHistory(ctx context.Context, h *history.History) error {
	// 用 (profile_id, media_id, episode_id) 作为唯一键做 upsert
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Model(&history.History{}).
			Where("profile_id = ? AND media_id = ?", h.ProfileID, h.MediaID)
		if h.EpisodeID != nil {
			q = q.Where("episode_id = ?", *h.EpisodeID)
		} else {
			q = q.Where("episode_id IS NULL")
		}

		var existing history.History
		err := q.First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			// insert
			if err := tx.Create(h).Error; err != nil {
				return apperr.Wrap(err, apperr.CodeInternal, "记录历史失败")
			}
			return nil
		}
		if err != nil {
			return apperr.Wrap(err, apperr.CodeInternal, "查询历史失败")
		}

		// update
		existing.Progress = h.Progress
		existing.Duration = h.Duration
		existing.Completed = h.Completed
		existing.Device = h.Device
		existing.EpisodeID = h.EpisodeID
		if err := tx.Save(&existing).Error; err != nil {
			return apperr.Wrap(err, apperr.CodeInternal, "更新历史失败")
		}
		*h = existing
		return nil
	})
}

// ListByProfile 按 Profile 拉取历史
func (r *HistoryRepo) ListByProfile(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	var hs []history.History
	if err := r.db.WithContext(ctx).
		Preload("Media").
		Where("profile_id = ?", profileID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&hs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询历史失败")
	}
	return hs, nil
}

// ListInProgress 在看（未完成）
func (r *HistoryRepo) ListInProgress(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	var hs []history.History
	if err := r.db.WithContext(ctx).
		Preload("Media").
		Where("profile_id = ? AND completed = ? AND progress > 0", profileID, false).
		Order("updated_at DESC").
		Limit(limit).
		Find(&hs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询历史失败")
	}
	return hs, nil
}

// ---- Favorite ----

// ToggleFavorite 切换收藏
func (r *HistoryRepo) ToggleFavorite(ctx context.Context, profileID, mediaID string, favType common.FavoriteType, rating *float64) (bool, error) {
	var fav history.Favorite
	err := r.db.WithContext(ctx).
		Where("profile_id = ? AND media_id = ?", profileID, mediaID).
		First(&fav).Error

	if err == gorm.ErrRecordNotFound {
		// 创建
		fav = history.Favorite{
			ProfileID:    uuid.MustParse(profileID),
			MediaID:      uuid.MustParse(mediaID),
			FavoriteType: favType,
			Rating:       rating,
		}
		if err := r.db.WithContext(ctx).Create(&fav).Error; err != nil {
			return false, apperr.Wrap(err, apperr.CodeInternal, "收藏失败")
		}
		return true, nil
	}
	if err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "查询收藏失败")
	}

	// 已存在则删除（toggle）
	if err := r.db.WithContext(ctx).Delete(&fav).Error; err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "取消收藏失败")
	}
	return false, nil
}

// ListFavorites 按 Profile 列出收藏
func (r *HistoryRepo) ListFavorites(ctx context.Context, profileID string, favType string) ([]history.Favorite, error) {
	var fs []history.Favorite
	q := r.db.WithContext(ctx).
		Preload("Media").
		Where("profile_id = ?", profileID)
	if favType != "" {
		q = q.Where("favorite_type = ?", favType)
	}
	if err := q.Order("created_at DESC").Find(&fs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询收藏失败")
	}
	return fs, nil
}

// IsFavorited 是否已收藏
func (r *HistoryRepo) IsFavorited(ctx context.Context, profileID, mediaID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&history.Favorite{}).
		Where("profile_id = ? AND media_id = ?", profileID, mediaID).
		Count(&count).Error; err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "查询收藏失败")
	}
	return count > 0, nil
}
