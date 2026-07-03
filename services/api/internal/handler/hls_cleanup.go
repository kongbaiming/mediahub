package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mediahub/api/pkg/logger"
)

// StartHLSCacheCleanup 启动 HLS 缓存定期清理
//
// 每小时检查一次，删除超过 maxAge 的 HLS 缓存目录。
// 目录结构：hlsCacheRoot/<media_id>/playlist.m3u8 + *.ts + .profile.json
func StartHLSCacheCleanup(ctx context.Context, hlsCacheRoot string, maxAge time.Duration) {
	if hlsCacheRoot == "" {
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// 启动时先清理一次
	cleanExpired(ctx, hlsCacheRoot, maxAge)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanExpired(ctx, hlsCacheRoot, maxAge)
		}
	}
}

func cleanExpired(ctx context.Context, root string, maxAge time.Duration) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Warn("HLS 缓存清理：读取目录失败", "path", root, "err", err)
		return
	}

	now := time.Now()
	removed := 0
	freedBytes := int64(0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		dirPath := filepath.Join(root, entry.Name())

		// 检查目录内最新的文件修改时间
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 用目录本身的修改时间判断
		if now.Sub(info.ModTime()) < maxAge {
			continue
		}

		// 计算目录大小
		size := dirSize(dirPath)

		// 删除过期目录
		if err := os.RemoveAll(dirPath); err != nil {
			logger.Warn("HLS 缓存清理：删除失败", "path", dirPath, "err", err)
			continue
		}

		removed++
		freedBytes += size
	}

	if removed > 0 {
		logger.Info("HLS 缓存清理完成",
			"removed", removed,
			"freed", formatBytes(freedBytes),
			"max_age", maxAge.String(),
		)
	}
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
