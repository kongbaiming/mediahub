package middleware

import (
	"strings"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// CtxKey 上下文 key
const (
	CtxKeyUserID   = "user_id"
	CtxKeyUsername = "username"
	CtxKeyRole     = "role"
	CtxProfileID   = "profile_id"
)

// Auth JWT 鉴权中间件
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			respondError(c, apperr.Unauthorized("缺少 Authorization 头"))
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			respondError(c, apperr.Unauthorized("Authorization 格式错误"))
			return
		}

		claims, err := authSvc.ParseToken(parts[1])
		if err != nil {
			respondError(c, apperr.Unauthorized("Token 无效或已过期"))
			return
		}

		// 注入上下文
		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyUsername, claims.Username)
		c.Set(CtxKeyRole, claims.Role)

		// 可选 Profile ID（请求头 X-Profile-ID）
		if pid := c.GetHeader("X-Profile-ID"); pid != "" {
			c.Set(CtxProfileID, pid)
		}

		c.Next()
	}
}

// RequireAdmin 管理员权限
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxKeyRole)
		if role != "admin" {
			respondError(c, apperr.Forbidden("需要管理员权限"))
			return
		}
		c.Next()
	}
}

// respondError 统一错误响应
func respondError(c *gin.Context, err *apperr.AppError) {
	c.AbortWithStatusJSON(err.HTTPStatus(), gin.H{
		"error":   err.Code,
		"message": err.Message,
		"detail":  err.Detail,
	})
}

// GetUserID 从上下文取 user_id
func GetUserID(c *gin.Context) string {
	v, _ := c.Get(CtxKeyUserID)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetProfileID 从上下文取 profile_id
func GetProfileID(c *gin.Context) string {
	v, _ := c.Get(CtxProfileID)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
