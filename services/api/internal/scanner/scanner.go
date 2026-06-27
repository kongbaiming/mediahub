// Package scanner 提供媒资文件扫描与识别能力
//
// 核心能力：
//   - 监听目录新增/删除文件
//   - 解析文件名识别电影 / 剧集 / 季 / 集
//   - 调 ffprobe 获取视频元数据
package scanner

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mediahub/api/internal/mediafile"
	"github.com/mediahub/api/pkg/logger"
)

// Scanner 文件扫描器
type Scanner struct {
	roots    []string
	watcher  *fsnotify.Watcher
	onChange func(event FileEvent)
	stopCh   chan struct{}
}

// FileEvent 文件事件
type FileEvent struct {
	Type   string // created | modified | deleted | renamed
	Path   string
	Parsed *ParsedFile
}

// NewScanner 构造
func NewScanner(roots []string) (*Scanner, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	s := &Scanner{
		roots:   roots,
		watcher: w,
		stopCh:  make(chan struct{}),
	}
	for _, r := range roots {
		if err := w.Add(r); err != nil {
			logger.Warn("添加监听失败", "path", r, "err", err)
			continue
		}
		logger.Info("开始监听目录", "path", r)
	}
	return s, nil
}

// OnChange 设置回调
func (s *Scanner) OnChange(fn func(event FileEvent)) {
	s.onChange = fn
}

// Start 启动监听（阻塞直到 ctx 取消）
func (s *Scanner) Start(ctx context.Context) {
	defer s.watcher.Close()

	// 启动时全量扫描一次
	go s.FullScan(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case ev, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.handleEvent(ev)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			logger.Warn("watcher 错误", "err", err)
		}
	}
}

// Stop 停止
func (s *Scanner) Stop() {
	close(s.stopCh)
}

// FullScan 全量扫描
func (s *Scanner) FullScan(ctx context.Context) {
	logger.Info("开始全量扫描", "roots", s.roots)
	count := 0
	for _, root := range s.roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if info.IsDir() {
				// 监听子目录
				if path != root {
					_ = s.watcher.Add(path)
				}
				return nil
			}
			if !IsMediaFile(path) {
				return nil
			}
			parsed := ParseFileName(path)
			count++
			if s.onChange != nil {
				s.onChange(FileEvent{
					Type:   "scanned",
					Path:   path,
					Parsed: parsed,
				})
			}
			return nil
		})
		if err != nil {
			logger.Warn("扫描失败", "root", root, "err", err)
		}
	}
	logger.Info("全量扫描完成", "count", count)
}

func (s *Scanner) handleEvent(ev fsnotify.Event) {
	if !IsMediaFile(ev.Name) {
		return
	}
	var eventType string
	switch {
	case ev.Op&fsnotify.Create == fsnotify.Create:
		eventType = "created"
	case ev.Op&fsnotify.Write == fsnotify.Write:
		eventType = "modified"
	case ev.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = "deleted"
	case ev.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = "renamed"
	default:
		return
	}

	parsed := ParseFileName(ev.Name)
	if s.onChange != nil {
		s.onChange(FileEvent{
			Type:   eventType,
			Path:   ev.Name,
			Parsed: parsed,
		})
	}
}

// ---- 文件识别 ----

// mediaExtensions 常见视频扩展
var mediaExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".m4v": true, ".avi": true, ".mov": true,
	".wmv": true, ".flv": true, ".webm": true, ".ts": true,
	".m2ts": true, ".iso": true, ".bdmv": true,
	".rmvb": true, ".rm": true,
}

// Emby/Jellyfin：「吃面.mp4 5678」等 basename 含 .mp4 但 Ext() 无法识别
var embeddedVideoExt = regexp.MustCompile(`(?i)\.(mkv|mp4|m4v|avi|mov|wmv|flv|webm|ts|m2ts|rmvb|rm)(?:[\s.]|$)`)

// IsMediaFile 是否是视频文件
func IsMediaFile(path string) bool {
	if mediafile.ShouldSkipScan(path) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	if mediaExtensions[ext] {
		return true
	}
	// Emby/Jellyfin 元数据后缀：如「吃面.mp4 5678」导致 Ext() 无法识别
	return embeddedVideoExt.MatchString(filepath.Base(path))
}

// IsDirectoryEmpty 目录是否为空
func IsDirectoryEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// NowMillis 当前毫秒时间戳
func NowMillis() int64 {
	return time.Now().UnixMilli()
}
