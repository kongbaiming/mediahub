package service

import (
	"context"
	"errors"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/user"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ProfileService Profile 业务
type ProfileService struct {
	users *repository.UserRepo
}

// NewProfileService 构造
func NewProfileService(users *repository.UserRepo) *ProfileService {
	return &ProfileService{users: users}
}

// CreateProfileRequest 创建 Profile 请求
type CreateProfileRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=50"`
	Avatar  string `json:"avatar,omitempty"`
	IsKid   bool   `json:"is_kid"`
	Pin     string `json:"pin,omitempty"` // 家长锁 PIN（4-8 位）
}

// UpdateProfileRequest 更新 Profile 请求
type UpdateProfileRequest struct {
	Name   *string `json:"name,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
	IsKid  *bool   `json:"is_kid,omitempty"`
	Pin    *string `json:"pin,omitempty"` // 传空字符串表示清除
}

const webPlayerOwnerUsername = "admin"

func (s *ProfileService) webPlayerOwnerID(ctx context.Context) (string, error) {
	u, err := s.users.GetByUsername(ctx, webPlayerOwnerUsername)
	if err != nil {
		return "", err
	}
	return u.ID.String(), nil
}

// ListForWebPlayer 列出 Web 播放端可用 Profile（绑定 admin 账号下的家庭成员）
func (s *ProfileService) ListForWebPlayer(ctx context.Context) ([]user.Profile, error) {
	uid, err := s.webPlayerOwnerID(ctx)
	if err != nil {
		return nil, err
	}
	return s.ListMyProfiles(ctx, uid)
}

// CreateForWebPlayer 为 Web 播放端创建 Profile
func (s *ProfileService) CreateForWebPlayer(ctx context.Context, req CreateProfileRequest) (*user.Profile, error) {
	uid, err := s.webPlayerOwnerID(ctx)
	if err != nil {
		return nil, err
	}
	return s.Create(ctx, uid, req)
}

// ListMyProfiles 列出当前用户的所有 Profile
func (s *ProfileService) ListMyProfiles(ctx context.Context, userID string) ([]user.Profile, error) {
	return s.users.ListProfilesByUser(ctx, userID)
}

// Create 创建 Profile
func (s *ProfileService) Create(ctx context.Context, userID string, req CreateProfileRequest) (*user.Profile, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperr.Validation(map[string]string{"user_id": "格式错误"})
	}
	if req.Name == "" {
		return nil, apperr.Validation(map[string]string{"name": "名称不能为空"})
	}

	// 限制 Profile 数量（每个家庭最多 8 个）
	existing, _ := s.users.ListProfilesByUser(ctx, userID)
	if len(existing) >= 8 {
		return nil, apperr.Validation(map[string]string{"profile": "家庭成员已达上限（8 个）"})
	}

	p := &user.Profile{
		UserID:    uid,
		Name:      req.Name,
		AvatarURL: req.Avatar,
		IsKid:     req.IsKid,
	}
	if req.Pin != "" {
		if len(req.Pin) < 4 || len(req.Pin) > 8 {
			return nil, apperr.Validation(map[string]string{"pin": "PIN 长度必须 4-8 位"})
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
		if err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "PIN 哈希失败")
		}
		p.PinHash = string(hash)
	}
	if err := s.users.CreateProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Update 更新 Profile
func (s *ProfileService) Update(ctx context.Context, userID, profileID string, req UpdateProfileRequest) (*user.Profile, error) {
	p, err := s.users.GetProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	// 校验归属
	if p.UserID.String() != userID {
		return nil, apperr.Forbidden("无权修改此 Profile")
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, apperr.Validation(map[string]string{"name": "名称不能为空"})
		}
		p.Name = *req.Name
	}
	if req.Avatar != nil {
		p.AvatarURL = *req.Avatar
	}
	if req.IsKid != nil {
		p.IsKid = *req.IsKid
	}
	if req.Pin != nil {
		if *req.Pin == "" {
			p.PinHash = "" // 清除 PIN
		} else {
			if len(*req.Pin) < 4 || len(*req.Pin) > 8 {
				return nil, apperr.Validation(map[string]string{"pin": "PIN 长度必须 4-8 位"})
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(*req.Pin), bcrypt.DefaultCost)
			if err != nil {
				return nil, apperr.Wrap(err, apperr.CodeInternal, "PIN 哈希失败")
			}
			p.PinHash = string(hash)
		}
	}

	if err := s.users.UpdateProfile(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete 删除 Profile
func (s *ProfileService) Delete(ctx context.Context, userID, profileID string) error {
	p, err := s.users.GetProfile(ctx, profileID)
	if err != nil {
		return err
	}
	if p.UserID.String() != userID {
		return apperr.Forbidden("无权删除此 Profile")
	}

	// 至少保留一个 Profile
	all, _ := s.users.ListProfilesByUser(ctx, userID)
	if len(all) <= 1 {
		return apperr.Validation(map[string]string{"profile": "至少保留一个 Profile"})
	}

	return s.users.DeleteProfile(ctx, profileID)
}

// VerifyPin 验证家长锁 PIN（儿童 Profile 切换时）
func (s *ProfileService) VerifyPin(ctx context.Context, profileID, pin string) error {
	p, err := s.users.GetProfile(ctx, profileID)
	if err != nil {
		return err
	}
	if !p.IsKid {
		return errors.New("非儿童 Profile 无需 PIN")
	}
	if p.PinHash == "" {
		// 没设 PIN，直接通过
		return nil
	}
	if err := bcrypt.CompareHashAndPassword([]byte(p.PinHash), []byte(pin)); err != nil {
		return apperr.Unauthorized("PIN 错误")
	}
	return nil
}
