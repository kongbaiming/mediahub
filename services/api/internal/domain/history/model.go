// Package history 是观看历史与收藏领域模型
package history

import (
	"encoding/json"
	"time"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"

	"github.com/google/uuid"
)

// History 播放历史（按 Profile 隔离）
type History struct {
	common.BaseModel
	ProfileID uuid.UUID      `gorm:"type:uuid;not null;index" json:"profile_id"`
	MediaID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"media_id"`
	EpisodeID *uuid.UUID     `gorm:"type:uuid;index" json:"episode_id,omitempty"`
	Progress  int            `gorm:"default:0" json:"progress"` // 秒
	Duration  int            `gorm:"default:0" json:"duration"`
	Completed bool           `gorm:"default:false" json:"completed"`
	Device    string         `gorm:"type:varchar(50)" json:"device,omitempty"`
	Media     *media.Media   `gorm:"foreignKey:MediaID" json:"media,omitempty"`
}

func (History) TableName() string { return "history" }

// ProgressPct 进度百分比
func (h *History) ProgressPct() float64 {
	if h.Duration <= 0 {
		return 0
	}
	return float64(h.Progress) / float64(h.Duration) * 100
}

// Favorite 收藏
type Favorite struct {
	common.BaseModel
	ProfileID    uuid.UUID            `gorm:"type:uuid;not null;index" json:"profile_id"`
	MediaID      uuid.UUID            `gorm:"type:uuid;not null;index" json:"media_id"`
	FavoriteType common.FavoriteType  `gorm:"type:varchar(20);default:'want'" json:"favorite_type"`
	Rating       *float64             `gorm:"type:decimal(3,1)" json:"rating,omitempty"`

	Media *media.Media `gorm:"foreignKey:MediaID" json:"media,omitempty"`
}

func (Favorite) TableName() string { return "favorites" }

// Recommendation 推荐缓存
type Recommendation struct {
	common.BaseModel
	ProfileID  uuid.UUID  `gorm:"type:uuid;index" json:"profile_id"`                 // 可为空（全局推荐）
	Algo       string     `gorm:"type:varchar(50);not null;index" json:"algo"`      // hot | similar | cf | content | hybrid
	MediaIDs   UUIDArray  `gorm:"type:uuid[];not null;default:'{}'" json:"media_ids"`
	ComputedAt time.Time  `gorm:"not null" json:"computed_at"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
}

func (Recommendation) TableName() string { return "recommendations" }

// UUIDArray uuid 数组（Postgres uuid[]）
type UUIDArray []uuid.UUID

// Scan 实现 sql.Scanner
func (u *UUIDArray) Scan(src any) error {
	if src == nil {
		*u = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	s := string(b)
	if s == "" || s == "{}" {
		*u = []uuid.UUID{}
		return nil
	}
	// 解析 "{uuid1,uuid2}" 格式
	s = s[1 : len(s)-1]
	if s == "" {
		*u = []uuid.UUID{}
		return nil
	}
	out := []uuid.UUID{}
	curr := []byte{}
	for _, c := range s {
		if c == ',' {
			if len(curr) > 0 {
				id, err := uuid.Parse(string(curr))
				if err == nil {
					out = append(out, id)
				}
				curr = curr[:0]
			}
		} else if c != ' ' {
			curr = append(curr, byte(c))
		}
	}
	if len(curr) > 0 {
		id, err := uuid.Parse(string(curr))
		if err == nil {
			out = append(out, id)
		}
	}
	*u = out
	return nil
}

// Value 实现 driver.Valuer
func (u UUIDArray) Value() (any, error) {
	if u == nil {
		return "{}", nil
	}
	return u, nil
}

// MarshalJSON JSON 序列化（避免 stack overflow）
//
// json.Marshal 会识别实现了 json.Marshaler 的类型并再次调用它的 MarshalJSON。
// 直接把 UUIDArray 传进去会无限递归。必须先转成 []uuid.UUID（不是自定义类型）。
func (u UUIDArray) MarshalJSON() ([]byte, error) {
	if u == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]uuid.UUID(u))
}
