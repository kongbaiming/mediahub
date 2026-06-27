package repository

import (
	"context"
	"errors"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"

	"gorm.io/gorm"
)

// LiveRepo 直播间仓储
type LiveRepo struct {
	db *gorm.DB
}

// NewLiveRepo 构造
func NewLiveRepo(db *gorm.DB) *LiveRepo {
	return &LiveRepo{db: db}
}

// Create 创建直播间
func (r *LiveRepo) Create(ctx context.Context, room *live.Room) error {
	if err := r.db.WithContext(ctx).Create(room).Error; err != nil {
		return wrapDBErr(err, "创建直播间失败")
	}
	return nil
}

// GetByID 按 ID 获取
func (r *LiveRepo) GetByID(ctx context.Context, id string) (*live.Room, error) {
	var room live.Room
	if err := r.db.WithContext(ctx).First(&room, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("直播间不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询直播间失败")
	}
	return &room, nil
}

// GetByStreamKey 按推流密钥获取
func (r *LiveRepo) GetByStreamKey(ctx context.Context, key string) (*live.Room, error) {
	var room live.Room
	if err := r.db.WithContext(ctx).First(&room, "stream_key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("直播间不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询直播间失败")
	}
	return &room, nil
}

// List 列表
func (r *LiveRepo) List(ctx context.Context, status string, limit, offset int) ([]live.Room, int64, error) {
	q := r.db.WithContext(ctx).Model(&live.Room{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(err, apperr.CodeInternal, "统计直播间失败")
	}
	var items []live.Room
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, apperr.Wrap(err, apperr.CodeInternal, "查询直播间列表失败")
	}
	return items, total, nil
}

// Update 更新
func (r *LiveRepo) Update(ctx context.Context, room *live.Room) error {
	if err := r.db.WithContext(ctx).Save(room).Error; err != nil {
		return wrapDBErr(err, "更新直播间失败")
	}
	return nil
}

// Delete 软删除
func (r *LiveRepo) Delete(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Delete(&live.Room{}, "id = ?", id)
	if res.Error != nil {
		return wrapDBErr(res.Error, "删除直播间失败")
	}
	if res.RowsAffected == 0 {
		return apperr.NotFound("直播间不存在")
	}
	return nil
}

// DeleteIPTVByPlaylistURL 删除指定 M3U 来源的 IPTV 频道
func (r *LiveRepo) DeleteIPTVByPlaylistURL(ctx context.Context, playlistURL string) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("room_type = ? AND playlist_url = ?", live.RoomTypeIPTV, playlistURL).
		Delete(&live.Room{})
	if res.Error != nil {
		return 0, wrapDBErr(res.Error, "删除 M3U 频道失败")
	}
	return res.RowsAffected, nil
}

// ListIPTVSourceURLs 列出已有 IPTV 源地址（去重用）
func (r *LiveRepo) ListIPTVSourceURLs(ctx context.Context) ([]string, error) {
	var urls []string
	err := r.db.WithContext(ctx).Model(&live.Room{}).
		Where("room_type = ?", live.RoomTypeIPTV).
		Where("source_url <> ''").
		Pluck("source_url", &urls).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询 IPTV 源失败")
	}
	return urls, nil
}
