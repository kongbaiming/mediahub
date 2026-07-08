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

// GetLatestByMedia 取某媒资最近一条播放记录
func (r *HistoryRepo) GetLatestByMedia(ctx context.Context, profileID, mediaID string) (*history.History, error) {
	var hs []history.History
	err := r.db.WithContext(ctx).
		Where("profile_id = ? AND media_id = ?", profileID, mediaID).
		Order("updated_at DESC").
		Limit(1).
		Find(&hs).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询续播记录失败")
	}
	if len(hs) == 0 {
		return nil, nil
	}
	return &hs[0], nil
}

// GetLatestByMediaEpisode 取某媒资指定单集的最近播放记录
func (r *HistoryRepo) GetLatestByMediaEpisode(ctx context.Context, profileID, mediaID, episodeID string) (*history.History, error) {
	eid, err := uuid.Parse(episodeID)
	if err != nil {
		return nil, apperr.Validation(map[string]string{"episode_id": "格式错误"})
	}
	var hs []history.History
	err = r.db.WithContext(ctx).
		Where("profile_id = ? AND media_id = ? AND episode_id = ?", profileID, mediaID, eid).
		Order("updated_at DESC").
		Limit(1).
		Find(&hs).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询单集续播记录失败")
	}
	if len(hs) == 0 {
		return nil, nil
	}
	return &hs[0], nil
}

// UpsertPlaybackProgress 更新或插入跨设备续播进度
func (r *HistoryRepo) UpsertPlaybackProgress(ctx context.Context, p *history.PlaybackProgress) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Model(&history.PlaybackProgress{}).
			Where("profile_id = ? AND media_id = ?", p.ProfileID, p.MediaID)
		if p.EpisodeID != nil {
			q = q.Where("episode_id = ?", *p.EpisodeID)
		} else {
			q = q.Where("episode_id IS NULL")
		}

		var existing history.PlaybackProgress
		err := q.First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := tx.Create(p).Error; err != nil {
				return apperr.Wrap(err, apperr.CodeInternal, "记录续播进度失败")
			}
			return nil
		}
		if err != nil {
			return apperr.Wrap(err, apperr.CodeInternal, "查询续播进度失败")
		}

		existing.PositionSec = p.PositionSec
		existing.DurationSec = p.DurationSec
		existing.Device = p.Device
		existing.EpisodeID = p.EpisodeID
		if err := tx.Save(&existing).Error; err != nil {
			return apperr.Wrap(err, apperr.CodeInternal, "更新续播进度失败")
		}
		*p = existing
		return nil
	})
}

// GetPlaybackResumePoint 读取跨设备续播进度并映射为历史响应结构
func (r *HistoryRepo) GetPlaybackResumePoint(ctx context.Context, profileID, mediaID, episodeID string) (*history.History, error) {
	q := r.db.WithContext(ctx).
		Where("profile_id = ? AND media_id = ?", profileID, mediaID)
	if episodeID != "" {
		eid, err := uuid.Parse(episodeID)
		if err != nil {
			return nil, apperr.Validation(map[string]string{"episode_id": "格式错误"})
		}
		q = q.Where("episode_id = ?", eid)
	} else {
		q = q.Order("updated_at DESC").Limit(1)
	}

	var rows []history.PlaybackProgress
	if err := q.Find(&rows).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询跨设备续播失败")
	}
	if len(rows) == 0 {
		return nil, nil
	}

	row := rows[0]
	completed := row.DurationSec > 0 && row.PositionSec >= row.DurationSec*95/100
	out := &history.History{
		BaseModel: row.BaseModel,
		ProfileID: row.ProfileID,
		MediaID:   row.MediaID,
		EpisodeID: row.EpisodeID,
		Progress:  row.PositionSec,
		Duration:  row.DurationSec,
		Completed: completed,
		Device:    row.Device,
	}
	return out, nil
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

// ListInProgress 在看（未完成，按媒资去重）
func (r *HistoryRepo) ListInProgress(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	if limit <= 0 {
		limit = 12
	}
	var hs []history.History
	if err := r.db.WithContext(ctx).
		Preload("Media").
		Where("profile_id = ? AND completed = ? AND progress > 0", profileID, false).
		Order("updated_at DESC").
		Limit(limit * 4).
		Find(&hs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询历史失败")
	}
	return dedupeHistoryByMedia(hs, limit), nil
}

func dedupeHistoryByMedia(hs []history.History, limit int) []history.History {
	seen := map[string]struct{}{}
	out := make([]history.History, 0, limit)
	for _, h := range hs {
		key := h.MediaID.String()
		if h.EpisodeID != nil {
			key += ":" + h.EpisodeID.String()
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, h)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// ---- Favorite ----

// ToggleFavorite 切换收藏
func (r *HistoryRepo) ToggleFavorite(ctx context.Context, profileID, mediaID string, favType common.FavoriteType, rating *float64) (bool, error) {
	var fav history.Favorite
	err := r.db.WithContext(ctx).
		Where("profile_id = ? AND media_id = ?", profileID, mediaID).
		First(&fav).Error

	if err == gorm.ErrRecordNotFound {
		mid, err := uuid.Parse(mediaID)
		if err != nil {
			return false, apperr.Validation(map[string]string{"media_id": "格式错误"})
		}
		fav = history.Favorite{
			ProfileID:    uuid.MustParse(profileID),
			MediaID:      &mid,
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

// AddWantTMDB 加入 TMDB 想看（幂等，已存在则跳过）
func (r *HistoryRepo) AddWantTMDB(ctx context.Context, profileID string, tmdbID int, mediaType, title, posterURL string, year *int) (bool, error) {
	ok, err := r.IsWantTMDB(ctx, profileID, tmdbID)
	if err != nil {
		return false, err
	}
	if ok {
		return false, nil
	}
	fav := history.Favorite{
		ProfileID:    uuid.MustParse(profileID),
		TMDBID:       &tmdbID,
		MediaType:    mediaType,
		Title:        title,
		Year:         year,
		PosterURL:    posterURL,
		FavoriteType: common.FavWant,
	}
	if err := r.db.WithContext(ctx).Create(&fav).Error; err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "加入想看失败")
	}
	return true, nil
}

// ToggleWantTMDB 切换 TMDB 库外想看
func (r *HistoryRepo) ToggleWantTMDB(ctx context.Context, profileID string, tmdbID int, mediaType, title, posterURL string, year *int) (bool, error) {
	var fav history.Favorite
	err := r.db.WithContext(ctx).
		Where("profile_id = ? AND tmdb_id = ? AND favorite_type = ? AND media_id IS NULL",
			profileID, tmdbID, common.FavWant).
		First(&fav).Error

	if err == gorm.ErrRecordNotFound {
		fav = history.Favorite{
			ProfileID:    uuid.MustParse(profileID),
			TMDBID:       &tmdbID,
			MediaType:    mediaType,
			Title:        title,
			Year:         year,
			PosterURL:    posterURL,
			FavoriteType: common.FavWant,
		}
		if err := r.db.WithContext(ctx).Create(&fav).Error; err != nil {
			return false, apperr.Wrap(err, apperr.CodeInternal, "加入想看失败")
		}
		return true, nil
	}
	if err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "查询想看失败")
	}

	if err := r.db.WithContext(ctx).Delete(&fav).Error; err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "取消想看失败")
	}
	return false, nil
}

// IsWantTMDB 是否已标记 TMDB 想看
func (r *HistoryRepo) IsWantTMDB(ctx context.Context, profileID string, tmdbID int) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&history.Favorite{}).
		Where("profile_id = ? AND tmdb_id = ? AND favorite_type = ? AND media_id IS NULL",
			profileID, tmdbID, common.FavWant).
		Count(&count).Error; err != nil {
		return false, apperr.Wrap(err, apperr.CodeInternal, "查询想看失败")
	}
	return count > 0, nil
}

// RemoveWantTMDB 取消 TMDB 想看（仅删除，不创建）
func (r *HistoryRepo) RemoveWantTMDB(ctx context.Context, profileID string, tmdbID int) error {
	res := r.db.WithContext(ctx).
		Where("profile_id = ? AND tmdb_id = ? AND favorite_type = ? AND media_id IS NULL",
			profileID, tmdbID, common.FavWant).
		Delete(&history.Favorite{})
	if res.Error != nil {
		return apperr.Wrap(res.Error, apperr.CodeInternal, "取消想看失败")
	}
	return nil
}

// ListAllWants 列出全部 Profile 的想看（CMS 用）
func (r *HistoryRepo) ListAllWants(ctx context.Context, limit int) ([]history.Favorite, error) {
	if limit <= 0 {
		limit = 200
	}
	var fs []history.Favorite
	if err := r.db.WithContext(ctx).
		Preload("Media").
		Preload("Profile").
		Where("favorite_type = ?", common.FavWant).
		Order("created_at DESC").
		Limit(limit).
		Find(&fs).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询想看列表失败")
	}
	return fs, nil
}
