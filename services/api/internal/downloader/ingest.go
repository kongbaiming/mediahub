package downloader

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mediahub/api/internal/mediafile"
	"github.com/mediahub/api/internal/scanner"
	"github.com/mediahub/api/pkg/logger"
)

// collectTorrentMediaPaths 收集种子目录/文件下的可入库媒体路径
func collectTorrentMediaPaths(torrentRoot string) ([]string, error) {
	info, err := os.Stat(torrentRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if scanner.IsMediaFile(torrentRoot) {
			return []string{torrentRoot}, nil
		}
		return nil, nil
	}

	var paths []string
	err = filepath.WalkDir(torrentRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if scanner.IsMediaFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// torrentHasPlayableMedia 完成态种子下至少有一个可播放文件（避免未完成下载误入库）
func torrentHasPlayableMedia(torrentRoot string) bool {
	paths, err := collectTorrentMediaPaths(torrentRoot)
	if err != nil || len(paths) == 0 {
		return false
	}
	for _, p := range paths {
		if ok, _ := mediafile.IsPlayable(p); ok {
			return true
		}
	}
	return false
}

// ingestTorrentPaths 将种子内媒体文件走 scanner 入库管线
func ingestTorrentPaths(ctx context.Context, deps scanner.IngestDeps, paths []string) (added, skipped int) {
	for _, p := range paths {
		res, err := scanner.IngestMediaFile(ctx, deps, p)
		if err != nil {
			logger.Warn("下载入库失败", "path", p, "err", err)
			continue
		}
		if res == nil {
			continue
		}
		if res.Added {
			added++
		}
		if res.Skipped {
			skipped++
		}
	}
	return added, skipped
}
