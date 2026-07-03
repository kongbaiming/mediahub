package handler

import (
	"time"

	"github.com/mediahub/api/internal/apperr"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnalyticsHandler 数据统计 HTTP handler
type AnalyticsHandler struct {
	db *gorm.DB
}

// NewAnalyticsHandler 构造
func NewAnalyticsHandler(db *gorm.DB) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

// RecordEvent 记录 Feed 曝光/点击事件（匿名）
func (h *AnalyticsHandler) RecordEvent(c *gin.Context) {
	var req struct {
		PublicationID *string `json:"publication_id"`
		Variant       string  `json:"variant"`
		EventType     string  `json:"event_type" binding:"required"` // impression, click, play
		RowKey        *string `json:"row_key"`
		MediaID       *string `json:"media_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}

	profileID := c.GetHeader("X-Profile-ID")

	type abEvent struct {
		PublicationID *uuid.UUID `gorm:"type:uuid"`
		Variant       string
		ProfileID     *uuid.UUID `gorm:"type:uuid"`
		EventType     string
		RowKey        *string
		MediaID       *uuid.UUID `gorm:"type:uuid"`
		CreatedAt     time.Time
	}

	event := abEvent{
		Variant:   req.Variant,
		EventType: req.EventType,
		RowKey:    req.RowKey,
		CreatedAt: time.Now(),
	}

	if req.PublicationID != nil {
		if pid, err := uuid.Parse(*req.PublicationID); err == nil {
			event.PublicationID = &pid
		}
	}
	if profileID != "" {
		if pid, err := uuid.Parse(profileID); err == nil {
			event.ProfileID = &pid
		}
	}
	if req.MediaID != nil {
		if mid, err := uuid.Parse(*req.MediaID); err == nil {
			event.MediaID = &mid
		}
	}

	if err := h.db.WithContext(c.Request.Context()).Table("ab_test_events").Create(&event).Error; err != nil {
		respondError(c, apperr.Internal("记录事件失败"))
		return
	}

	c.JSON(200, gin.H{"status": "recorded"})
}

// FeedStats Feed 行曝光/点击统计
func (h *AnalyticsHandler) FeedStats(c *gin.Context) {
	days := 7

	type RowStat struct {
		RowKey    string `json:"row_key"`
		EventType string `json:"event_type"`
		Count     int64  `json:"count"`
	}
	var stats []RowStat
	h.db.WithContext(c.Request.Context()).Table("ab_test_events").
		Select("row_key, event_type, count(*) as count").
		Where("created_at > NOW() - (? || ' days')::interval", days).
		Where("row_key IS NOT NULL").
		Group("row_key, event_type").
		Order("count DESC").
		Scan(&stats)

	c.JSON(200, gin.H{"data": stats, "days": days})
}
