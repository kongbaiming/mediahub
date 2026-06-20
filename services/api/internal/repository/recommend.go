package repository

import (
	"context"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/history"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecommendRepo 推荐仓储
type RecommendRepo struct {
	db *gorm.DB
}

// NewRecommendRepo 构造
func NewRecommendRepo(db *gorm.DB) *RecommendRepo {
	return &RecommendRepo{db: db}
}

// Save 保存推荐结果
func (r *RecommendRepo) Save(ctx context.Context, profileID *uuid.UUID, algo string, mediaIDs []uuid.UUID, expiresAt time.Time) error {
	rec := history.Recommendation{
		ProfileID:  uuid.Nil,
		Algo:       algo,
		MediaIDs:   history.UUIDArray(mediaIDs),
		ComputedAt: time.Now(),
		ExpiresAt:  expiresAt,
	}
	if profileID != nil {
		rec.ProfileID = *profileID
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "保存推荐失败")
	}
	return nil
}

// GetLatest 拉取最新有效推荐
func (r *RecommendRepo) GetLatest(ctx context.Context, profileID *uuid.UUID, algo string) (*history.Recommendation, error) {
	q := r.db.WithContext(ctx).Where("algo = ?", algo).
		Where("expires_at > ?", time.Now()).
		Order("computed_at DESC")
	if profileID != nil {
		q = q.Where("profile_id = ?", *profileID)
	} else {
		q = q.Where("profile_id IS NULL OR profile_id = ?", uuid.Nil)
	}

	var rec history.Recommendation
	if err := q.First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// DeleteExpired 清理过期推荐
func (r *RecommendRepo) DeleteExpired(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&history.Recommendation{})
	return res.RowsAffected, res.Error
}
