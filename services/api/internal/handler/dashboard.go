package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/mediahub/api/pkg/logger"
	"gorm.io/gorm"
)

// DashboardHandler CMS 仪表盘 HTTP handler
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler 构造
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// Stats 返回媒资库和刮削队列统计
func (h *DashboardHandler) Stats(c *gin.Context) {
	ctx := c.Request.Context()

	// 媒资总数
	var totalMedia int64
	if err := h.db.WithContext(ctx).Table("media").Count(&totalMedia).Error; err != nil {
		logger.Warn("统计媒资总数失败", "err", err)
	}

	// 按类型统计
	type TypeCount struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	var typeCounts []TypeCount
	h.db.WithContext(ctx).Table("media").
		Select("type, count(*) as count").
		Group("type").Scan(&typeCounts)

	// 刮削状态统计
	type ScrapeCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var scrapeCounts []ScrapeCount
	h.db.WithContext(ctx).Table("media").
		Select("scrape_status as status, count(*) as count").
		Group("scrape_status").Scan(&scrapeCounts)

	// 可播状态统计
	type AvailCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var availCounts []AvailCount
	h.db.WithContext(ctx).Table("media").
		Select("availability_status as status, count(*) as count").
		Group("availability_status").Scan(&availCounts)

	// 最近 7 天入库趋势
	type DailyCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var dailyTrend []DailyCount
	h.db.WithContext(ctx).Table("media").
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at > NOW() - INTERVAL '7 days'").
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyTrend)

	// 总文件大小
	var totalSize int64
	h.db.WithContext(ctx).Table("media_files").
		Select("COALESCE(SUM(file_size), 0)").Scan(&totalSize)

	// Profile 数量
	var totalProfiles int64
	h.db.WithContext(ctx).Table("profiles").Count(&totalProfiles)

	c.JSON(200, gin.H{
		"data": gin.H{
			"total_media":    totalMedia,
			"type_counts":    typeCounts,
			"scrape_counts":  scrapeCounts,
			"avail_counts":   availCounts,
			"daily_trend":    dailyTrend,
			"total_size":     totalSize,
			"total_profiles": totalProfiles,
		},
	})
}

// Health 返回系统健康状态
func (h *DashboardHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()

	checks := gin.H{}

	// DB 连通性
	sqlDB, err := h.db.DB()
	if err != nil {
		checks["database"] = gin.H{"status": "error", "message": err.Error()}
	} else if err := sqlDB.PingContext(ctx); err != nil {
		checks["database"] = gin.H{"status": "error", "message": err.Error()}
	} else {
		checks["database"] = gin.H{"status": "ok"}
	}

	c.JSON(200, gin.H{"data": checks})
}
