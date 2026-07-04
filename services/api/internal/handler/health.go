package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck 存活检查
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "mediahub-api",
		"time":    time.Now().Format(time.RFC3339),
		"version": "0.1.0",
	})
}

// ReadinessCheck 就绪检查（带 context 超时 + 性能指标）
func ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{
		"database": checkDatabase(ctx),
		"redis":    checkRedis(ctx),
	}
	if tmdbChecker != nil {
		checks["tmdb"] = tmdbChecker(ctx)
	}
	if qbitChecker != nil {
		checks["qbittorrent"] = qbitChecker(ctx)
	}
	if aiChecker != nil {
		checks["ai"] = aiChecker(ctx)
	}

	allOk := true
	for _, v := range checks {
		if s, ok := v.(string); !ok || s != "ok" {
			allOk = false
			break
		}
	}

	status := http.StatusOK
	if !allOk {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": "ready",
		"checks": checks,
		"time":   time.Now().Format(time.RFC3339),
	})
}

// MetricsHandler 性能指标（Prometheus 格式输出，但不是 Prometheus）
//
// 用于运维人员直接 curl 看：
//   curl http://nas:3000/metrics
func MetricsHandler(c *gin.Context) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	c.JSON(http.StatusOK, gin.H{
		"runtime": gin.H{
			"goroutines": runtime.NumGoroutine(),
			"heap_alloc_mb": float64(mem.HeapAlloc) / 1024 / 1024,
			"heap_sys_mb":   float64(mem.HeapSys) / 1024 / 1024,
			"gc_runs":       mem.NumGC,
			"go_version":    runtime.Version(),
		},
		"database": getDBStats(),
		"time":     time.Now().Format(time.RFC3339),
	})
}

// checkDatabase / checkRedis 由 main.go 注入实现
var (
	dbChecker      func(ctx context.Context) string
	redisChecker   func(ctx context.Context) string
	dbStats        func() any
	tmdbChecker    func(ctx context.Context) string
	qbitChecker    func(ctx context.Context) string
	aiChecker      func(ctx context.Context) string
)

// SetHealthCheckers 设置健康检查器
func SetHealthCheckers(db, redis func(ctx context.Context) string) {
	dbChecker = db
	redisChecker = redis
}

// SetExtraHealthCheckers 设置额外的健康检查器
func SetExtraHealthCheckers(tmdb, qbit, ai func(ctx context.Context) string) {
	tmdbChecker = tmdb
	qbitChecker = qbit
	aiChecker = ai
}

// SetDBStatsProvider 设置 DB 连接池统计 provider
func SetDBStatsProvider(fn func() any) {
	dbStats = fn
}

func checkDatabase(ctx context.Context) string {
	if dbChecker != nil {
		return dbChecker(ctx)
	}
	return "ok"
}

func checkRedis(ctx context.Context) string {
	if redisChecker != nil {
		return redisChecker(ctx)
	}
	return "ok"
}

func getDBStats() any {
	if dbStats != nil {
		return dbStats()
	}
	return nil
}