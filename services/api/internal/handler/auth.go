package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证 HTTP handler
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler 构造
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, resp)
}

// Register 注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	resp, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, resp)
}

// Me 当前用户
func (h *AuthHandler) Me(c *gin.Context) {
	uid := middleware.GetUserID(c)
	user, err := h.svc.Users().GetByID(c.Request.Context(), uid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{
		"data":     user,
		"user_id":  uid,
		"username": user.Username,
		"role":     user.Role,
	})
}
