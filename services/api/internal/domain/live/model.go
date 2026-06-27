// Package live 直播间领域模型
package live

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"

	"github.com/google/uuid"
)

// RoomStatus 直播间状态
type RoomStatus string

const (
	StatusIdle   RoomStatus = "idle"
	StatusLive   RoomStatus = "live"
	StatusEnded  RoomStatus = "ended"
)

// Room 直播间
type Room struct {
	common.BaseModel
	Title       string     `gorm:"type:varchar(200);not null" json:"title"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	CoverURL    string     `gorm:"type:text" json:"cover_url,omitempty"`
	Status      RoomStatus `gorm:"type:varchar(20);not null;default:'idle';index" json:"status"`
	StreamKey   string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"stream_key"`
	ViewerCount int        `gorm:"default:0" json:"viewer_count"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
}

func (Room) TableName() string { return "live_rooms" }

// RoomView 对外展示的直播间信息（含推流/播放地址）
type RoomView struct {
	Room
	RTMPURL    string `json:"rtmp_url,omitempty"`
	HLSURL     string `json:"hls_url,omitempty"`
	PlayURL    string `json:"play_url,omitempty"`
	StreamPath string `json:"stream_path,omitempty"`
}
