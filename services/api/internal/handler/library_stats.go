package handler

import (
	"context"

	"github.com/mediahub/api/internal/ailayout"

	"gorm.io/gorm"
)

// DBLibraryStatsProvider 从数据库获取媒资库统计
type DBLibraryStatsProvider struct {
	db *gorm.DB
}

// NewDBLibraryStatsProvider 构造
func NewDBLibraryStatsProvider(db *gorm.DB) *DBLibraryStatsProvider {
	return &DBLibraryStatsProvider{db: db}
}

// GetLibraryStats 获取媒资库统计
func (p *DBLibraryStatsProvider) GetLibraryStats(ctx context.Context) (*ailayout.LibraryStats, error) {
	stats := &ailayout.LibraryStats{}

	// 总数
	p.db.WithContext(ctx).Table("media").Count(&stats.TotalMedia)

	// 按类型统计
	type typeCount struct {
		Type  string
		Count int64
	}
	var typeCounts []typeCount
	p.db.WithContext(ctx).Table("media").
		Select("type, count(*) as count").
		Group("type").Scan(&typeCounts)

	for _, tc := range typeCounts {
		switch tc.Type {
		case "movie":
			stats.MovieCount = tc.Count
		case "tvshow":
			stats.TVShowCount = tc.Count
		case "anime":
			stats.AnimeCount = tc.Count
		case "documentary":
			stats.DocumentaryCount = tc.Count
		}
	}

	// 标签
	p.db.WithContext(ctx).Table("tags").
		Select("name").Order("name ASC").Limit(30).
		Pluck("name", &stats.Tags)

	// 分类
	p.db.WithContext(ctx).Table("categories").
		Select("name").Order("name ASC").Limit(30).
		Pluck("name", &stats.Categories)

	// 专辑
	p.db.WithContext(ctx).Table("albums").
		Select("title").Order("title ASC").Limit(20).
		Pluck("title", &stats.Albums)

	return stats, nil
}
