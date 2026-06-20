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

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/media"

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
		Preload("Seasons.Episodes").
		First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("媒资不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询媒资失败")
	}
	return &m, nil
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

// Update 更新（按主键，不触碰 seasons/episodes 关联）
func (r *MediaRepo) Update(ctx context.Context, m *media.Media) error {
	if err := r.db.WithContext(ctx).
		Session(&gorm.Session{FullSaveAssociations: false}).
		Omit("Seasons").
		Save(m).Error; err != nil {
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
