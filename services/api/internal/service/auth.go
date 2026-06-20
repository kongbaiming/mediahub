package service

import (
	"context"
	"errors"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/user"
	"github.com/mediahub/api/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证业务
type AuthService struct {
	users     *repository.UserRepo
	jwtSecret []byte
	expire    time.Duration
}

// NewAuthService 构造
func NewAuthService(users *repository.UserRepo, jwtSecret string) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		expire:    30 * 24 * time.Hour, // 30 天
	}
}

// Users 返回用户仓储（供 handler Me 用）
func (s *AuthService) Users() *repository.UserRepo {
	return s.users
}

// Claims JWT 载荷
type Claims struct {
	UserID    string `json:"uid"`
	Username  string `json:"un"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    string      `json:"token"`
	User     *user.User  `json:"user"`
	Profiles []user.Profile `json:"profiles"`
}

// RegisterRequest 注册请求（家庭账号唯一，无须重复注册）
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=6"`
	DisplayName string `json:"display_name"`
}

// Login 登录
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	u, err := s.users.GetByUsername(ctx, req.Username)
	if err != nil {
		if ae, ok := apperr.As(err); ok && ae.Code == apperr.CodeNotFound {
			return nil, apperr.Unauthorized("用户名或密码错误")
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, apperr.Forbidden("账号已禁用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperr.Unauthorized("用户名或密码错误")
	}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:    token,
		User:     u,
		Profiles: u.Profiles,
	}, nil
}

// Register 注册（创建家庭账号 + 默认 Profile）
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*LoginResponse, error) {
	// 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "密码哈希失败")
	}

	u := &user.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		Role:         "admin", // 第一个用户为管理员
		IsActive:     true,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}

	// 创建默认 Profile
	p := &user.Profile{
		UserID: u.ID,
		Name:   "我",
	}
	if err := s.users.CreateProfile(ctx, p); err != nil {
		return nil, err
	}
	u.Profiles = []user.Profile{*p}

	token, err := s.generateToken(u)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:    token,
		User:     u,
		Profiles: u.Profiles,
	}, nil
}

// generateToken 签发 JWT
func (s *AuthService) generateToken(u *user.User) (string, error) {
	claims := &Claims{
		UserID:   u.ID.String(),
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mediahub",
			Subject:   u.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ParseToken 解析 + 校验
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("非预期签名方法")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if c, ok := t.Claims.(*Claims); ok && t.Valid {
		return c, nil
	}
	return nil, errors.New("无效 token")
}

// EnsureDefaultAdmin 启动时确保存在默认管理员（首次启动）
//
// 行为：
//   - admin 不存在 → 创建用户 + 默认 Profile "主人"
//   - admin 已存在但没 Profile → 补建默认 Profile "主人"
//     （历史原因：000001 schema 缺 updated_at 列导致首次启动 Profile 创建失败，
//      留下无 Profile 的孤儿 admin；schema 修好后重启就能补上）
//   - admin 已存在且有 Profile → 直接返回
func (s *AuthService) EnsureDefaultAdmin(ctx context.Context) error {
	const defaultUsername = "admin"
	const defaultPassword = "admin123" // 首次登录后必须修改！

	existing, err := s.users.GetByUsername(ctx, defaultUsername)
	if err == nil && existing != nil {
		// admin 已存在；检查是否需要补 Profile
		if len(existing.Profiles) == 0 {
			return s.users.CreateProfile(ctx, &user.Profile{
				UserID: existing.ID,
				Name:   "主人",
			})
		}
		return nil
	}
	if ae, ok := apperr.As(err); ok && ae.Code != apperr.CodeNotFound {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := &user.User{
		Username:     defaultUsername,
		PasswordHash: string(hash),
		DisplayName:  "默认管理员",
		Role:         "admin",
		IsActive:     true,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return err
	}
	// 默认 Profile
	p := &user.Profile{
		UserID: u.ID,
		Name:   "主人",
	}
	return s.users.CreateProfile(ctx, p)
}

// DefaultWebProfileID 与 Web 播放器 localStorage 约定的默认 Profile
const DefaultWebProfileID = "00000000-0000-0000-0000-000000000001"

// EnsureDefaultWebProfile 确保 Web 播放端默认 Profile 存在（history FK 依赖）
func (s *AuthService) EnsureDefaultWebProfile(ctx context.Context) error {
	if _, err := s.users.GetProfile(ctx, DefaultWebProfileID); err == nil {
		return nil
	} else if ae, ok := apperr.As(err); !ok || ae.Code != apperr.CodeNotFound {
		return err
	}

	admin, err := s.users.GetByUsername(ctx, "admin")
	if err != nil {
		return err
	}

	p := &user.Profile{
		UserID: admin.ID,
		Name:   "默认",
	}
	p.ID = uuid.MustParse(DefaultWebProfileID)
	return s.users.CreateProfile(ctx, p)
}

// Helper: uuid nil 检查
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
