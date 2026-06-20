// Package layout 是布局领域模型
package layout

import (
	"time"

	"github.com/mediahub/api/internal/domain/common"

	"github.com/google/uuid"
)

// Layout 布局（核心：CMS 编辑的对象）
type Layout struct {
	common.BaseModel
	ParentID         *uuid.UUID          `gorm:"type:uuid;index" json:"parent_id,omitempty"` // 模板继承
	Name             string              `gorm:"type:varchar(200);not null" json:"name"`
	Description      string              `gorm:"type:text" json:"description,omitempty"`
	IsTemplate       bool                `gorm:"default:false;index" json:"is_template"`
	Version          int                 `gorm:"default:1" json:"version"`
	Status           common.LayoutStatus `gorm:"type:varchar(20);default:'draft';index" json:"status"`
	Config           LayoutConfig        `gorm:"type:jsonb;default:'{}'::jsonb" json:"config"`
	LastPublishedAt  *time.Time          `gorm:"column:last_published_at" json:"last_published_at,omitempty"`

	Publications []Publication `gorm:"foreignKey:LayoutID;constraint:OnDelete:CASCADE" json:"publications,omitempty"`
}

func (Layout) TableName() string { return "layouts" }

// LayoutConfig 布局配置（rows 数组）
// 这是 CMS 后台编辑的核心数据结构
type LayoutConfig struct {
	Theme  string             `json:"theme,omitempty"`  // dark | light
	Rows   []Row              `json:"rows"`              // 行数组（顺序敏感）
	Global map[string]any     `json:"global,omitempty"`  // 全局配置
}

// Row 布局中的一行
type Row struct {
	ID        string         `json:"id"`                   // 唯一标识（前端生成）
	Type      string         `json:"type"`                 // hero-banner | shelf | category-grid | topic | text-banner | divider
	Title     string         `json:"title,omitempty"`
	Subtitle  string         `json:"subtitle,omitempty"`
	CardStyle string         `json:"card_style,omitempty"` // poster | landscape | square | banner
	Source    DataSource     `json:"source"`               // 数据源
	Visible   *bool          `json:"visible,omitempty"`    // 是否显示（省略=显示）
	Config    map[string]any `json:"config,omitempty"`     // 额外配置
	Inherited bool           `json:"_inherited,omitempty"` // 编辑器：来自父布局（不持久化）
}

// DataSource 数据源（10 种类型）
// 用 JSON oneOf 表达
type DataSource struct {
	Type   string         `json:"type"`               // manual | library | tag | trending | similar-to | continue-watching | recently-added | recommend-algorithm | union | exclude
	Params map[string]any `json:"params,omitempty"`   // 各类型参数
}

// Publication 布局发布（多端 + AB + 动态规则）
type Publication struct {
	common.BaseModel
	LayoutID       uuid.UUID         `gorm:"type:uuid;not null;index" json:"layout_id"`
	Version        int               `gorm:"not null" json:"version"`
	TargetPlatform common.Platform   `gorm:"type:varchar(20);not null" json:"target_platform"`
	TrafficSplit   TrafficSplit      `gorm:"type:jsonb" json:"traffic_split,omitempty"` // AB 权重 {"A":50,"B":50}
	Enabled        bool              `gorm:"default:true" json:"enabled"`
	ActiveFrom     *time.Time        `json:"active_from,omitempty"`
	ActiveTo       *time.Time        `json:"active_to,omitempty"`

	// 动态规则（按时间段 / 星期几筛选）
	DynamicRules *DynamicRules `gorm:"type:jsonb" json:"dynamic_rules,omitempty"`
}

// DynamicRules 动态布局规则
type DynamicRules struct {
	// 时段（24h），如 [18, 24] 表示 18:00-24:00 启用
	HourOfDay *HourRange `json:"hour_of_day,omitempty"`
	// 星期几（0=周日，6=周六），如 [1,2,3,4,5] 表示工作日
	DayOfWeek []int `json:"day_of_week,omitempty"`
	// 季节（1=春，2=夏，3=秋，4=冬）
	Seasons []int `json:"seasons,omitempty"`
	// 自定义标签（如 "holiday", "weekend"）
	Tags []string `json:"tags,omitempty"`
}

// HourRange 时段范围
type HourRange struct {
	Start int `json:"start"` // 0-23
	End   int `json:"end"`   // 0-23, 包含
}

// Matches 是否在当前时刻匹配规则
func (d *DynamicRules) Matches(now time.Time) bool {
	if d == nil {
		return true
	}

	// 时段
	if d.HourOfDay != nil {
		h := now.Hour()
		if d.HourOfDay.Start <= d.HourOfDay.End {
			if h < d.HourOfDay.Start || h > d.HourOfDay.End {
				return false
			}
		} else {
			// 跨夜，如 22-6
			if h < d.HourOfDay.Start && h > d.HourOfDay.End {
				return false
			}
		}
	}

	// 星期几
	if len(d.DayOfWeek) > 0 {
		w := int(now.Weekday())
		found := false
		for _, dw := range d.DayOfWeek {
			if dw == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (Publication) TableName() string { return "layout_publications" }

// RowIsVisible 行是否可见（省略 visible 时默认显示）
func RowIsVisible(r Row) bool {
	return r.Visible == nil || *r.Visible
}
type FeedRow struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title,omitempty"`
	Subtitle  string         `json:"subtitle,omitempty"`
	CardStyle string         `json:"card_style,omitempty"`
	Items     []FeedItem     `json:"items"`
	Config    map[string]any `json:"config,omitempty"`
}

// FeedItem 播放端单个卡片
type FeedItem struct {
	MediaID   uuid.UUID `json:"media_id"`
	Title     string    `json:"title"`
	Year      *int      `json:"year,omitempty"`
	PosterURL string    `json:"poster_url,omitempty"`
	BackdropURL string  `json:"backdrop_url,omitempty"`
	Rating    float64   `json:"rating"`
	Type      string    `json:"type"`
	Duration  *int      `json:"duration,omitempty"`
	Overview  string    `json:"overview,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Progress  *int      `json:"progress,omitempty"` // 续播进度（秒）
}

// Feed 播放端拉取的完整布局
type Feed struct {
	Version   int       `json:"version"`
	Platform  string    `json:"platform"`
	UpdatedAt time.Time `json:"updated_at"`
	Rows      []FeedRow `json:"rows"`
}
