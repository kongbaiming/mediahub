package recommend

import (
	"time"

	"github.com/google/uuid"
)

// Config 推荐权重配置（单行表，id=1）
type Config struct {
	ID                 int     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContentWeight      float32 `gorm:"default:0.4" json:"content_weight"`
	CFWeight           float32 `gorm:"default:0.4" json:"cf_weight"`
	PopularityWeight   float32 `gorm:"default:0.2" json:"popularity_weight"`
	CFMinCooccurrence  int     `gorm:"default:3" json:"cf_min_cooccurrence"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (Config) TableName() string { return "recommend_config" }

// CFSimilarity 协同过滤共现相似度
type CFSimilarity struct {
	MediaAID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"media_a_id"`
	MediaBID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"media_b_id"`
	Score     float32   `gorm:"not null" json:"score"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CFSimilarity) TableName() string { return "cf_similarity" }

// UserList 用户自建片单
type UserList struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProfileID   uuid.UUID `gorm:"type:uuid;not null;index" json:"profile_id"`
	Name        string    `gorm:"type:varchar(128);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CoverURL    string    `gorm:"type:text" json:"cover_url,omitempty"`
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Items       []UserListItem `gorm:"foreignKey:ListID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

func (UserList) TableName() string { return "user_lists" }

// UserListItem 片单条目
type UserListItem struct {
	ListID    int64     `gorm:"primaryKey;autoIncrement:false" json:"list_id"`
	MediaID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"media_id"`
	SortOrder int16     `gorm:"default:0" json:"sort_order"`
	AddedAt   time.Time `gorm:"default:now()" json:"added_at"`
}

func (UserListItem) TableName() string { return "user_list_items" }
