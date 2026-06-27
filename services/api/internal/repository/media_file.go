package repository

import (
	"context"
	"errors"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/media"

	"gorm.io/gorm"
)

// FileProbeInfo ffprobe 写入文件层的字段
type FileProbeInfo struct {
	Duration    int
	VideoCodec  string
	AudioCodec  string
	Resolution  string
	HasSubtitle bool
	BitRate     int64
}

// UpsertMediaFile 按路径 upsert 文件记录（扫描入库用）
func (r *MediaRepo) UpsertMediaFile(ctx context.Context, f *media.MediaFile) (*media.MediaFile, error) {
	if f == nil || f.Path == "" {
		return nil, apperr.Validation(map[string]string{"path": "路径不能为空"})
	}

	var existing media.MediaFile
	err := r.db.WithContext(ctx).Where("path = ?", f.Path).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := r.db.WithContext(ctx).Create(f).Error; err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "创建文件记录失败")
		}
		return f, r.syncLegacyFileFields(ctx, f)
	}
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询文件记录失败")
	}

	existing.MediaID = f.MediaID
	existing.EpisodeID = f.EpisodeID
	existing.IsPrimary = f.IsPrimary
	if f.Source != "" {
		existing.Source = f.Source
	}
	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "更新文件记录失败")
	}
	return &existing, r.syncLegacyFileFields(ctx, &existing)
}

// GetFileByPath 按绝对路径查文件
func (r *MediaRepo) GetFileByPath(ctx context.Context, path string) (*media.MediaFile, error) {
	var f media.MediaFile
	err := r.db.WithContext(ctx).Where("path = ?", path).First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询文件失败")
	}
	return &f, nil
}

// ApplyProbeToFile 将 ffprobe 结果写入 media_files
func (r *MediaRepo) ApplyProbeToFile(ctx context.Context, path string, info FileProbeInfo, width, height int) error {
	f, err := r.GetFileByPath(ctx, path)
	if err != nil {
		return err
	}
	if f == nil {
		return nil
	}

	now := time.Now()
	updates := map[string]any{
		"probe_status": "done",
		"probed_at":    now,
		"duration_sec": info.Duration,
		"has_subtitle": info.HasSubtitle,
	}
	if info.VideoCodec != "" {
		updates["video_codec"] = info.VideoCodec
	}
	if info.AudioCodec != "" {
		updates["audio_codec"] = info.AudioCodec
	}
	if width > 0 {
		updates["width"] = width
	}
	if height > 0 {
		updates["height"] = height
	}
	if info.Resolution != "" {
		updates["resolution"] = info.Resolution
	}
	if info.BitRate > 0 && info.Duration > 0 {
		updates["file_size"] = info.BitRate * int64(info.Duration) / 8
	}

	if err := r.db.WithContext(ctx).Model(f).Updates(updates).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "更新 probe 失败")
	}

	var refreshed media.MediaFile
	if err := r.db.WithContext(ctx).First(&refreshed, "id = ?", f.ID).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "读取文件失败")
	}
	return r.syncLegacyFileFields(ctx, &refreshed)
}

// ListOrphanSeriesFilePaths 剧集作品下未关联 episode 的文件路径（需重建季/集结构）
func (r *MediaRepo) ListOrphanSeriesFilePaths(ctx context.Context) ([]string, error) {
	var paths []string
	err := r.db.WithContext(ctx).
		Table("media_files AS mf").
		Joins("JOIN media AS m ON m.id = mf.media_id").
		Where("mf.episode_id IS NULL AND m.kind = ?", media.MediaKindSeries).
		Pluck("mf.path", &paths).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询剧集孤儿文件失败")
	}
	return paths, nil
}

// syncLegacyFileFields 双写兼容列（v0.4 过渡）
func (r *MediaRepo) syncLegacyFileFields(ctx context.Context, f *media.MediaFile) error {
	if f.EpisodeID != nil {
		return r.db.WithContext(ctx).Model(&media.Episode{}).
			Where("id = ?", *f.EpisodeID).
			Updates(map[string]any{
				"file_path": f.Path,
				"file_size": f.FileSize,
			}).Error
	}

	updates := map[string]any{
		"storage_path":  f.Path,
		"file_size":     f.FileSize,
		"video_codec":   f.VideoCodec,
		"audio_codec":   f.AudioCodec,
		"resolution":    f.Resolution,
		"container":     f.Container,
		"has_subtitle":  f.HasSubtitle,
	}
	return r.db.WithContext(ctx).Model(&media.Media{}).
		Where("id = ?", f.MediaID).
		Updates(updates).Error
}
