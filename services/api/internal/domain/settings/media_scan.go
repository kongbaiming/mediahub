package settings

import "time"

// ScanStatus 扫描结果状态
type ScanStatus string

const (
	ScanStatusOK     ScanStatus = "ok"
	ScanStatusFailed ScanStatus = "failed"
)

// MediaScanConfig 媒资库自动扫描配置（单行 id=1）
type MediaScanConfig struct {
	ID               int16      `gorm:"primaryKey" json:"id"`
	Enabled          bool       `gorm:"not null;default:true" json:"enabled"`
	IntervalMinutes  int        `gorm:"not null;default:30" json:"interval_minutes"`
	LastScanAt       *time.Time `json:"last_scan_at,omitempty"`
	LastScanStatus   ScanStatus `gorm:"type:varchar(20)" json:"last_scan_status,omitempty"`
	LastScanMessage  string     `gorm:"type:text" json:"last_scan_message,omitempty"`
	LastScanAdded    int        `gorm:"not null;default:0" json:"last_scan_added"`
	LastScanTotal    int        `gorm:"not null;default:0" json:"last_scan_total"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (MediaScanConfig) TableName() string { return "media_scan_config" }

// AllowedScanIntervals 允许的扫描间隔（分钟）
var AllowedScanIntervals = []int{15, 30, 60, 360, 720, 1440, 10080}

func NormalizeScanInterval(minutes int) int {
	for _, v := range AllowedScanIntervals {
		if minutes == v {
			return v
		}
	}
	if minutes < 15 {
		return 15
	}
	return 30
}
