package downloader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"

	"github.com/google/uuid"
)

// Service 下载业务
type Service struct {
	client       *Client
	mediaRepo    *repository.MediaRepo
	queue        *queue.Queue
	downloadRoot string
}

// NewService 构造
func NewService(c *Client, m *repository.MediaRepo, q *queue.Queue, downloadRoot string) *Service {
	return &Service{
		client:       c,
		mediaRepo:    m,
		queue:        q,
		downloadRoot: downloadRoot,
	}
}

// AddRequest 添加下载请求
type AddRequest struct {
	URL      string `json:"url" binding:"required"`
	Category string `json:"category"`
	SavePath string `json:"save_path,omitempty"`
}

// Add 添加下载任务
func (s *Service) Add(ctx context.Context, req AddRequest) (string, error) {
	if req.URL == "" {
		return "", apperr.BadRequest("URL 不能为空")
	}
	ck := req.Category
	if ck == "" {
		ck = "movie"
	}

	savePath := req.SavePath
	if savePath == "" {
		savePath = fmt.Sprintf("%s/%s", strings.TrimRight(s.downloadRoot, "/"), ck)
	}

	hash, err := s.client.AddTorrentURL(ctx, req.URL, savePath, ck)
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeInternal, "qBit 添加任务失败")
	}

	logger.Info("下载任务已添加", "hash", hash, "category", ck, "save_path", savePath)
	return hash, nil
}

// List 列出下载任务（按 category）
func (s *Service) List(ctx context.Context, category string) ([]Torrent, error) {
	return s.client.ListTorrents(ctx, category)
}

// Remove 删除任务
func (s *Service) Remove(ctx context.Context, hash string, deleteFiles bool) error {
	if err := s.client.RemoveTorrent(ctx, hash, deleteFiles); err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "删除下载任务失败")
	}
	return nil
}

// Pause / Resume
func (s *Service) Pause(ctx context.Context, hash string) error {
	if err := s.client.PauseTorrent(ctx, hash); err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "暂停任务失败")
	}
	return nil
}

func (s *Service) Resume(ctx context.Context, hash string) error {
	if err := s.client.ResumeTorrent(ctx, hash); err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "恢复任务失败")
	}
	return nil
}

// Health 健康检查
func (s *Service) Health(ctx context.Context) error {
	return s.client.Health(ctx)
}

// CheckCompleted 扫描 qBit 中已完成的任务，自动入库
func (s *Service) CheckCompleted(ctx context.Context) (int, error) {
	imported := 0

	all, err := s.client.ListTorrents(ctx, "")
	if err != nil {
		return 0, err
	}

	for _, t := range all {
		if t.State != string(StatusCompleted) {
			continue
		}

		existing, _ := s.mediaRepo.GetByStoragePath(ctx, t.SavePath+"/"+t.Name)
		if existing != nil {
			continue
		}

		m := &media.Media{
			Type:         inferTypeFromCategory(t.Category),
			Title:        cleanTitle(t.Name),
			StoragePath:  t.SavePath + "/" + t.Name,
			FileSize:     t.Size,
			ScrapeStatus: common.ScrapeStatusPending,
		}

		if err := s.mediaRepo.Create(ctx, m); err != nil {
			logger.Warn("入库失败", "name", t.Name, "err", err)
			continue
		}

		if s.queue != nil {
			_ = s.queue.EnqueueScrape(ctx, m.ID.String())
		}

		imported++
		logger.Info("媒资自动入库", "id", m.ID, "title", m.Title)
	}

	return imported, nil
}

// StartWatcher 启动定期检查
func (s *Service) StartWatcher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	logger.Info("下载监听器启动", "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := s.CheckCompleted(ctx)
			if err != nil {
				logger.Warn("检查下载任务失败", "err", err)
				continue
			}
			if n > 0 {
				logger.Info("自动入库完成", "imported", n)
			}
		}
	}
}

// inferTypeFromCategory 根据 category 推断类型
func inferTypeFromCategory(cat string) common.MediaType {
	switch strings.ToLower(cat) {
	case "tvshow", "tv":
		return common.MediaTypeTVShow
	case "anime":
		return common.MediaTypeAnime
	case "doc", "documentary":
		return common.MediaTypeDocumentary
	default:
		return common.MediaTypeMovie
	}
}

// cleanTitle 清理文件名
func cleanTitle(name string) string {
	for _, ext := range []string{".mkv", ".mp4", ".avi", ".ts"} {
		name = strings.TrimSuffix(name, ext)
	}
	for _, tag := range []string{"-GROUP", "-RARBG", "-YIFY"} {
		if idx := strings.Index(name, tag); idx > 0 {
			name = name[:idx]
		}
	}
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")
	return strings.TrimSpace(name)
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
