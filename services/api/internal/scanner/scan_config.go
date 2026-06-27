package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mediahub/api/internal/domain/settings"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"
)

// ScanConfigView 对外展示的扫描配置
type ScanConfigView struct {
	Enabled         bool       `json:"enabled"`
	IntervalMinutes int        `json:"interval_minutes"`
	Roots           []string   `json:"roots"`
	LastScanAt      *time.Time `json:"last_scan_at,omitempty"`
	LastScanStatus  string     `json:"last_scan_status,omitempty"`
	LastScanMessage string     `json:"last_scan_message,omitempty"`
	LastScanAdded   int        `json:"last_scan_added"`
	LastScanTotal   int        `json:"last_scan_total"`
}

// UpdateScanConfigRequest 更新扫描配置
type UpdateScanConfigRequest struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"interval_minutes"`
}

var scanMu sync.Mutex

func (s *Service) configRepo() *repository.MediaScanConfigRepo {
	return s.scanConfigRepo
}

func (s *Service) GetScanConfig(ctx context.Context) (*ScanConfigView, error) {
	repo := s.configRepo()
	if repo == nil {
		return &ScanConfigView{Enabled: true, IntervalMinutes: 30, Roots: s.roots}, nil
	}
	cfg, err := repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	return toScanConfigView(cfg, s.roots), nil
}

func (s *Service) UpdateScanConfig(ctx context.Context, req UpdateScanConfigRequest) (*ScanConfigView, error) {
	repo := s.configRepo()
	if repo == nil {
		return nil, fmt.Errorf("扫描配置未初始化")
	}
	cfg, err := repo.Update(ctx, req.Enabled, req.IntervalMinutes)
	if err != nil {
		return nil, err
	}
	return toScanConfigView(cfg, s.roots), nil
}

func toScanConfigView(cfg *settings.MediaScanConfig, roots []string) *ScanConfigView {
	return &ScanConfigView{
		Enabled:         cfg.Enabled,
		IntervalMinutes: cfg.IntervalMinutes,
		Roots:           roots,
		LastScanAt:      cfg.LastScanAt,
		LastScanStatus:  string(cfg.LastScanStatus),
		LastScanMessage: cfg.LastScanMessage,
		LastScanAdded:   cfg.LastScanAdded,
		LastScanTotal:   cfg.LastScanTotal,
	}
}

func (s *Service) recordScanResult(ctx context.Context, result *ScanResult, err error) {
	repo := s.configRepo()
	if repo == nil {
		return
	}
	status := settings.ScanStatusOK
	msg := "扫描完成"
	added, total := 0, 0
	if result != nil {
		added = result.Added
		total = result.Total
		msg = fmt.Sprintf("扫描 %d 个文件，新增 %d，跳过 %d，失败 %d", result.Total, result.Added, result.Skipped, result.Failed)
	}
	if err != nil {
		status = settings.ScanStatusFailed
		msg = err.Error()
	}
	_ = repo.UpdateScanResult(ctx, status, msg, added, total)
}

func (s *Service) RunScan(ctx context.Context, root string) (*ScanResult, error) {
	if !scanMu.TryLock() {
		return nil, fmt.Errorf("扫描正在进行中")
	}
	defer scanMu.Unlock()

	var result *ScanResult
	var err error
	if root != "" {
		result, err = s.ScanRoot(ctx, root)
	} else {
		result, err = s.ScanAll(ctx)
	}
	s.recordScanResult(ctx, result, err)
	return result, err
}

func (s *Service) runScan(ctx context.Context) (*ScanResult, error) {
	return s.RunScan(ctx, "")
}

// StartWatcher 启动定期扫描（每分钟检查配置）
func (s *Service) StartWatcher(ctx context.Context) {
	logger.Info("库扫描监听器启动", "roots", s.roots)

	if n, err := s.RemigrateMisplacedMovies(ctx); err != nil {
		logger.Warn("剧集单集迁移失败", "err", err)
	} else if n > 0 {
		logger.Info("启动时已迁移误入库剧集单集", "count", n)
	}

	if repo := s.configRepo(); repo != nil {
		cfg, err := repo.Get(ctx)
		if err == nil && cfg.Enabled {
			if _, err := s.runScan(ctx); err != nil && err.Error() != "扫描正在进行中" {
				logger.Warn("初始扫描失败", "err", err)
			}
		}
	} else if _, err := s.runScan(ctx); err != nil && err.Error() != "扫描正在进行中" {
		logger.Warn("初始扫描失败", "err", err)
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickScheduledScan(ctx)
		}
	}
}

func (s *Service) tickScheduledScan(ctx context.Context) {
	repo := s.configRepo()
	if repo == nil {
		return
	}
	due, _, err := repo.IsDue(ctx, time.Now())
	if err != nil || !due {
		return
	}
	if _, err := s.runScan(ctx); err != nil {
		if err.Error() != "扫描正在进行中" {
			logger.Warn("定期扫描失败", "err", err)
		}
	}
}
