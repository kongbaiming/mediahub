package scanner

import (
	"context"
	"path/filepath"

	"github.com/mediahub/api/pkg/logger"
)

// RemigrateMisplacedMovies 将误按电影入库的剧集单集重建为专辑结构（基于 DB，无需全库 walk）
func (s *Service) RemigrateMisplacedMovies(ctx context.Context) (int, error) {
	if s.mediaRepo == nil {
		return 0, nil
	}

	movies, err := s.mediaRepo.ListByType(ctx, "movie")
	if err != nil {
		return 0, err
	}

	deps := IngestDeps{MediaRepo: s.mediaRepo, Catalog: s.catalogRepo, Queue: s.queue}
	migrated := 0

	for _, m := range movies {
		filePath := m.StoragePath
		parsed := ParseFilePath(filePath)
		parentDir := filepath.Dir(filePath)
		mtype := inferTypeFromDir(parsed, parentDir)
		if !IsEpisodeFile(parsed, string(mtype)) {
			continue
		}

		if err := s.mediaRepo.Delete(ctx, m.ID.String()); err != nil {
			logger.Warn("迁移剧集单集失败", "id", m.ID, "path", filePath, "err", err)
			continue
		}

		if _, err := IngestMediaFile(ctx, deps, filePath); err != nil {
			logger.Warn("剧集单集重建失败", "path", filePath, "err", err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		logger.Info("误入库剧集单集迁移完成", "count", migrated)
	}
	return migrated, nil
}

// RemigrateSeriesOrphanFiles 将已入库但未建季/集结构的剧集文件重建为 episodes
func (s *Service) RemigrateSeriesOrphanFiles(ctx context.Context) (int, error) {
	if s.mediaRepo == nil {
		return 0, nil
	}

	paths, err := s.mediaRepo.ListOrphanSeriesFilePaths(ctx)
	if err != nil {
		return 0, err
	}

	deps := IngestDeps{MediaRepo: s.mediaRepo, Catalog: s.catalogRepo, Queue: s.queue}
	migrated := 0

	for _, filePath := range paths {
		parsed := ParseFilePath(filePath)
		parentDir := filepath.Dir(filePath)
		mtype := inferTypeFromDir(parsed, parentDir)
		if !IsEpisodeFile(parsed, string(mtype)) {
			continue
		}
		if _, err := IngestMediaFile(ctx, deps, filePath); err != nil {
			logger.Warn("剧集孤儿文件重建失败", "path", filePath, "err", err)
			continue
		}
		migrated++
	}

	if migrated > 0 {
		logger.Info("剧集孤儿文件迁移完成", "count", migrated)
	}
	return migrated, nil
}
