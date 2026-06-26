package scanner

import (
	"context"
	"path/filepath"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/mediafile"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"

	"github.com/google/uuid"
)

// IngestDeps 入库依赖
type IngestDeps struct {
	MediaRepo *repository.MediaRepo
	Catalog   *repository.CatalogRepo
	Queue     *queue.Queue
}

// IngestResult 单文件入库结果
type IngestResult struct {
	Added   bool
	Skipped bool
}

// IngestMediaFile 扫描到的媒体文件入库（电影单条；电视剧按专辑聚合）
func IngestMediaFile(ctx context.Context, deps IngestDeps, filePath string) (*IngestResult, error) {
	if deps.MediaRepo == nil {
		return nil, nil
	}
	if mediafile.ShouldSkipScan(filePath) {
		return &IngestResult{Skipped: true}, nil
	}
	if ok, reason := mediafile.IsPlayable(filePath); !ok {
		logger.Warn("跳过无效媒体文件", "path", filePath, "reason", reason)
		return &IngestResult{Skipped: true}, nil
	}

	parsed := ParseFilePath(filePath)
	parentDir := filepath.Dir(filePath)
	mtype := inferTypeFromDir(parsed, parentDir)

	if IsEpisodeFile(parsed, string(mtype)) {
		return ingestEpisodeFile(ctx, deps, filePath, parsed, mtype)
	}
	return ingestMovieFile(ctx, deps, filePath, parsed, mtype)
}

func ingestMovieFile(ctx context.Context, deps IngestDeps, filePath string, parsed *ParsedFile, mtype common.MediaType) (*IngestResult, error) {
	res := &IngestResult{}
	existing, _ := deps.MediaRepo.GetByStoragePath(ctx, filePath)
	if existing != nil {
		res.Skipped = true
		return res, nil
	}

	albumDir := movieAlbumDir(filePath)
	if albumDir != "" {
		if album, _ := deps.MediaRepo.FindMovieInFolder(ctx, albumDir); album != nil {
			_, _ = deps.MediaRepo.UpsertMediaFile(ctx, scanMediaFile(album.ID, nil, filePath))
			res.Skipped = true
			return res, nil
		}
	}

	storagePath := filePath
	if albumDir != "" {
		storagePath = albumDir
	}

	m := &media.Media{
		Type:         mtype,
		Kind:         mediaKind(mtype),
		Title:        parsed.Title,
		Year:         parsed.Year,
		StoragePath:  storagePath,
		Container:    strPtr(parsed.Container),
		VideoCodec:   strPtr(parsed.VideoCodec),
		AudioCodec:   strPtr(parsed.AudioCodec),
		ScrapeStatus: common.ScrapeStatusPending,
		Genres:       media.StringArray{},
		Tags:         media.StringArray(buildTags(parsed)),
	}
	if parsed.Resolution != "" {
		r := parsed.Resolution
		m.Resolution = &r
	}

	if err := deps.MediaRepo.Create(ctx, m); err != nil {
		return res, err
	}
	_, _ = deps.MediaRepo.UpsertMediaFile(ctx, scanMediaFile(m.ID, nil, filePath))
	res.Added = true
	enqueueScrape(ctx, deps.Queue, m.ID.String(), true)
	if deps.Catalog != nil {
		_ = deps.Catalog.RefreshAvailability(ctx, m.ID)
	}
	logger.Info("媒资入库", "id", m.ID, "title", m.Title, "path", filePath)
	return res, nil
}

func movieAlbumDir(filePath string) string {
	parent := filepath.Dir(filePath)
	if isMovieCategoryFolder(filepath.Base(parent)) {
		return ""
	}
	return parent
}

func ingestEpisodeFile(ctx context.Context, deps IngestDeps, filePath string, parsed *ParsedFile, mtype common.MediaType) (*IngestResult, error) {
	res := &IngestResult{}

	// 旧版扫描把单集误入库为 movie（storage_path=文件路径），需删除后重建专辑结构
	migrated, err := removeMisplacedMovieRecord(ctx, deps.MediaRepo, filePath)
	if err != nil {
		return res, err
	}

	if _, err := deps.MediaRepo.GetEpisodeByFilePath(ctx, filePath); err == nil {
		res.Skipped = true
		return res, nil
	} else if ae, ok := apperr.As(err); !ok || ae.Code != apperr.CodeNotFound {
		return res, err
	}

	seriesDir := filepath.Dir(filePath)
	seasonNum := 1
	if parsed.Season != nil && *parsed.Season > 0 {
		seasonNum = *parsed.Season
	}

	series, err := deps.MediaRepo.GetBySeriesPath(ctx, seriesDir)
	isNewSeries := false
	if err != nil {
		if ae, ok := apperr.As(err); !ok || ae.Code != apperr.CodeNotFound {
			return res, err
		}
		series = &media.Media{
			Type:         mtype,
			Kind:         mediaKind(mtype),
			Title:        parsed.Title,
			StoragePath:  seriesDir,
			ScrapeStatus: common.ScrapeStatusPending,
		}
		if err := deps.MediaRepo.Create(ctx, series); err != nil {
			return res, err
		}
		isNewSeries = true
	}

	epNum := 1
	epTitle := parsed.OriginalName
	if parsed.Episode != nil && *parsed.Episode > 0 {
		epNum = *parsed.Episode
	} else if isCollectionAlbumDir(filepath.Base(seriesDir)) {
		epNum, err = deps.MediaRepo.NextEpisodeNumber(ctx, series.ID, seasonNum)
		if err != nil {
			return res, err
		}
		epTitle = collectionEpisodeTitle(filePath)
	}

	if _, err := deps.MediaRepo.UpsertEpisode(ctx, series.ID, seasonNum, epNum, filePath, epTitle); err != nil {
		return res, err
	}
	ep, err := deps.MediaRepo.GetEpisodeByFilePath(ctx, filePath)
	if err == nil && ep != nil {
		_, _ = deps.MediaRepo.UpsertMediaFile(ctx, scanMediaFile(series.ID, &ep.ID, filePath))
	}

	if isNewSeries || migrated {
		res.Added = true
		if isNewSeries {
			logger.Info("剧集专辑入库", "id", series.ID, "title", series.Title, "path", seriesDir)
		}
		if deps.Catalog != nil {
			_ = deps.Catalog.RefreshAvailability(ctx, series.ID)
		}
	} else {
		res.Skipped = true
	}

	enqueueScrape(ctx, deps.Queue, series.ID.String(), series.ScrapeStatus != common.ScrapeStatusDone)
	return res, nil
}

func mediaKind(mtype common.MediaType) media.MediaKind {
	if mtype == common.MediaTypeTVShow || mtype == common.MediaTypeAnime {
		return media.MediaKindSeries
	}
	return media.MediaKindSingle
}

func scanMediaFile(mediaID uuid.UUID, episodeID *uuid.UUID, path string) *media.MediaFile {
	return &media.MediaFile{
		MediaID:     mediaID,
		EpisodeID:   episodeID,
		Path:        path,
		IsPrimary:   true,
		ProbeStatus: "pending",
		Source:      "scan",
	}
}

func enqueueScrape(ctx context.Context, q *queue.Queue, mediaID string, need bool) {
	if q == nil || !need {
		return
	}
	_ = q.EnqueueScrape(ctx, mediaID)
}

// removeMisplacedMovieRecord 删除误将剧集单集按电影入库的记录，便于重新聚合为专辑
func removeMisplacedMovieRecord(ctx context.Context, repo *repository.MediaRepo, filePath string) (bool, error) {
	if repo == nil {
		return false, nil
	}
	existing, err := repo.GetByStoragePath(ctx, filePath)
	if err != nil {
		return false, err
	}
	if existing == nil || existing.Type != common.MediaTypeMovie {
		return false, nil
	}
	if err := repo.Delete(ctx, existing.ID.String()); err != nil {
		return false, err
	}
	logger.Info("迁移错误入库的单集电影记录", "id", existing.ID, "title", existing.Title, "path", filePath)
	return true, nil
}
