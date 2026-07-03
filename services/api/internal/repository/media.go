// Package repository 提供数据访问层（仓储模式）
//
// 设计原则：
//   - 每个领域一个仓储
//   - 仓储只依赖 *gorm.DB，不依赖 service
//   - 仓储返回领域实体（domain 包），不返回 GORM 模型
package repository

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/media"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaRepo 媒资仓储
type MediaRepo struct {
	db *gorm.DB
}

// NewMediaRepo 构造媒资仓储
func NewMediaRepo(db *gorm.DB) *MediaRepo {
	return &MediaRepo{db: db}
}

// Filter 媒资查询过滤条件
type MediaFilter struct {
	Type          string
	Genre         string
	Year          *int
	MinRating     *float64
	Search        string  // 模糊搜索 title/original_title
	Sort          string  // year | rating | created_at | title
	SortDesc      bool
	ExcludeIDs    []string // 排除的媒资 ID
	HasSubtitle   *bool
	ScrapeStatus  string
	ExcludeAdult  bool     // 排除成人内容（儿童模式用）
}

// List 列表查询（带分页 + 过滤）
func (r *MediaRepo) List(ctx context.Context, f MediaFilter, limit, offset int) ([]media.Media, int64, error) {
	q := r.db.WithContext(ctx).Model(&media.Media{})

	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Genre != "" {
		q = q.Where("? = ANY(genres)", f.Genre)
	}
	if f.Year != nil {
		q = q.Where("year = ?", *f.Year)
	}
	if f.MinRating != nil {
		q = q.Where("rating >= ?", *f.MinRating)
	}
	if f.Search != "" {
		q = q.Where("title ILIKE ? OR original_title ILIKE ?",
			"%"+f.Search+"%", "%"+f.Search+"%")
	}
	if f.HasSubtitle != nil {
		q = q.Where("has_subtitle = ?", *f.HasSubtitle)
	}
	if f.ScrapeStatus != "" {
		q = q.Where("scrape_status = ?", f.ScrapeStatus)
	}
	if f.ExcludeAdult {
		q = q.Where("is_adult = ?", false)
	}
	if len(f.ExcludeIDs) > 0 {
		q = q.Where("id NOT IN ?", f.ExcludeIDs)
	}

	// 排序
	switch f.Sort {
	case "year":
		if f.SortDesc {
			q = q.Order("year DESC")
		} else {
			q = q.Order("year ASC")
		}
	case "rating":
		if f.SortDesc {
			q = q.Order("rating DESC")
		} else {
			q = q.Order("rating ASC")
		}
	case "title":
		q = q.Order("title ASC")
	case "created_at", "":
		if f.SortDesc {
			q = q.Order("created_at DESC")
		} else {
			q = q.Order("created_at ASC")
		}
	default:
		q = q.Order("created_at DESC")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(err, apperr.CodeInternal, "媒资计数失败")
	}

	var items []media.Media
	if err := q.Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, apperr.Wrap(err, apperr.CodeInternal, "媒资列表查询失败")
	}

	return items, total, nil
}

// GetByID 按 ID 获取
func (r *MediaRepo) GetByID(ctx context.Context, id string) (*media.Media, error) {
	var m media.Media
	if err := r.db.WithContext(ctx).
		Preload("Seasons", func(db *gorm.DB) *gorm.DB {
			return db.Order("season_number ASC")
		}).
		Preload("Seasons.Episodes", func(db *gorm.DB) *gorm.DB {
			return db.Order("episode_number ASC")
		}).
		Preload("Seasons.Episodes.Files").
		Preload("Files").
		First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("媒资不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询媒资失败")
	}
	return &m, nil
}

// ListByType 按类型列出全部媒资（启动迁移等批处理用）
func (r *MediaRepo) ListByType(ctx context.Context, mediaType string) ([]media.Media, error) {
	var items []media.Media
	if err := r.db.WithContext(ctx).Where("type = ?", mediaType).Find(&items).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询媒资失败")
	}
	return items, nil
}

// GetByStoragePath 按文件路径获取（用于去重）
func (r *MediaRepo) GetByStoragePath(ctx context.Context, path string) (*media.Media, error) {
	var m media.Media
	err := r.db.WithContext(ctx).Where("storage_path = ?", path).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询媒资失败")
	}
	return &m, nil
}

// GetByTMDBID 按 TMDB ID 获取
func (r *MediaRepo) GetByTMDBID(ctx context.Context, tmdbID int) (*media.Media, error) {
	var m media.Media
	err := r.db.WithContext(ctx).Where("tmdb_id = ?", tmdbID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询媒资失败")
	}
	return &m, nil
}

// Create 创建
func (r *MediaRepo) Create(ctx context.Context, m *media.Media) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建媒资失败")
	}
	return nil
}

// GetBySeriesPath 按剧集专辑目录查找（storage_path 为文件夹；兼容误设为单集文件路径的情况）
func (r *MediaRepo) GetBySeriesPath(ctx context.Context, seriesDir string) (*media.Media, error) {
	seriesDir = filepath.Clean(seriesDir)
	if seriesDir == "" || seriesDir == "." {
		return nil, apperr.NotFound("剧集专辑不存在")
	}
	seriesTypes := []string{"tvshow", "anime"}

	var m media.Media
	err := r.db.WithContext(ctx).
		Where("storage_path = ? AND type IN ?", seriesDir, seriesTypes).
		First(&m).Error
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询剧集专辑失败")
	}

	prefix := seriesDir + string(filepath.Separator)
	err = r.db.WithContext(ctx).
		Where("kind = ? AND type IN ? AND storage_path LIKE ?", media.MediaKindSeries, seriesTypes, prefix+"%").
		Order("updated_at DESC").
		First(&m).Error
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询剧集专辑失败")
	}

	var f media.MediaFile
	err = r.db.WithContext(ctx).
		Table("media_files AS mf").
		Joins("JOIN media AS m ON m.id = mf.media_id").
		Where("m.kind = ? AND mf.path LIKE ?", media.MediaKindSeries, prefix+"%").
		Select("mf.*").
		Order("mf.created_at ASC").
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("剧集专辑不存在")
	}
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询剧集专辑失败")
	}
	return r.GetByID(ctx, f.MediaID.String())
}

// SetStoragePath 更新媒资 storage_path（剧集专辑目录修正等）
func (r *MediaRepo) SetStoragePath(ctx context.Context, mediaID string, path string) error {
	if err := r.db.WithContext(ctx).Model(&media.Media{}).
		Where("id = ?", mediaID).
		Update("storage_path", path).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "更新 storage_path 失败")
	}
	return nil
}

// GetEpisodeByFilePath 按单集文件路径查找
func (r *MediaRepo) GetEpisodeByFilePath(ctx context.Context, filePath string) (*media.Episode, error) {
	var ep media.Episode
	err := r.db.WithContext(ctx).Where("file_path = ?", filePath).First(&ep).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("单集不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询单集失败")
	}
	return &ep, nil
}

// GetFirstEpisodeFilePath 取专辑下第一个单集文件（ffprobe 用）
func (r *MediaRepo) GetFirstEpisodeFilePath(ctx context.Context, mediaID string) (string, error) {
	var ep media.Episode
	err := r.db.WithContext(ctx).
		Where("media_id = ? AND file_path <> ''", mediaID).
		Order("season_id, episode_number").
		First(&ep).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperr.NotFound("无单集文件")
		}
		return "", apperr.Wrap(err, apperr.CodeInternal, "查询单集失败")
	}
	return ep.FilePath, nil
}

// UpsertEpisode 创建或更新季/集记录
func (r *MediaRepo) UpsertEpisode(ctx context.Context, mediaID uuid.UUID, seasonNum, epNum int, filePath, epTitle string) (*media.Episode, error) {
	var season media.Season
	err := r.db.WithContext(ctx).
		Where("media_id = ? AND season_number = ?", mediaID, seasonNum).
		First(&season).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		season = media.Season{
			MediaID:      mediaID,
			SeasonNumber: seasonNum,
		}
		if err := r.db.WithContext(ctx).Create(&season).Error; err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "创建季失败")
		}
	} else if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询季失败")
	}

	var ep media.Episode
	err = r.db.WithContext(ctx).
		Where("season_id = ? AND episode_number = ?", season.ID, epNum).
		First(&ep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ep = media.Episode{
			SeasonID:      season.ID,
			MediaID:       mediaID,
			EpisodeNumber: epNum,
			Title:         epTitle,
			FilePath:      filePath,
		}
		if err := r.db.WithContext(ctx).Create(&ep).Error; err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "创建集失败")
		}
	} else if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询集失败")
	} else {
		ep.Title = epTitle
		ep.FilePath = filePath
		if err := r.db.WithContext(ctx).Save(&ep).Error; err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "更新集失败")
		}
	}

	// 更新季集数
	var count int64
	r.db.WithContext(ctx).Model(&media.Episode{}).Where("season_id = ?", season.ID).Count(&count)
	r.db.WithContext(ctx).Model(&season).Update("episode_count", count)

	return &ep, nil
}

// NextEpisodeNumber 取季内下一个可用集号（小品集等无集号命名时使用）
func (r *MediaRepo) NextEpisodeNumber(ctx context.Context, mediaID uuid.UUID, seasonNum int) (int, error) {
	var season media.Season
	err := r.db.WithContext(ctx).
		Where("media_id = ? AND season_number = ?", mediaID, seasonNum).
		First(&season).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, apperr.Wrap(err, apperr.CodeInternal, "查询季失败")
	}
	var maxEp int
	if err := r.db.WithContext(ctx).Model(&media.Episode{}).
		Where("season_id = ?", season.ID).
		Select("COALESCE(MAX(episode_number), 0)").
		Scan(&maxEp).Error; err != nil {
		return 0, apperr.Wrap(err, apperr.CodeInternal, "查询集数失败")
	}
	return maxEp + 1, nil
}

// NextEpisode 取下一集
func (r *MediaRepo) NextEpisode(ctx context.Context, mediaID string, afterEpisodeID string) (*media.Episode, error) {
	var current media.Episode
	if err := r.db.WithContext(ctx).First(&current, "id = ? AND media_id = ?", afterEpisodeID, mediaID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("当前集不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询集失败")
	}
	var season media.Season
	if err := r.db.WithContext(ctx).First(&season, "id = ?", current.SeasonID).Error; err != nil {
		return nil, err
	}
	var next media.Episode
	err := r.db.WithContext(ctx).
		Where("season_id = ? AND episode_number > ? AND file_path <> ''", season.ID, current.EpisodeNumber).
		Order("episode_number ASC").First(&next).Error
	if err == nil {
		return &next, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var nextSeason media.Season
	err = r.db.WithContext(ctx).
		Where("media_id = ? AND season_number > ?", mediaID, season.SeasonNumber).
		Order("season_number ASC").First(&nextSeason).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	err = r.db.WithContext(ctx).Where("season_id = ? AND file_path <> ''", nextSeason.ID).
		Order("episode_number ASC").First(&next).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &next, nil
}

// UpdateScrapeStatus 仅更新刮削状态（不触碰 poster/overview 等元数据）
func (r *MediaRepo) UpdateScrapeStatus(ctx context.Context, id string, status string, scrapeError string) error {
	if err := r.db.WithContext(ctx).Model(&media.Media{}).Where("id = ?", id).Updates(map[string]any{
		"scrape_status": status,
		"scrape_error":  scrapeError,
	}).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "更新刮削状态失败")
	}
	return nil
}

// ApplyScrapeResult 写入 TMDB 刮削结果（局部更新，避免并发 Save 覆盖）
func (r *MediaRepo) ApplyScrapeResult(ctx context.Context, m *media.Media) error {
	genres := m.Genres
	if genres == nil {
		genres = media.StringArray{}
	}
	tags := m.Tags
	if tags == nil {
		tags = media.StringArray{}
	}

	// 重新读 tags，避免刮削任务启动后用户已在 CMS 改标题
	var current media.Media
	if err := r.db.WithContext(ctx).
		Select("tags").
		Where("id = ?", m.ID).
		First(&current).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "读取媒资标签失败")
	}
	manualTitle := media.HasTag(current.Tags, media.TagManualTitle)

	updates := map[string]any{
		"type":           m.Type,
		"kind":           m.Kind,
		"scrape_status":  m.ScrapeStatus,
		"scrape_error":   m.ScrapeError,
		"last_scrape_at": m.LastScrapeAt,
		"year":           m.Year,
		"runtime":        m.Runtime,
		"overview":       m.Overview,
		"rating":         m.Rating,
		"vote_count":     m.VoteCount,
		"tmdb_id":        m.TMDBID,
		"genres":         genres,
		"tags":           tags,
		"file_size":      m.FileSize,
		"video_codec":    m.VideoCodec,
		"audio_codec":    m.AudioCodec,
		"resolution":     m.Resolution,
		"has_subtitle":   m.HasSubtitle,
		"is_adult":       m.IsAdult,
		// 同系列
		"collection_id":         m.CollectionID,
		"collection_name":       m.CollectionName,
		"collection_poster_url": m.CollectionPosterURL,
	}
	if m.PosterURL != "" {
		updates["poster_url"] = m.PosterURL
	}
	if m.BackdropURL != "" {
		updates["backdrop_url"] = m.BackdropURL
	}
	if !manualTitle {
		updates["title"] = m.Title
		updates["original_title"] = m.OriginalTitle
	}
	if err := r.db.WithContext(ctx).Model(m).Where("id = ?", m.ID).Updates(updates).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "写入刮削结果失败")
	}
	return nil
}

// ResetStuckScraping 将卡在 scraping 的媒资重置为 pending（API 重启时恢复）
func (r *MediaRepo) ResetStuckScraping(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Model(&media.Media{}).
		Where("scrape_status = ?", "scraping").
		Updates(map[string]any{
			"scrape_status": "pending",
			"scrape_error":  "",
		})
	if res.Error != nil {
		return 0, apperr.Wrap(res.Error, apperr.CodeInternal, "重置刮削状态失败")
	}
	return res.RowsAffected, nil
}

// ListByCollectionID 按 collection_id 查询系列中的其他媒资
func (r *MediaRepo) ListByCollectionID(ctx context.Context, collectionID int, excludeMediaID string) ([]media.Media, error) {
	var items []media.Media
	q := r.db.WithContext(ctx).Where("collection_id = ?", collectionID)
	if excludeMediaID != "" {
		q = q.Where("id != ?", excludeMediaID)
	}
	if err := q.Order("year ASC").Find(&items).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询同系列失败")
	}
	return items, nil
}

// Update 更新（按主键，不触碰 seasons/episodes 关联）
func (r *MediaRepo) Update(ctx context.Context, m *media.Media) error {
	genres := m.Genres
	if genres == nil {
		genres = media.StringArray{}
	}
	tags := m.Tags
	if tags == nil {
		tags = media.StringArray{}
	}
	if err := r.db.WithContext(ctx).Model(&media.Media{}).Where("id = ?", m.ID).Updates(map[string]any{
		"title":          m.Title,
		"original_title": m.OriginalTitle,
		"year":           m.Year,
		"overview":       m.Overview,
		"storage_path":   m.StoragePath,
		"poster_url":     m.PosterURL,
		"backdrop_url":   m.BackdropURL,
		"rating":         m.Rating,
		"genres":         genres,
		"tags":           tags,
	}).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "更新媒资失败")
	}
	return nil
}

// Delete 删除（软删）
func (r *MediaRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&media.Media{}, "id = ?", id).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "删除媒资失败")
	}
	return nil
}

// CountByType 按类型统计
func (r *MediaRepo) CountByType(ctx context.Context) (map[string]int64, error) {
	type row struct {
		Type  string
		Count int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&media.Media{}).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&rows).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "统计媒资失败")
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.Type] = r.Count
	}
	return out, nil
}

// Stats 媒资统计
type Stats struct {
	Total       int64            `json:"total"`
	ByType      map[string]int64 `json:"by_type"`
	ByScrape    map[string]int64 `json:"by_scrape"`
}

// Stats 综合统计
func (r *MediaRepo) Stats(ctx context.Context) (*Stats, error) {
	s := &Stats{}

	// 总数
	if err := r.db.WithContext(ctx).Model(&media.Media{}).Count(&s.Total).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "统计失败")
	}

	// 按类型
	byType, err := r.CountByType(ctx)
	if err != nil {
		return nil, err
	}
	s.ByType = byType

	// 按刮削状态
	type row struct {
		Status string
		Count  int64
	}
	var scrapeRows []row
	if err := r.db.WithContext(ctx).
		Model(&media.Media{}).
		Select("scrape_status as status, COUNT(*) as count").
		Group("scrape_status").
		Scan(&scrapeRows).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "统计失败")
	}
	s.ByScrape = make(map[string]int64, len(scrapeRows))
	for _, r := range scrapeRows {
		s.ByScrape[r.Status] = r.Count
	}

	return s, nil
}
