package repository

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/mediafile"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindMovieInFolder 查找文件夹内已入库的电影（storage_path 或 media_files 路径匹配）
func (r *MediaRepo) FindMovieInFolder(ctx context.Context, folder string) (*media.Media, error) {
	folder = filepath.Clean(folder)
	if folder == "" || folder == "." {
		return nil, nil
	}

	var m media.Media
	err := r.db.WithContext(ctx).
		Where("type = ? AND storage_path = ?", common.MediaTypeMovie, folder).
		First(&m).Error
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询文件夹电影失败")
	}

	prefix := folder + string(filepath.Separator)
	err = r.db.WithContext(ctx).
		Where("type = ? AND storage_path LIKE ?", common.MediaTypeMovie, prefix+"%").
		Order("created_at ASC").
		First(&m).Error
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询文件夹电影失败")
	}

	var f media.MediaFile
	err = r.db.WithContext(ctx).
		Where("path LIKE ?", prefix+"%").
		Order("created_at ASC").
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询文件记录失败")
	}
	return r.GetByID(ctx, f.MediaID.String())
}

// DedupeMoviesInFolder 合并同一文件夹内重复入库的电影记录
func (r *MediaRepo) DedupeMoviesInFolder(ctx context.Context) (int, error) {
	var movies []media.Media
	if err := r.db.WithContext(ctx).
		Where("type = ?", common.MediaTypeMovie).
		Find(&movies).Error; err != nil {
		return 0, apperr.Wrap(err, apperr.CodeInternal, "查询电影失败")
	}

	type groupKey struct {
		folder string
	}
	groups := map[groupKey][]media.Media{}
	for _, m := range movies {
		folder := movieFolderKey(m.StoragePath)
		if folder == "" {
			continue
		}
		k := groupKey{folder: folder}
		groups[k] = append(groups[k], m)
	}

	merged := 0
	for _, list := range groups {
		if len(list) < 2 {
			continue
		}
		keeper := pickMovieKeeper(list)
		for _, dup := range list {
			if dup.ID == keeper.ID {
				continue
			}
			if err := r.mergeMovieInto(ctx, keeper.ID, dup); err != nil {
				return merged, err
			}
			merged++
		}
	}
	return merged, nil
}

func movieFolderKey(storagePath string) string {
	storagePath = filepath.Clean(storagePath)
	if storagePath == "" || storagePath == "." {
		return ""
	}
	if isVideoPath(storagePath) {
		return filepath.Dir(storagePath)
	}
	return storagePath
}

func isVideoPath(p string) bool {
	ok, _ := mediafile.IsPlayable(p)
	return ok
}

func pickMovieKeeper(list []media.Media) media.Media {
	best := list[0]
	bestScore := movieKeeperScore(best)
	for _, m := range list[1:] {
		if s := movieKeeperScore(m); s > bestScore {
			best = m
			bestScore = s
		}
	}
	return best
}

func movieKeeperScore(m media.Media) int {
	score := 0
	if m.TMDBID != nil && *m.TMDBID > 0 {
		score += 100
	}
	if m.ScrapeStatus == common.ScrapeStatusDone {
		score += 50
	}
	if strings.TrimSpace(m.Overview) != "" {
		score += 20
	}
	if strings.TrimSpace(m.PosterURL) != "" {
		score += 10
	}
	if m.Rating > 0 {
		score += 5
	}
	return score
}

func (r *MediaRepo) mergeMovieInto(ctx context.Context, keeperID uuid.UUID, dup media.Media) error {
	dupID := dup.ID
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&media.MediaFile{}).
			Where("media_id = ?", dupID).
			Update("media_id", keeperID).Error; err != nil {
			return err
		}
		if isVideoPath(dup.StoragePath) {
			var existing media.MediaFile
			err := tx.Where("path = ?", dup.StoragePath).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				f := &media.MediaFile{
					MediaID:     keeperID,
					Path:        dup.StoragePath,
					IsPrimary:   false,
					ProbeStatus: "pending",
					Source:      "dedupe",
				}
				if err := tx.Create(f).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else if existing.MediaID != keeperID {
				if err := tx.Model(&existing).Update("media_id", keeperID).Error; err != nil {
					return err
				}
			}
		}
		return tx.Where("id = ?", dupID).Delete(&media.Media{}).Error
	})
}
