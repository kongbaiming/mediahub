package scanner

import (
	"context"
	"path/filepath"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"
)

// IngestDeps 入库依赖
type IngestDeps struct {
	MediaRepo *repository.MediaRepo
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

	m := &media.Media{
		Type:         mtype,
		Title:        parsed.Title,
		Year:         parsed.Year,
		StoragePath:  filePath,
		Container:    strPtr(parsed.Container),
		VideoCodec:   strPtr(parsed.VideoCodec),
		AudioCodec:   strPtr(parsed.AudioCodec),
		ScrapeStatus: common.ScrapeStatusPending,
		Tags:         buildTags(parsed),
	}
	if parsed.Resolution != "" {
		r := parsed.Resolution
		m.Resolution = &r
	}

	if err := deps.MediaRepo.Create(ctx, m); err != nil {
		return res, err
	}
	res.Added = true
	enqueueScrape(ctx, deps.Queue, m.ID.String(), true)
	logger.Info("媒资入库", "id", m.ID, "title", m.Title, "path", filePath)
	return res, nil
}

func ingestEpisodeFile(ctx context.Context, deps IngestDeps, filePath string, parsed *ParsedFile, mtype common.MediaType) (*IngestResult, error) {
	res := &IngestResult{}

	if _, err := deps.MediaRepo.GetEpisodeByFilePath(ctx, filePath); err == nil {
		res.Skipped = true
		return res, nil
	} else if ae, ok := apperr.As(err); !ok || ae.Code != apperr.CodeNotFound {
		return res, err
	}

	seriesDir := filepath.Dir(filePath)
	seasonNum := 1
	epNum := 1
	if parsed.Season != nil && *parsed.Season > 0 {
		seasonNum = *parsed.Season
	}
	if parsed.Episode != nil && *parsed.Episode > 0 {
		epNum = *parsed.Episode
	}

	series, err := deps.MediaRepo.GetBySeriesPath(ctx, seriesDir)
	isNewSeries := false
	if err != nil {
		if ae, ok := apperr.As(err); !ok || ae.Code != apperr.CodeNotFound {
			return res, err
		}
		series = &media.Media{
			Type:         mtype,
			Title:        parsed.Title,
			StoragePath:  seriesDir,
			ScrapeStatus: common.ScrapeStatusPending,
		}
		if err := deps.MediaRepo.Create(ctx, series); err != nil {
			return res, err
		}
		isNewSeries = true
		res.Added = true
	}

	if _, err := deps.MediaRepo.UpsertEpisode(ctx, series.ID, seasonNum, epNum, filePath, parsed.OriginalName); err != nil {
		return res, err
	}

	if !isNewSeries {
		res.Skipped = true
	} else {
		logger.Info("剧集专辑入库", "id", series.ID, "title", series.Title, "path", seriesDir)
	}

	enqueueScrape(ctx, deps.Queue, series.ID.String(), series.ScrapeStatus != common.ScrapeStatusDone)
	return res, nil
}

func enqueueScrape(ctx context.Context, q *queue.Queue, mediaID string, need bool) {
	if q == nil || !need {
		return
	}
	_ = q.EnqueueScrape(ctx, mediaID)
}
