// Package middleware 提供 Gin 中间件
package middleware

import (
	"time"

	"github.com/mediahub/api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Recovery panic 恢复 + 错误日志
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"err", err,
				)
				c.AbortWithStatusJSON(500, gin.H{
					"error":   "internal_server_error",
					"message": "服务内部异常",
				})
			}
		}()
		c.Next()
	}
}

// RequestLogger 请求日志
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"ip", c.ClientIP(),
			"size", c.Writer.Size(),
		}

		switch {
		case status >= 500:
			logger.Error("HTTP", fields...)
		case status >= 400:
			logger.Warn("HTTP", fields...)
		default:
			logger.Info("HTTP", fields...)
		}
	}
}
