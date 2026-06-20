package repository

import (
	"context"
	"errors"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/user"

	"gorm.io/gorm"
)

// UserRepo 用户仓储
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 构造用户仓储
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// GetByUsername 按用户名查询
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).
		Preload("Profiles").
		Where("username = ?", username).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("用户不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询用户失败")
	}
	return &u, nil
}

// GetByID 按 ID 查询
func (r *UserRepo) GetByID(ctx context.Context, id string) (*user.User, error) {
	var u user.User
	err := r.db.WithContext(ctx).
		Preload("Profiles").
		First(&u, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("用户不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询用户失败")
	}
	return &u, nil
}

// Create 创建用户
func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建用户失败")
	}
	return nil
}

// CreateProfile 创建 Profile
func (r *UserRepo) CreateProfile(ctx context.Context, p *user.Profile) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建 Profile 失败")
	}
	return nil
}

// GetProfile 按 ID 查询
func (r *UserRepo) GetProfile(ctx context.Context, id string) (*user.Profile, error) {
	var p user.Profile
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Profile 不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询 Profile 失败")
	}
	return &p, nil
}

// ListProfilesByUser 列出用户的所有 Profile
func (r *UserRepo) ListProfilesByUser(ctx context.Context, userID string) ([]user.Profile, error) {
	var ps []user.Profile
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&ps).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询 Profile 失败")
	}
	return ps, nil
}

// UpdateProfile 更新 Profile
func (r *UserRepo) UpdateProfile(ctx context.Context, p *user.Profile) error {
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "更新 Profile 失败")
	}
	return nil
}

// DeleteProfile 删除 Profile
func (r *UserRepo) DeleteProfile(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&user.Profile{}, "id = ?", id).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "删除 Profile 失败")
	}
	return nil
}

// ListProfilesByIDs 按 ID 批量查询
func (r *UserRepo) ListProfilesByIDs(ctx context.Context, ids []string) ([]user.Profile, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var ps []user.Profile
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&ps).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询 Profile 失败")
	}
	return ps, nil
}
