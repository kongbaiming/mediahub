package repository

import (
	"context"
	"errors"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/settings"

	"gorm.io/gorm"
)

const mediaScanConfigID int16 = 1

// MediaScanConfigRepo 媒资扫描配置仓储
type MediaScanConfigRepo struct {
	db *gorm.DB
}

func NewMediaScanConfigRepo(db *gorm.DB) *MediaScanConfigRepo {
	return &MediaScanConfigRepo{db: db}
}

func (r *MediaScanConfigRepo) Get(ctx context.Context) (*settings.MediaScanConfig, error) {
	var cfg settings.MediaScanConfig
	err := r.db.WithContext(ctx).First(&cfg, "id = ?", mediaScanConfigID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = settings.MediaScanConfig{
			ID:              mediaScanConfigID,
			Enabled:         true,
			IntervalMinutes: 30,
		}
		if createErr := r.db.WithContext(ctx).Create(&cfg).Error; createErr != nil {
			return nil, apperr.Wrap(createErr, apperr.CodeInternal, "初始化扫描配置失败")
		}
		return &cfg, nil
	}
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询扫描配置失败")
	}
	return &cfg, nil
}

func (r *MediaScanConfigRepo) Update(ctx context.Context, enabled bool, intervalMinutes int) (*settings.MediaScanConfig, error) {
	cfg, err := r.Get(ctx)
	if err != nil {
		return nil, err
	}
	cfg.Enabled = enabled
	cfg.IntervalMinutes = settings.NormalizeScanInterval(intervalMinutes)
	cfg.UpdatedAt = time.Now()
	if err := r.db.WithContext(ctx).Save(cfg).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "保存扫描配置失败")
	}
	return cfg, nil
}

func (r *MediaScanConfigRepo) UpdateScanResult(ctx context.Context, status settings.ScanStatus, message string, added, total int) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&settings.MediaScanConfig{}).
		Where("id = ?", mediaScanConfigID).
		Updates(map[string]any{
			"last_scan_at":      now,
			"last_scan_status":  status,
			"last_scan_message": message,
			"last_scan_added":   added,
			"last_scan_total":   total,
			"updated_at":        now,
		}).Error
}

func (r *MediaScanConfigRepo) IsDue(ctx context.Context, now time.Time) (bool, *settings.MediaScanConfig, error) {
	cfg, err := r.Get(ctx)
	if err != nil {
		return false, nil, err
	}
	if !cfg.Enabled {
		return false, cfg, nil
	}
	if cfg.LastScanAt == nil {
		return true, cfg, nil
	}
	next := cfg.LastScanAt.Add(time.Duration(cfg.IntervalMinutes) * time.Minute)
	return !now.Before(next), cfg, nil
}
