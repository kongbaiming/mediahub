package media

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"

	"github.com/google/uuid"
)

// MediaKind 作品结构类型
type MediaKind string

const (
	MediaKindSingle MediaKind = "single" // 电影、纪录片等单文件作品
	MediaKindSeries MediaKind = "series" // 剧集、动漫（季/集）
)

// MediaFile 物理文件层（播放与 ffprobe 的事实源）
type MediaFile struct {
	common.BaseModel
	MediaID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"media_id"`
	EpisodeID  *uuid.UUID `gorm:"type:uuid;index" json:"episode_id,omitempty"`
	Path       string     `gorm:"type:text;not null;uniqueIndex" json:"path"`
	FileSize   int64      `gorm:"default:0" json:"file_size"`
	DurationSec int       `gorm:"default:0" json:"duration_sec"`
	VideoCodec *string    `gorm:"type:varchar(32)" json:"video_codec,omitempty"`
	AudioCodec *string    `gorm:"type:varchar(32)" json:"audio_codec,omitempty"`
	Width      int        `gorm:"default:0" json:"width,omitempty"`
	Height     int        `gorm:"default:0" json:"height,omitempty"`
	Resolution *string    `gorm:"type:varchar(20)" json:"resolution,omitempty"`
	Container  *string    `gorm:"type:varchar(20)" json:"container,omitempty"`
	HasSubtitle bool      `gorm:"default:false" json:"has_subtitle"`
	IsPrimary  bool       `gorm:"default:true" json:"is_primary"`
	ProbeStatus string    `gorm:"type:varchar(20);default:'pending';index" json:"probe_status"`
	ProbedAt   *time.Time `json:"probed_at,omitempty"`
	Source     string     `gorm:"type:varchar(20);default:'scan'" json:"source"`
}

func (MediaFile) TableName() string { return "media_files" }

// PrimaryFile 从文件列表中取主播放文件（优先 is_primary，否则取最高分辨率）
func PrimaryFile(files []MediaFile) *MediaFile {
	if len(files) == 0 {
		return nil
	}
	var primary *MediaFile
	var best *MediaFile
	for i := range files {
		f := &files[i]
		if f.IsPrimary {
			if primary == nil || f.Height > primary.Height {
				primary = f
			}
		}
		if best == nil || f.Height > best.Height {
			best = f
		}
	}
	if primary != nil {
		return primary
	}
	return best
}

// PlayablePath 解析可播放路径（文件层优先，兼容旧列）
func (m *Media) PlayablePath() string {
	if m.IsTV() {
		return ""
	}
	if f := PrimaryFile(m.Files); f != nil {
		return f.Path
	}
	return m.StoragePath
}

// PlayablePath 单集可播放路径
func (e *Episode) PlayablePath() string {
	if f := PrimaryFile(e.Files); f != nil {
		return f.Path
	}
	return e.FilePath
}
