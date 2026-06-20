package subtitle

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/repository"

	"github.com/mediahub/api/pkg/logger"
)

// Service 字幕业务
type Service struct {
	mediaRepo *repository.MediaRepo
}

// NewService 构造
func NewService(m *repository.MediaRepo) *Service {
	return &Service{mediaRepo: m}
}

// SearchRequest 搜索请求
type SearchRequest struct {
	MediaID string `json:"media_id" binding:"required"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	Lang    string `json:"lang"`
}

// Search 搜索字幕
func (s *Service) Search(ctx context.Context, req SearchRequest) ([]Subtitle, error) {
	m, err := s.mediaRepo.GetByID(ctx, req.MediaID)
	if err != nil {
		return nil, err
	}

	subs, err := MatchSubtitles(ctx, m, req.Season, req.Episode)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "搜索字幕失败")
	}
	logger.Info("字幕搜索", "media", m.Title, "found", len(subs))
	return subs, nil
}

// DownloadRequest 下载请求
type DownloadRequest struct {
	Subtitle Subtitle `json:"subtitle" binding:"required"`
}

// Download 下载字幕到媒资目录
func (s *Service) Download(ctx context.Context, mediaID string, sub Subtitle) error {
	m, err := s.mediaRepo.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}

	// 字幕保存到媒资同目录
	mediaDir := filepath.Dir(m.StoragePath)
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "创建目录失败")
	}

	// 实际下载（使用 SubHD 或 OpenSubtitles）
	client := NewSubHDClient()
	if err := client.Download(ctx, sub, mediaDir); err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "下载字幕失败")
	}

	logger.Info("字幕已下载", "media", m.Title, "lang", sub.Language)
	return nil
}
