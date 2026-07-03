package repository

import (
	"context"
	"errors"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/recommend"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserListRepo 用户片单仓储
type UserListRepo struct {
	db *gorm.DB
}

// NewUserListRepo 构造
func NewUserListRepo(db *gorm.DB) *UserListRepo {
	return &UserListRepo{db: db}
}

// ListByProfile 列出用户的片单
func (r *UserListRepo) ListByProfile(ctx context.Context, profileID uuid.UUID) ([]recommend.UserList, error) {
	var out []recommend.UserList
	if err := r.db.WithContext(ctx).
		Where("profile_id = ?", profileID).
		Order("updated_at DESC").
		Find(&out).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询片单失败")
	}
	return out, nil
}

// GetByID 获取片单详情（含条目）
func (r *UserListRepo) GetByID(ctx context.Context, id int64) (*recommend.UserList, error) {
	var ul recommend.UserList
	if err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, added_at ASC")
		}).
		First(&ul, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("片单不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询片单失败")
	}
	return &ul, nil
}

// Create 创建片单
func (r *UserListRepo) Create(ctx context.Context, ul *recommend.UserList) error {
	if err := r.db.WithContext(ctx).Create(ul).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建片单失败")
	}
	return nil
}

// Update 更新片单
func (r *UserListRepo) Update(ctx context.Context, id int64, name, description *string, isPublic *bool) error {
	updates := map[string]any{}
	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	if isPublic != nil {
		updates["is_public"] = *isPublic
	}
	if len(updates) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Model(&recommend.UserList{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "更新片单失败")
	}
	return nil
}

// Delete 删除片单
func (r *UserListRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&recommend.UserList{}).Error
}

// AddItem 添加条目
func (r *UserListRepo) AddItem(ctx context.Context, listID int64, mediaID uuid.UUID) error {
	item := recommend.UserListItem{ListID: listID, MediaID: mediaID}
	return r.db.WithContext(ctx).
		Where("list_id = ? AND media_id = ?", listID, mediaID).
		FirstOrCreate(&item).Error
}

// RemoveItem 移除条目
func (r *UserListRepo) RemoveItem(ctx context.Context, listID int64, mediaID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("list_id = ? AND media_id = ?", listID, mediaID).
		Delete(&recommend.UserListItem{}).Error
}
