package scanner

import (
	"context"
	"strings"
	"time"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/queue"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"
)

// Service 库扫描业务
type Service struct {
	roots      []string
	scanner    *Scanner
	mediaRepo  *repository.MediaRepo
	queue      *queue.Queue
}

// NewService 构造
func NewService(roots []string, mediaRepo *repository.MediaRepo, q *queue.Queue) *Service {
	return &Service{
		roots:     roots,
		mediaRepo: mediaRepo,
		queue:     q,
	}
}

// ScanResult 扫描结果
type ScanResult struct {
	Total   int `json:"total"`
	Added   int `json:"added"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

// ScanAll 全量扫描
func (s *Service) ScanAll(ctx context.Context) (*ScanResult, error) {
	if len(s.roots) == 0 {
		logger.Warn("未配置扫描根目录")
		return &ScanResult{}, nil
	}

	result := &ScanResult{}

	for _, root := range s.roots {
		count, err := s.scanRoot(ctx, root, result)
		if err != nil {
			logger.Warn("扫描目录失败", "root", root, "err", err)
			continue
		}
		result.Total += count
	}

	logger.Info("扫描完成",
		"roots", s.roots,
		"total", result.Total,
		"added", result.Added,
		"skipped", result.Skipped,
		"failed", result.Failed,
	)

	return result, nil
}

// ScanRoot 扫描单个根目录
func (s *Service) ScanRoot(ctx context.Context, root string) (*ScanResult, error) {
	result := &ScanResult{}
	if _, err := s.scanRoot(ctx, root, result); err != nil {
		return result, err
	}
	return result, nil
}

// scanRoot 内部递归扫描
func (s *Service) scanRoot(ctx context.Context, root string, result *ScanResult) (int, error) {
	count := 0
	sc, err := NewScanner([]string{root})
	if err != nil {
		return 0, err
	}
	defer sc.Stop()

	// 收集所有媒体文件路径
	var paths []string
	sc.OnChange(func(ev FileEvent) {
		if !IsMediaFile(ev.Path) {
			return
		}
		paths = append(paths, ev.Path)
	})

	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	sc.Start(scanCtx)

	// 处理扫描到的文件
	deps := IngestDeps{MediaRepo: s.mediaRepo, Queue: s.queue}
	for _, p := range paths {
		count++
		result.Total++

		ingestRes, err := IngestMediaFile(ctx, deps, p)
		if err != nil {
			logger.Warn("入库失败", "path", p, "err", err)
			result.Failed++
			continue
		}
		if ingestRes == nil {
			continue
		}
		if ingestRes.Added {
			result.Added++
		}
		if ingestRes.Skipped {
			result.Skipped++
		}
	}

	return count, nil
}

// StartWatcher 启动定期扫描
func (s *Service) StartWatcher(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	logger.Info("库扫描监听器启动", "interval", interval, "roots", s.roots)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 启动时立即扫一次
	if _, err := s.ScanAll(ctx); err != nil {
		logger.Warn("初始扫描失败", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.ScanAll(ctx); err != nil {
				logger.Warn("定期扫描失败", "err", err)
			}
		}
	}
}

// inferTypeFromDir 根据目录推断媒体类型（与 parser.go 重复，但 service 需要）
func inferTypeFromDir(p *ParsedFile, parentDir string) common.MediaType {
	lower := strings.ToLower(parentDir)
	if strings.Contains(lower, "documentar") || strings.Contains(lower, "纪录") {
		return common.MediaTypeDocumentary
	}
	if strings.Contains(lower, "anime") || strings.Contains(lower, "动画") {
		return common.MediaTypeAnime
	}
	if p.Type == "episode" {
		return common.MediaTypeTVShow
	}
	return common.MediaTypeMovie
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func buildTags(p *ParsedFile) []string {
	tags := []string{}
	if p.Resolution != "" {
		tags = append(tags, p.Resolution)
	}
	if p.Source != "" {
		tags = append(tags, p.Source)
	}
	if p.Group != "" {
		tags = append(tags, p.Group)
	}
	if p.VideoCodec != "" {
		tags = append(tags, p.VideoCodec)
	}
	return tags
}
