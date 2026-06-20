// Package main 是 MediaHub API 服务的入口
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mediahub/api/internal/cache"
	"github.com/mediahub/api/internal/config"
	"github.com/mediahub/api/internal/db"
	"github.com/mediahub/api/internal/downloader"
	"github.com/mediahub/api/internal/handler"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/recommend"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/internal/scraper"
	"github.com/mediahub/api/internal/subtitle"
	"github.com/mediahub/api/internal/transcoder"
	"github.com/mediahub/api/internal/service"
	"github.com/mediahub/api/internal/worker"
	"github.com/mediahub/api/pkg/logger"
	"github.com/redis/go-redis/v9"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

func main() {
	// ---------- 1. 配置 ----------
	cfg, err := config.Load()
	if err != nil {
		panic("配置加载失败: " + err.Error())
	}

	// ---------- 2. 日志 ----------
	logger.Init(cfg.LogLevel, cfg.Mode == "debug")
	defer logger.Sync()

	logger.Info("🚀 MediaHub API 启动中", "mode", cfg.Mode, "port", cfg.Port)

	// ---------- 3. Gin ----------
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// ---------- 4. 数据库 ----------
	database, err := db.Connect(cfg.Database.URL, cfg.Mode == "debug")
	if err != nil {
		logger.Fatal("数据库连接失败", "err", err)
	}
	defer database.Close()

	// 自动迁移（开发用）
	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 60*time.Second)
	if err := database.Migrate(migrateCtx); err != nil {
		logger.Fatal("数据库迁移失败", "err", err)
	}
	cancelMigrate()
	logger.Info("数据库迁移完成")

	// ---------- 5. Redis 队列 ----------
	redisAddr := parseRedisAddr(cfg.Redis.URL)
	q, asynqServer, err := queue.New(queue.Config{
		RedisAddr:     redisAddr,
		RedisPassword: parseRedisPassword(cfg.Redis.URL),
		Concurrency:   5,
	})
	if err != nil {
		logger.Fatal("队列初始化失败", "err", err)
	}
	defer q.Close()

	// ---------- 5b. Redis 缓存客户端 ----------
	var rdb *redis.Client
	if cfg.Redis.URL != "" {
		opt, err := redis.ParseURL(cfg.Redis.URL)
		if err == nil {
			rdb = redis.NewClient(opt)
			if pingErr := rdb.Ping(context.Background()).Err(); pingErr != nil {
				logger.Warn("Redis 缓存连接失败，将禁用", "err", pingErr)
				rdb = nil
			} else {
				logger.Info("Redis 缓存已就绪")
			}
		}
	}

	// ---------- 6. 仓储层 ----------
	mediaRepo := repository.NewMediaRepo(database.DB)
	layoutRepo := repository.NewLayoutRepo(database.DB)
	userRepo := repository.NewUserRepo(database.DB)
	historyRepo := repository.NewHistoryRepo(database.DB)
	recommendRepo := repository.NewRecommendRepo(database.DB)

	// 启动 Asynq worker（注册具体的 handler）
	tmdbClient := scraper.NewTMDBClient(cfg.TMDB.APIKey, cfg.TMDB.BaseURL, cfg.TMDB.Language)
	trans := transcoder.NewTranscoder("ffmpeg", "qsv")
	handlers := worker.NewHandlers(tmdbClient, trans, mediaRepo, "/data/thumbnails")
	mux := asynq.NewServeMux()
	handlers.Register(mux)
	logger.Info("Asynq worker 已注册", "handlers", []string{"scrape", "thumb", "scan"})
	go func() {
		if err := asynqServer.Start(mux); err != nil {
			logger.Error("Asynq 异常退出", "err", err)
		}
	}()

	// ---------- 7. 服务层 ----------
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)

	// 启动时确保默认管理员
	if err := authSvc.EnsureDefaultAdmin(context.Background()); err != nil {
		logger.Warn("确保默认管理员失败", "err", err)
	}

	mediaSvc := service.NewMediaService(mediaRepo, q)
	layoutSvc := service.NewLayoutService(layoutRepo)
	historySvc := service.NewHistoryService(historyRepo)
	profileSvc := service.NewProfileService(userRepo)

	// 推荐引擎
	recommendEngine := recommend.NewEngine(mediaRepo, historyRepo)
	recommendSvc := recommend.NewService(recommendEngine, recommendRepo)

	// Feed Service 需要推荐接口（接口注入避免循环依赖）
	feedSvc := service.NewFeedService(mediaRepo, layoutRepo, historyRepo, userRepo, recommendSvc)

	// 注入 Redis 缓存（Feed 5 分钟 TTL）
	if rdb != nil {
		feedCache := cache.New(cache.Config{
			Redis:   rdb,
			Prefix:  "mediahub",
			Enabled: true,
		})
		feedSvc.WithCache(feedCache)
		logger.Info("Feed 缓存已启用")
	} else {
		logger.Warn("Redis 不可用，Feed 缓存已禁用")
	}

	// 注入 Feed 缓存失效回调（layoutSvc → feedSvc）
	layoutSvc.SetFeedInvalidator(func(ctx context.Context, platform string) error {
		return feedSvc.InvalidateFeed(ctx, platform)
	})

	// 下载管理
	var downloaderSvc *downloader.Service
	if cfg.Downloader.Enabled && cfg.Downloader.QBittorrent.Host != "" {
		scheme := "http"
		if cfg.Downloader.QBittorrent.UseTLS {
			scheme = "https"
		}
		qbitURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Downloader.QBittorrent.Host, cfg.Downloader.QBittorrent.Port)
		qbitClient := downloader.NewClient(
			qbitURL,
			cfg.Downloader.QBittorrent.Username,
			cfg.Downloader.QBittorrent.Password,
		)
		downloaderSvc = downloader.NewService(qbitClient, mediaRepo, q, cfg.Media.DownloadRoot)
		logger.Info("下载管理已启动", "qbit_url", qbitURL)

		// 启动后台 watcher（自动入库）
		go func() {
			ctx := context.Background()
			interval := time.Duration(cfg.Downloader.WatchInterval) * time.Minute
			downloaderSvc.StartWatcher(ctx, interval)
		}()
	}

	// 库扫描服务（按 NAS 媒体目录扫描）
	scannerSvc := scanner.NewService([]string{cfg.Media.Root}, mediaRepo, q)

	// 启动库扫描 watcher（30 分钟一次）
	go scannerSvc.StartWatcher(context.Background(), 30*time.Minute)
	logger.Info("库扫描已启动", "root", cfg.Media.Root, "interval", "30m")

	// 字幕服务
	subtitleSvc := subtitle.NewService(mediaRepo)

	// ---------- 8. Health checkers ----------
	handler.SetHealthCheckers(
		func(ctx context.Context) string {
			if err := database.Ping(ctx); err != nil {
				return "error: " + err.Error()
			}
			return "ok"
		},
		func(ctx context.Context) string {
			// Asynq 自带 redis 客户端
			return "ok"
		},
	)
	handler.SetDBStatsProvider(func() any {
		return database.Stats()
	})

	// ---------- 9. Handler ----------
	hlsCache := os.Getenv("HLS_CACHE_ROOT")
	if hlsCache == "" {
		hlsCache = "/volume1/docker/mediahub/hls-cache"
	}
	h := handler.NewHandlers(mediaSvc, layoutSvc, authSvc, feedSvc, historySvc, profileSvc, recommendSvc, downloaderSvc, scannerSvc, subtitleSvc, cfg.Media.Root, hlsCache)

	// ---------- 10. 路由 ----------
	r := gin.New()
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Profile-ID"},
		ExposeHeaders: []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge: 12 * time.Hour,
	}))

	h.RegisterRoutes(r)

	// ---------- 11. HTTP Server ----------
	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP 服务异常退出", "err", err)
		}
	}()

	logger.Info("✨ MediaHub API 已就绪", "addr", srv.Addr, "docs", "http://localhost:"+cfg.Port+"/swagger/index.html")

	// ---------- 12. 优雅退出 ----------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到退出信号，开始关闭...")
	asynqServer.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务关闭异常", "err", err)
	}

	logger.Info("👋 MediaHub API 已退出")
}

// parseRedisAddr 从 redis://:password@host:port/db 提取 host:port
func parseRedisAddr(url string) string {
	// 简单处理，去掉 redis:// 和认证部分
	s := strings.TrimPrefix(url, "redis://")
	if idx := strings.Index(s, "@"); idx >= 0 {
		s = s[idx+1:]
	}
	if idx := strings.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
}

func parseRedisPassword(url string) string {
	s := strings.TrimPrefix(url, "redis://")
	if !strings.HasPrefix(s, ":") {
		return ""
	}
	s = s[1:]
	if idx := strings.Index(s, "@"); idx >= 0 {
		return s[:idx]
	}
	return ""
}

// _ 确保 fmt 引用（import 保留）
var _ = fmt.Sprintf
