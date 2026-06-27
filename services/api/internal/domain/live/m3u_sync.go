package live

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"
)

// M3USyncStatus 同步结果
type M3USyncStatus string

const (
	SyncStatusOK     M3USyncStatus = "ok"
	SyncStatusFailed M3USyncStatus = "failed"
)

// M3USyncJob M3U 自动同步任务
type M3USyncJob struct {
	common.BaseModel
	PlaylistURL     string        `gorm:"type:text;not null;uniqueIndex" json:"playlist_url"`
	Enabled         bool          `gorm:"not null;default:true" json:"enabled"`
	IntervalMinutes int           `gorm:"not null;default:1440" json:"interval_minutes"`
	LastSyncAt      *time.Time    `json:"last_sync_at,omitempty"`
	LastSyncStatus  M3USyncStatus `gorm:"type:varchar(20)" json:"last_sync_status,omitempty"`
	LastSyncMessage string        `gorm:"type:text" json:"last_sync_message,omitempty"`
}

func (M3USyncJob) TableName() string { return "live_m3u_sync_jobs" }

// AllowedSyncIntervals 允许的同步间隔（分钟）
var AllowedSyncIntervals = []int{60, 360, 720, 1440, 10080}

func NormalizeSyncInterval(minutes int) int {
	for _, v := range AllowedSyncIntervals {
		if minutes == v {
			return v
		}
	}
	if minutes < 60 {
		return 60
	}
	return 1440
}
