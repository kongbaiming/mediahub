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

	deps := IngestDeps{MediaRepo: s.mediaRepo, Queue: s.queue}
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
