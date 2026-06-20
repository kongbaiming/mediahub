package middleware

import (
	"github.com/mediahub/api/internal/apperr"

	"github.com/gin-gonic/gin"
)

// InjectProfileID 从 X-Profile-ID 注入上下文（播放端 Feed / 续播等，无需 JWT）
func InjectProfileID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if pid := c.GetHeader("X-Profile-ID"); pid != "" {
			c.Set(CtxProfileID, pid)
		}
		c.Next()
	}
}

// RequireProfile 要求请求头携带 X-Profile-ID（家庭播放端按 Profile 区分进度）
func RequireProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		pid := c.GetHeader("X-Profile-ID")
		if pid == "" {
			respondError(c, apperr.BadRequest("缺少 X-Profile-ID 请求头"))
			return
		}
		c.Set(CtxProfileID, pid)
		c.Next()
	}
}
