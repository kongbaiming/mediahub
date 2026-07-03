package service

import (
	"context"

	"github.com/mediahub/api/internal/apperr"
	recommendDomain "github.com/mediahub/api/internal/domain/recommend"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserListService 用户片单业务
type UserListService struct {
	db   *gorm.DB
	repo *repository.UserListRepo
}

// NewUserListService 构造
func NewUserListService(db *gorm.DB, repo *repository.UserListRepo) *UserListService {
	return &UserListService{db: db, repo: repo}
}

// List 列出用户的片单
func (s *UserListService) List(ctx context.Context, profileID string) ([]recommendDomain.UserList, error) {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return nil, apperr.Validation("invalid profile_id")
	}
	return s.repo.ListByProfile(ctx, pid)
}

// Get 获取片单详情（含所有权校验）
func (s *UserListService) Get(ctx context.Context, profileID string, id int64) (*recommendDomain.UserList, error) {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return nil, apperr.Validation("invalid profile_id")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.ProfileID != pid {
		return nil, apperr.NotFound("片单不存在")
	}
	return item, nil
}

// Create 创建片单
func (s *UserListService) Create(ctx context.Context, profileID, name, description string, isPublic bool) (*recommendDomain.UserList, error) {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return nil, apperr.Validation("invalid profile_id")
	}
	ul := &recommendDomain.UserList{
		ProfileID:   pid,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
	}
	if err := s.repo.Create(ctx, ul); err != nil {
		return nil, err
	}
	return ul, nil
}

// Update 更新片单
func (s *UserListService) Update(ctx context.Context, profileID string, id int64, name, description *string, isPublic *bool) error {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return apperr.Validation("invalid profile_id")
	}
	// 验证所有权
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ProfileID != pid {
		return apperr.Forbidden("无权修改此片单")
	}
	return s.repo.Update(ctx, id, name, description, isPublic)
}

// Delete 删除片单
func (s *UserListService) Delete(ctx context.Context, profileID string, id int64) error {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return apperr.Validation("invalid profile_id")
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ProfileID != pid {
		return apperr.Forbidden("无权删除此片单")
	}
	return s.repo.Delete(ctx, id)
}

// AddItem 添加媒资到片单
func (s *UserListService) AddItem(ctx context.Context, profileID string, listID int64, mediaIDStr string) error {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return apperr.Validation("invalid profile_id")
	}
	existing, err := s.repo.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if existing.ProfileID != pid {
		return apperr.Forbidden("无权修改此片单")
	}
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		return apperr.Validation("invalid media_id")
	}
	return s.repo.AddItem(ctx, listID, mediaID)
}

// RemoveItem 从片单移除媒资
func (s *UserListService) RemoveItem(ctx context.Context, profileID string, listID int64, mediaIDStr string) error {
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return apperr.Validation("invalid profile_id")
	}
	existing, err := s.repo.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if existing.ProfileID != pid {
		return apperr.Forbidden("无权修改此片单")
	}
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		return apperr.Validation("invalid media_id")
	}
	return s.repo.RemoveItem(ctx, listID, mediaID)
}
