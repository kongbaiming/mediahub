package catalog

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"

	"github.com/google/uuid"
)

// ContentRating 内容分级
type ContentRating struct {
	ID         uuid.UUID          `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MediaID    uuid.UUID          `gorm:"type:uuid;not null;index" json:"media_id"`
	Country    string             `gorm:"type:varchar(8);not null;default:'US'" json:"country"`
	System     string             `gorm:"type:varchar(32);not null;default:'tmdb'" json:"system"`
	Rating     string             `gorm:"type:varchar(32);not null" json:"rating"`
	Advisories media.StringArray  `gorm:"type:text[];default:'{}'" json:"advisories,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
}

func (ContentRating) TableName() string { return "content_ratings" }

// SubtitleTrack 字幕轨
type SubtitleTrack struct {
	common.BaseModel
	MediaFileID *uuid.UUID `gorm:"type:uuid;index" json:"media_file_id,omitempty"`
	MediaID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"media_id"`
	EpisodeID   *uuid.UUID `gorm:"type:uuid;index" json:"episode_id,omitempty"`
	Language    string     `gorm:"type:varchar(16);not null;default:'zh'" json:"language"`
	Format      string     `gorm:"type:varchar(16);not null;default:'srt'" json:"format"`
	Path        string     `gorm:"type:text" json:"path,omitempty"`
	Label       string     `gorm:"type:varchar(100)" json:"label,omitempty"`
	Source      string     `gorm:"type:varchar(20);default:'manual'" json:"source"`
	IsDefault   bool       `gorm:"default:false" json:"is_default"`
}

func (SubtitleTrack) TableName() string { return "subtitle_tracks" }

// MediaExtra 预告/花絮
type MediaExtra struct {
	common.BaseModel
	MediaID     uuid.UUID `gorm:"type:uuid;not null;index" json:"media_id"`
	ExtraType   string    `gorm:"type:varchar(32);not null;default:'trailer'" json:"extra_type"`
	Title       string    `gorm:"type:varchar(300)" json:"title,omitempty"`
	Source      string    `gorm:"type:varchar(20);default:'tmdb'" json:"source"`
	FilePath    string    `gorm:"type:text" json:"file_path,omitempty"`
	ExternalURL string    `gorm:"type:text" json:"external_url,omitempty"`
	ExternalKey string    `gorm:"type:varchar(64)" json:"external_key,omitempty"`
	DurationSec int       `gorm:"default:0" json:"duration_sec"`
}

func (MediaExtra) TableName() string { return "media_extras" }

// MediaArtwork 海报/背景变体
type MediaArtwork struct {
	common.BaseModel
	MediaID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"media_id"`
	SeasonID  *uuid.UUID `gorm:"type:uuid" json:"season_id,omitempty"`
	EpisodeID *uuid.UUID `gorm:"type:uuid" json:"episode_id,omitempty"`
	ArtType   string     `gorm:"type:varchar(32);not null" json:"art_type"`
	Locale    string     `gorm:"type:varchar(16);default:''" json:"locale,omitempty"`
	URL       string     `gorm:"type:text;not null" json:"url"`
	Width     int        `gorm:"default:0" json:"width,omitempty"`
	Height    int        `gorm:"default:0" json:"height,omitempty"`
}

func (MediaArtwork) TableName() string { return "media_artworks" }

// MediaRelation 作品关联
type MediaRelation struct {
	FromMediaID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"from_media_id"`
	ToMediaID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"to_media_id"`
	RelationType string    `gorm:"type:varchar(32);primaryKey" json:"relation_type"`
	SortOrder    int       `gorm:"default:0" json:"sort_order"`
}

func (MediaRelation) TableName() string { return "media_relations" }

// ProfileContentPolicy Profile 分龄策略
type ProfileContentPolicy struct {
	ProfileID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"profile_id"`
	MaxRatingLevel  int       `gorm:"default:100" json:"max_rating_level"`
	BlockAdult      bool      `gorm:"default:false" json:"block_adult"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ProfileContentPolicy) TableName() string { return "profile_content_policy" }
