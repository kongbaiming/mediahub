// Package handler 包含所有 HTTP 处理器
package handler

import (
	"github.com/mediahub/api/internal/downloader"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/recommend"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/internal/service"
	"github.com/mediahub/api/internal/subtitle"

	"github.com/gin-gonic/gin"
)

// Handlers 聚合所有 handler 依赖
type Handlers struct {
	Media          *MediaHandler
	Layout         *LayoutHandler
	Auth           *AuthHandler
	Feed           *FeedHandler
	History        *HistoryHandler
	Profile        *ProfileHandler
	Recommend      *RecommendHandler
	Downloader     *DownloaderHandler
	Scanner        *ScannerHandler
	Subtitle       *SubtitleHandler
	Stream         gin.HandlerFunc
	HLSPlaylist    gin.HandlerFunc
	HLSTaskStatus  gin.HandlerFunc
	HLSCacheRoot   string
}

// NewHandlers 构造
func NewHandlers(
	media *service.MediaService,
	layout *service.LayoutService,
	auth *service.AuthService,
	feed *service.FeedService,
	history *service.HistoryService,
	profile *service.ProfileService,
	recommend *recommend.Service,
	dl *downloader.Service,
	scannerSvc *scanner.Service,
	subSvc *subtitle.Service,
	mediaRoot string,
	hlsCacheRoot string,
	transcode HLSTranscodeSettings,
) *Handlers {
	h := &Handlers{
		Media:         NewMediaHandler(media),
		Layout:        NewLayoutHandler(layout, feed),
		Auth:          NewAuthHandler(auth),
		Feed:          NewFeedHandler(feed),
		History:       NewHistoryHandler(history),
		Profile:       NewProfileHandler(profile),
		Recommend:     NewRecommendHandler(recommend),
		Stream:        StreamHandler(mediaRoot, hlsCacheRoot, transcode),
		HLSPlaylist:   ServeHLSPlaylist(hlsCacheRoot),
		HLSTaskStatus: GetHLSTaskStatus(hlsCacheRoot),
		HLSCacheRoot:  hlsCacheRoot,
	}
	if dl != nil {
		h.Downloader = NewDownloaderHandler(dl)
	}
	if scannerSvc != nil {
		h.Scanner = NewScannerHandler(scannerSvc)
	}
	if subSvc != nil {
		h.Subtitle = NewSubtitleHandler(subSvc)
	}
	return h
}

// RegisterRoutes 注册所有路由
func (h *Handlers) RegisterRoutes(r *gin.Engine) {
	// ---------- 健康检查 ----------
	r.GET("/health", HealthCheck)
	r.GET("/healthz", HealthCheck) // alias for k8s-style
	r.GET("/health/ready", ReadinessCheck)
	r.GET("/metrics", MetricsHandler)

	// ---------- API v1 ----------
	v1 := r.Group("/api/v1")
	v1.Use(middleware.InjectProfileID())
	{
		// 认证（无需登录）
		v1.POST("/auth/login", h.Auth.Login)
		v1.POST("/auth/register", h.Auth.Register)
		v1.GET("/auth/me", middleware.Auth(h.Auth.svc), h.Auth.Me)

		// 媒资（部分需要登录）
		v1.GET("/media", h.Media.List)
		v1.GET("/media/stats", h.Media.Stats)
		v1.GET("/media/:id", h.Media.Get)
		v1.POST("/media", middleware.Auth(h.Auth.svc), h.Media.Create)
		v1.PATCH("/media/:id", middleware.Auth(h.Auth.svc), h.Media.Update)
		v1.DELETE("/media/:id", middleware.Auth(h.Auth.svc), middleware.RequireAdmin(), h.Media.Delete)
		v1.POST("/media/:id/rescan", middleware.Auth(h.Auth.svc), h.Media.Rescan)

		// 布局
		v1.GET("/layouts", h.Layout.List)
		v1.GET("/layouts/:id/preview", middleware.Auth(h.Auth.svc), h.Layout.Preview)
		v1.GET("/layouts/:id", h.Layout.Get)
		v1.POST("/layouts", middleware.Auth(h.Auth.svc), h.Layout.Create)
		v1.PATCH("/layouts/:id", middleware.Auth(h.Auth.svc), h.Layout.Update)
		v1.DELETE("/layouts/:id", middleware.Auth(h.Auth.svc), middleware.RequireAdmin(), h.Layout.Delete)
		v1.POST("/layouts/:id/publish", middleware.Auth(h.Auth.svc), h.Layout.Publish)
		v1.GET("/layouts/:id/publications", h.Layout.ListPublications)
		v1.DELETE("/layouts/publications/:pub_id", middleware.Auth(h.Auth.svc), middleware.RequireAdmin(), h.Layout.DisablePublication)

		// 播放端 Feed（无需登录）
		v1.GET("/feed/:platform", h.Feed.Get)

		// 媒资搜索（无需登录）
		v1.GET("/search", h.Media.Search)

		// 推荐（无需登录）
		v1.GET("/recommend/hot", h.Recommend.Hot)
		v1.GET("/recommend/similar/:id", h.Recommend.SimilarTo)

		// 流代理（无需登录）
		// 注意：Gin 不允许 catch-all (*action) 和具体 path segment (hls)
		// 共存于同一父节点下，无论注册顺序都会 panic。所以这里彻底放弃
		// catch-all，把 stream 的两个 action 改成具体路由。
		//   /stream/direct?path=...           直连文件
		//   /stream/hls?path=...&media_id=... 启动 HLS 转码
		//   /stream/hls/:media_id/playlist.m3u8  提供 HLS playlist
		//   /stream/hls/:media_id/:file         提供 ts 切片
		//   /stream/hls/:media_id/status        查询转码状态
		v1.GET("/stream/hls/:media_id/playlist.m3u8", h.HLSPlaylist)
		v1.GET("/stream/hls/:media_id/:file", h.HLSPlaylist)
		v1.GET("/stream/hls/:media_id/status", h.HLSTaskStatus)
		v1.GET("/stream/direct", h.Stream)
		v1.GET("/stream/hls", h.Stream)

		// 下载管理（无需登录，但生产应该限管理员）
		if h.Downloader != nil {
			dl := v1.Group("/downloader")
			{
				dl.POST("/add", h.Downloader.Add)
				dl.GET("/list", h.Downloader.List)
				dl.DELETE("/:hash", h.Downloader.Remove)
				dl.POST("/:hash/pause", h.Downloader.Pause)
				dl.POST("/:hash/resume", h.Downloader.Resume)
				dl.POST("/check-completed", h.Downloader.CheckCompleted)
				dl.GET("/health", h.Downloader.Health)
			}
		}

		// 库扫描
		if h.Scanner != nil {
			v1.POST("/scanner/scan", h.Scanner.Scan)
		}

		// 字幕
		if h.Subtitle != nil {
			v1.POST("/subtitle/search", h.Subtitle.Search)
			v1.POST("/subtitle/:id/download", h.Subtitle.Download)
		}

		// 播放进度 / 续播 / 收藏（播放端：仅需 X-Profile-ID，无需 JWT）
		v1.GET("/playback/default-profile", h.History.DefaultProfile)
		playback := v1.Group("/")
		playback.Use(middleware.RequireProfile())
		{
			playback.POST("/history", h.History.Record)
			playback.GET("/resume/:media_id", h.History.GetResumePoint)
			playback.GET("/continue-watching", h.History.ContinueWatching)
			playback.POST("/favorites", h.History.ToggleFavorite)
			playback.GET("/favorites", h.History.ListFavorites)
		}

		// 历史 / Profile 管理（CMS 需登录）
		authed := v1.Group("/")
		authed.Use(middleware.Auth(h.Auth.svc))
		{
			authed.GET("/history", h.History.List)
			authed.GET("/profiles", h.Profile.List)
			authed.POST("/profiles", h.Profile.Create)
			authed.PATCH("/profiles/:id", h.Profile.Update)
			authed.DELETE("/profiles/:id", h.Profile.Delete)
			authed.POST("/profiles/:id/verify-pin", h.Profile.VerifyPin)

			// 个人推荐（需 Profile ID）
			authed.GET("/recommend/for-me", h.Recommend.ForProfile)
		}
	}

	// ---------- Swagger 文档 ----------
	r.GET("/swagger/*any", SwaggerHandler())
}
