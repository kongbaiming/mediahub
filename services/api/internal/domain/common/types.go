// Package common 提供跨领域共享的类型
package common

import (
	"time"

	"github.com/google/uuid"
)

// MediaType 媒资类型
type MediaType string

const (
	MediaTypeMovie        MediaType = "movie"
	MediaTypeTVShow       MediaType = "tvshow"
	MediaTypeAnime        MediaType = "anime"
	MediaTypeDocumentary  MediaType = "documentary"
)

// Platform 目标平台
type Platform string

const (
	PlatformWeb       Platform = "web"
	PlatformAndroidTV Platform = "android-tv"
	PlatformTVOS      Platform = "tvos"
)

// ScrapeStatus 刮削状态
type ScrapeStatus string

const (
	ScrapeStatusPending   ScrapeStatus = "pending"
	ScrapeStatusScraping  ScrapeStatus = "scraping"
	ScrapeStatusDone      ScrapeStatus = "done"
	ScrapeStatusFailed    ScrapeStatus = "failed"
)

// AvailabilityStatus 可播状态（OTT 运营）
type AvailabilityStatus string

const (
	AvailAvailable   AvailabilityStatus = "available"
	AvailProcessing  AvailabilityStatus = "processing"
	AvailMissing     AvailabilityStatus = "missing"
	AvailUnreleased  AvailabilityStatus = "unreleased"
)

// FavoriteType 收藏类型
type FavoriteType string

const (
	FavWant     FavoriteType = "want"     // 想看
	FavWatching FavoriteType = "watching" // 在看
	FavWatched  FavoriteType = "watched"  // 看过
	FavLiked    FavoriteType = "liked"    // 喜欢
)

// LayoutStatus 布局状态
type LayoutStatus string

const (
	LayoutDraft     LayoutStatus = "draft"
	LayoutPublished LayoutStatus = "published"
	LayoutArchived  LayoutStatus = "archived"
)

// BaseModel 是所有表的公共字段
type BaseModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// Pagination 通用分页参数
type Pagination struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// Normalize 标准化（默认值 + 上限保护）
func (p *Pagination) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 200 {
		p.PageSize = 200
	}
}

// Offset 计算 offset
func (p *Pagination) Offset() int { return (p.Page - 1) * p.PageSize }

// PageResult 通用分页结果
type PageResult[T any] struct {
	Data     []T `json:"data"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// NewPageResult 构造分页结果
func NewPageResult[T any](data []T, total int, p Pagination) *PageResult[T] {
	return &PageResult[T]{
		Data:     data,
		Total:    total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}
}
