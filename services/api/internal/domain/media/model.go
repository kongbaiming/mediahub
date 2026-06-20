// Package media 是媒资领域模型
package media

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"

	"github.com/google/uuid"
)

// Media 媒资主表
type Media struct {
	common.BaseModel
	Type           common.MediaType `gorm:"type:varchar(20);not null;index" json:"type"`
	Title          string           `gorm:"type:varchar(500);not null;index" json:"title"`
	OriginalTitle  string           `gorm:"type:varchar(500)" json:"original_title,omitempty"`
	Year           *int             `gorm:"index" json:"year,omitempty"`
	Runtime        *int             `json:"runtime,omitempty"` // 分钟
	Overview       string           `gorm:"type:text" json:"overview,omitempty"`
	PosterURL      string           `gorm:"type:text" json:"poster_url,omitempty"`
	BackdropURL    string           `gorm:"type:text" json:"backdrop_url,omitempty"`
	Rating         float64          `gorm:"type:decimal(3,1);default:0;index" json:"rating"`
	VoteCount      int              `gorm:"default:0" json:"vote_count"`

	// 外部 ID
	TMDBID  *int    `gorm:"column:tmdb_id;uniqueIndex:idx_media_tmdb_unique,priority:2" json:"tmdb_id,omitempty"`
	DoubanID *string `gorm:"type:varchar(50);uniqueIndex:idx_media_douban_unique,priority:2" json:"douban_id,omitempty"`

	// 标签（Postgres text[]）
	Genres StringArray `gorm:"type:text[];default:'{}'" json:"genres"`
	Tags   StringArray `gorm:"type:text[];default:'{}'" json:"tags"`

	// 文件信息
	StoragePath string  `gorm:"type:text;not null" json:"storage_path"`
	FileSize    int64   `gorm:"default:0" json:"file_size"`
	VideoCodec  *string `gorm:"type:varchar(20)" json:"video_codec,omitempty"`
	AudioCodec  *string `gorm:"type:varchar(20)" json:"audio_codec,omitempty"`
	Resolution  *string `gorm:"type:varchar(20)" json:"resolution,omitempty"`
	HasSubtitle bool    `gorm:"default:false" json:"has_subtitle"`
	Container   *string `gorm:"type:varchar(20)" json:"container,omitempty"`

	// 内容分级（来自 TMDB 或手动标记）
	IsAdult bool `gorm:"default:false;index" json:"is_adult"` // 成人内容（R18+）

	// 刮削状态
	ScrapeStatus  common.ScrapeStatus `gorm:"type:varchar(20);default:'pending';index" json:"scrape_status"`
	ScrapeError   string              `gorm:"type:text" json:"scrape_error,omitempty"`
	LastScrapeAt  *time.Time          `json:"last_scrape_at,omitempty"`

	// 关联
	Seasons []Season `gorm:"foreignKey:MediaID;constraint:OnDelete:CASCADE" json:"seasons,omitempty"`
}

// TableName 表名
func (Media) TableName() string { return "media" }

// IsTV 是否剧集类
func (m *Media) IsTV() bool {
	return m.Type == common.MediaTypeTVShow || m.Type == common.MediaTypeAnime
}

// IsScraped 是否已刮削
func (m *Media) IsScraped() bool {
	return m.ScrapeStatus == common.ScrapeStatusDone
}

// DisplayTitle 展示标题（带年份）
func (m *Media) DisplayTitle() string {
	if m.Year != nil {
		return m.Title + " (" + itoa(*m.Year) + ")"
	}
	return m.Title
}

// RuntimeOrZero 取 Runtime（处理 nil）
func (m *Media) RuntimeOrZero() int {
	if m.Runtime == nil {
		return 0
	}
	return *m.Runtime
}

// Season 季
type Season struct {
	common.BaseModel
	MediaID      uuid.UUID `gorm:"type:uuid;not null;index" json:"media_id"`
	SeasonNumber int       `gorm:"not null" json:"season_number"`
	Title        string    `gorm:"type:varchar(500)" json:"title,omitempty"`
	Overview     string    `gorm:"type:text" json:"overview,omitempty"`
	PosterURL    string    `gorm:"type:text" json:"poster_url,omitempty"`
	AirDate      *time.Time `gorm:"type:date" json:"air_date,omitempty"`
	EpisodeCount int       `gorm:"default:0" json:"episode_count"`

	Episodes []Episode `gorm:"foreignKey:SeasonID;constraint:OnDelete:CASCADE" json:"episodes,omitempty"`
}

func (Season) TableName() string { return "seasons" }

// Episode 集
type Episode struct {
	common.BaseModel
	SeasonID      uuid.UUID `gorm:"type:uuid;not null;index" json:"season_id"`
	MediaID       uuid.UUID `gorm:"type:uuid;not null;index" json:"media_id"`
	EpisodeNumber int       `gorm:"not null" json:"episode_number"`
	Title         string    `gorm:"type:varchar(500)" json:"title,omitempty"`
	Overview      string    `gorm:"type:text" json:"overview,omitempty"`
	Duration      int       `gorm:"default:0" json:"duration"` // 秒
	FilePath      string    `gorm:"type:text" json:"file_path,omitempty"`
	FileSize      int64     `gorm:"default:0" json:"file_size"`
	StillURL      string    `gorm:"type:text" json:"still_url,omitempty"`
}

func (Episode) TableName() string { return "episodes" }

// ---- StringArray：Postgres text[] 支持 ----

// StringArray 字符串数组，兼容 JSON 序列化
type StringArray []string

// Scan 实现 sql.Scanner（从数据库读取）
func (s *StringArray) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return s.scanBytes(v)
	case string:
		return s.scanBytes([]byte(v))
	}
	return nil
}

// Value 实现 driver.Valuer（写入数据库）
func (s StringArray) Value() (any, error) {
	if s == nil {
		return "{}", nil
	}
	return s, nil
}

func (s *StringArray) scanBytes(b []byte) error {
	str := string(b)
	if str == "" || str == "{}" {
		*s = []string{}
		return nil
	}
	// 简单解析："{a,b,c}" -> ["a","b","c"]
	str = str[1 : len(str)-1]
	if str == "" {
		*s = []string{}
		return nil
	}
	var out []string
	var cur []byte
	inQuote := false
	for i := 0; i < len(str); i++ {
		c := str[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ',' && !inQuote:
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = cur[:0]
			}
		default:
			if inQuote || c != ' ' {
				cur = append(cur, c)
			}
		}
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	*s = out
	return nil
}

// MarshalJSON JSON 序列化
//
// 注意：必须先把 s 转成 []string 再交给 json.Marshal，否则
// json.Marshal 会看到参数实现了 json.Marshaler，再次调用
// s.MarshalJSON()，导致 stack overflow（fatal error: stack overflow）。
func (s StringArray) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return jsonMarshal([]string(s))
}

// UnmarshalJSON JSON 反序列化
func (s *StringArray) UnmarshalJSON(b []byte) error {
	return jsonUnmarshal(b, s)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
