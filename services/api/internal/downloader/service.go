package downloader

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/pkg/logger"
)

// Service 下载业务
type Service struct {
	client       *Client
	mediaRepo    *repository.MediaRepo
	catalogRepo  *repository.CatalogRepo
	queue        *queue.Queue
	downloadRoot string
}

// NewService 构造
func NewService(c *Client, m *repository.MediaRepo, catalog *repository.CatalogRepo, q *queue.Queue, downloadRoot string) *Service {
	return &Service{
		client:       c,
		mediaRepo:    m,
		catalogRepo:  catalog,
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
	sourceURL, err := NormalizeDownloadURL(req.URL)
	if err != nil {
		return "", err
	}
	ck := req.Category
	if ck == "" {
		ck = "movie"
	}

	savePath := req.SavePath
	if savePath == "" {
		savePath = fmt.Sprintf("%s/%s", strings.TrimRight(s.downloadRoot, "/"), ck)
	}

	hash, err := s.client.AddTorrentURL(ctx, sourceURL, savePath, ck)
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeInternal, "qBit 添加任务失败")
	}

	logger.Info("下载任务已添加", "hash", hash, "category", ck, "save_path", savePath, "source", maskURL(sourceURL))
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

// CheckCompleted 扫描 qBit 中已完成的任务（progress=100%），走 scanner 入库
func (s *Service) CheckCompleted(ctx context.Context) (int, error) {
	imported := 0

	all, err := s.client.ListTorrents(ctx, "")
	if err != nil {
		return 0, err
	}

	deps := scanner.IngestDeps{
		MediaRepo: s.mediaRepo,
		Catalog:   s.catalogRepo,
		Queue:     s.queue,
	}

	for _, t := range all {
		if !isTorrentCompleted(t) {
			continue
		}

		torrentRoot := filepath.Join(t.SavePath, t.Name)
		if !torrentHasPlayableMedia(torrentRoot) {
			continue
		}

		paths, err := collectTorrentMediaPaths(torrentRoot)
		if err != nil {
			logger.Warn("收集种子媒体文件失败", "name", t.Name, "err", err)
			continue
		}

		added, _ := ingestTorrentPaths(ctx, deps, paths)
		if added > 0 {
			imported += added
			logger.Info("下载完成自动入库", "torrent", t.Name, "added", added, "hash", t.Hash)
		}
	}

	return imported, nil
}

// StartWatcher 启动定期检查（下载 100% 后入库）
func (s *Service) StartWatcher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	logger.Info("下载监听器启动", "interval", interval)

	// 启动后立即检查一次
	if n, err := s.CheckCompleted(ctx); err != nil {
		logger.Warn("初始下载入库检查失败", "err", err)
	} else if n > 0 {
		logger.Info("初始下载入库完成", "imported", n)
	}

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

// isTorrentCompleted qBit 完成态：progress=100% 且不在下载中
func isTorrentCompleted(t Torrent) bool {
	if t.Progress < 1.0 {
		return false
	}
	return !isActiveDownloadState(t.State)
}

func isActiveDownloadState(state string) bool {
	switch strings.ToLower(state) {
	case "downloading", "stalleddl", "metadl", "forceddl", "checkingdl", "queueddl", "allocating":
		return true
	default:
		return false
	}
}

func maskURL(u string) string {
	if len(u) <= 80 {
		return u
	}
	return u[:40] + "..." + u[len(u)-20:]
}
