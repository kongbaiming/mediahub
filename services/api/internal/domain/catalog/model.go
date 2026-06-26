// Package catalog 内容目录域（影人、分类、标签、专辑）
package catalog

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"

	"github.com/google/uuid"
)

// Person 影人
type Person struct {
	common.BaseModel
	Name               string  `gorm:"type:varchar(300);not null" json:"name"`
	OriginalName       string  `gorm:"type:varchar(300)" json:"original_name,omitempty"`
	TMDBPersonID       *int    `gorm:"column:tmdb_person_id;uniqueIndex" json:"tmdb_person_id,omitempty"`
	ProfilePath        string  `gorm:"type:text" json:"profile_path,omitempty"`
	Biography          string  `gorm:"type:text" json:"biography,omitempty"`
	Birthday           *time.Time `gorm:"type:date" json:"birthday,omitempty"`
	PlaceOfBirth       string  `gorm:"type:varchar(200)" json:"place_of_birth,omitempty"`
	Gender             int     `gorm:"default:0" json:"gender"`
	KnownForDepartment string  `gorm:"type:varchar(50)" json:"known_for_department,omitempty"`
	Popularity         float64 `gorm:"type:decimal(8,3);default:0" json:"popularity"`
}

func (Person) TableName() string { return "persons" }

// MediaCredit 演职员
type MediaCredit struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MediaID       uuid.UUID `gorm:"type:uuid;not null;index" json:"media_id"`
	PersonID      uuid.UUID `gorm:"type:uuid;not null;index" json:"person_id"`
	Role          string    `gorm:"type:varchar(32);not null" json:"role"`
	CharacterName string    `gorm:"type:varchar(300)" json:"character_name,omitempty"`
	BillingOrder  int       `gorm:"default:0" json:"billing_order"`
	CreatedAt     time.Time `json:"created_at"`
	Person        *Person   `gorm:"foreignKey:PersonID" json:"person,omitempty"`
}

func (MediaCredit) TableName() string { return "media_credits" }

// Category 分类
type Category struct {
	common.BaseModel
	ParentID    *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	Slug        string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"slug"`
	Kind        string     `gorm:"type:varchar(20);not null;default:'genre'" json:"kind"`
	TMDBGenreID *int       `gorm:"column:tmdb_genre_id" json:"tmdb_genre_id,omitempty"`
	SortOrder   int        `gorm:"default:0" json:"sort_order"`
}

func (Category) TableName() string { return "categories" }

// MediaCategory 作品-分类
type MediaCategory struct {
	MediaID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"media_id"`
	CategoryID uuid.UUID `gorm:"type:uuid;primaryKey" json:"category_id"`
	IsPrimary  bool      `gorm:"default:false" json:"is_primary"`
}

func (MediaCategory) TableName() string { return "media_categories" }

// Tag 标签
type Tag struct {
	common.BaseModel
	Name string `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Slug string `gorm:"type:varchar(100);not null;uniqueIndex" json:"slug"`
}

func (Tag) TableName() string { return "tags" }

// MediaTag 作品-标签
type MediaTag struct {
	MediaID uuid.UUID `gorm:"type:uuid;primaryKey" json:"media_id"`
	TagID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
	Source  string    `gorm:"type:varchar(20);default:'manual'" json:"source"`
}

func (MediaTag) TableName() string { return "media_tags" }

// Album 专题专辑
type Album struct {
	common.BaseModel
	Title       string `gorm:"type:varchar(500);not null" json:"title"`
	Overview    string `gorm:"type:text" json:"overview,omitempty"`
	PosterURL   string `gorm:"type:text" json:"poster_url,omitempty"`
	BackdropURL string `gorm:"type:text" json:"backdrop_url,omitempty"`
	AlbumType   string `gorm:"type:varchar(32);default:'collection'" json:"album_type"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
}

func (Album) TableName() string { return "albums" }

// AlbumItem 专辑条目
type AlbumItem struct {
	AlbumID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"album_id"`
	MediaID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"media_id"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	Note      string    `gorm:"type:varchar(200)" json:"note,omitempty"`
}

func (AlbumItem) TableName() string { return "album_items" }
