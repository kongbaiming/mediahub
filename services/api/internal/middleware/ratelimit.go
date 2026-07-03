package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 基于 IP 的简单令牌桶限流器
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // 每秒允许的请求数
	burst    int           // 突发上限
	cleanup  time.Duration // 清理间隔
	stop     chan struct{}  // 停止信号
}

type visitor struct {
	tokens    float64
	lastToken time.Time
}

// NewRateLimiter 构造
// rate: 每秒请求数, burst: 突发上限
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  5 * time.Minute,
		stop:     make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Close 停止清理 goroutine
func (rl *RateLimiter) Close() {
	close(rl.stop)
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastToken) > rl.cleanup {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:    float64(rl.burst) - 1,
			lastToken: time.Now(),
		}
		return true
	}

	elapsed := time.Since(v.lastToken).Seconds()
	v.tokens += elapsed * float64(rl.rate)
	if v.tokens > float64(rl.burst) {
		v.tokens = float64(rl.burst)
	}
	v.lastToken = time.Now()

	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

// RateLimit 返回 Gin 限流中间件
//
// 默认配置：
//   - API 端点：100 req/s，突发 200
//   - 登录端点：5 req/s，突发 10
func RateLimit(rate, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit",
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// RateLimitAuth 认证端点限流（更严格）
func RateLimitAuth() gin.HandlerFunc {
	return RateLimit(5, 10)
}

// RateLimitAPI 通用 API 限流
func RateLimitAPI() gin.HandlerFunc {
	return RateLimit(100, 200)
}
