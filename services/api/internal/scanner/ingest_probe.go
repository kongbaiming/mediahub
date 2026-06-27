package scanner

import (
	"context"
	"time"

	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/pkg/logger"
)

// probeIngestFileAsync 扫描入库后异步 ffprobe，写入 media_files（v0.4 B4）
func probeIngestFileAsync(repo *repository.MediaRepo, filePath string) {
	if repo == nil || filePath == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		result, err := Probe(ctx, "", filePath)
		if err != nil {
			logger.Debug("入库 ffprobe 失败", "path", filePath, "err", err)
			return
		}
		mi := result.Extract()
		w, h := result.VideoSize()
		if err := repo.ApplyProbeToFile(ctx, filePath, repository.FileProbeInfo{
			Duration:    mi.Duration,
			VideoCodec:  mi.VideoCodec,
			AudioCodec:  mi.AudioCodec,
			Resolution:  mi.Resolution,
			HasSubtitle: mi.HasSubtitle,
			BitRate:     mi.BitRate,
		}, w, h); err != nil {
			logger.Debug("写入 probe 失败", "path", filePath, "err", err)
		}
	}()
}
