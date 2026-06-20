package handler

import (
	"github.com/mediahub/api/internal/apperr"

	"github.com/gin-gonic/gin"
)

// respondError 统一错误响应
func respondError(c *gin.Context, err error) {
	if ae, ok := apperr.As(err); ok {
		c.AbortWithStatusJSON(ae.HTTPStatus(), gin.H{
			"error":   ae.Code,
			"message": ae.Message,
			"detail":  ae.Detail,
		})
		return
	}
	c.AbortWithStatusJSON(500, gin.H{
		"error":   apperr.CodeInternal,
		"message": "服务内部异常",
		"detail":  err.Error(),
	})
}
