// Package transcoder 提供视频转码能力（FFmpeg 封装）
//
// DS920+ (J4125) 优化：
//   - 优先使用 Intel Quick Sync 硬转（h264_qsv / hevc_qsv）
//   - HLS 切片用 HLS 协议（m3u8 + ts）
//   - 缩略图用 JPEG（多张）
package transcoder

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Transcoder 转码器
type Transcoder struct {
	ffmpegBin string
	hwAccel   string // qsv | nvenc | none
}

// NewTranscoder 构造
func NewTranscoder(ffmpegBin, hwAccel string) *Transcoder {
	if ffmpegBin == "" {
		ffmpegBin = "ffmpeg"
	}
	if hwAccel == "" {
		hwAccel = "qsv"
	}
	return &Transcoder{ffmpegBin: ffmpegBin, hwAccel: hwAccel}
}

// ThumbnailOptions 缩略图生成选项
type ThumbnailOptions struct {
	Input       string
	OutputDir   string
	Count       int    // 生成张数
	Width       int    // 缩略图宽度
	SeekPercent []int // 自定义时间点（秒），为空则均分
}

// GenerateThumbnails 生成多张缩略图
func (t *Transcoder) GenerateThumbnails(ctx context.Context, opts ThumbnailOptions) ([]string, error) {
	if opts.Count <= 0 {
		opts.Count = 3
	}
	if opts.Width <= 0 {
		opts.Width = 640
	}

	// 1. 先获取时长
	duration, err := t.probeDuration(ctx, opts.Input)
	if err != nil {
		return nil, err
	}

	// 2. 计算时间点
	seekPoints := opts.SeekPercent
	if len(seekPoints) == 0 {
		seekPoints = make([]int, opts.Count)
		step := duration / (opts.Count + 1)
		for i := 0; i < opts.Count; i++ {
			seekPoints[i] = step * (i + 1)
		}
	}

	// 3. 生成每张缩略图
	results := make([]string, 0, opts.Count)
	for i, sec := range seekPoints {
		out := filepath.Join(opts.OutputDir, fmt.Sprintf("thumb_%d.jpg", i))
		args := []string{
			"-ss", strconv.Itoa(sec),
			"-i", opts.Input,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale=%d:-1", opts.Width),
			"-q:v", "3",
			"-y",
			out,
		}
		cmd := exec.CommandContext(ctx, t.ffmpegBin, args...)
		if out2, err := cmd.CombinedOutput(); err != nil {
			return results, fmt.Errorf("缩略图 %d 生成失败: %w: %s", i, err, string(out2))
		}
		results = append(results, out)
	}
	return results, nil
}

// HLSOptions HLS 转码选项
type HLSOptions struct {
	Input        string
	OutputDir    string
	Width        int
	Height       int
	Bitrate      string // 视频码率，如 4000k
	AudioBitrate string // 音频码率
	SegmentTime  int    // 切片秒数
	Preset       string // libx264 preset
	CopyAudio    bool   // 源音频已是 AAC 时直接 copy
	CopyVideo    bool   // 视频流 copy（原分辨率/码率，适合内网 4K）
}

// TranscodeHLS 转码为 HLS
func (t *Transcoder) TranscodeHLS(ctx context.Context, opts HLSOptions) (string, error) {
	return t.transcodeHLS(ctx, opts, t.hwAccel)
}

// TranscodeHLSWithFallback 流复制失败时回退转码；硬转失败时回退 libx264
func (t *Transcoder) TranscodeHLSWithFallback(ctx context.Context, opts HLSOptions) (string, error) {
	if opts.CopyVideo {
		copyOpts := opts
		playlist, err := t.transcodeHLS(ctx, copyOpts, "none")
		if err == nil {
			return playlist, nil
		}
		opts.CopyVideo = false
		if opts.Height <= 0 && opts.Width <= 0 {
			opts.Height = 1080
		}
	}

	playlist, err := t.transcodeHLS(ctx, opts, t.hwAccel)
	if err == nil {
		return playlist, nil
	}
	if t.hwAccel == "none" {
		return "", err
	}
	soft := NewTranscoder(t.ffmpegBin, "none")
	playlist, softErr := soft.transcodeHLS(ctx, opts, "none")
	if softErr != nil {
		return "", fmt.Errorf("硬转(%s)失败: %v; 软转失败: %w", t.hwAccel, err, softErr)
	}
	return playlist, nil
}

func (t *Transcoder) transcodeHLS(ctx context.Context, opts HLSOptions, hwAccel string) (string, error) {
	if opts.Bitrate == "" {
		opts.Bitrate = "2500k"
	}
	if opts.AudioBitrate == "" {
		opts.AudioBitrate = "128k"
	}
	if opts.SegmentTime <= 0 {
		opts.SegmentTime = 4
	}
	if opts.Preset == "" {
		opts.Preset = "ultrafast"
	}
	if !opts.CopyVideo && opts.Height <= 0 && opts.Width <= 0 {
		opts.Height = 480
	}

	// 源音频已是 AAC 时可 copy，跳过重编码
	if !opts.CopyAudio {
		opts.CopyAudio = t.probeAudioIsAAC(ctx, opts.Input)
	}

	playlist := filepath.Join(opts.OutputDir, "playlist.m3u8")

	tr := *t
	tr.hwAccel = hwAccel
	args := tr.buildHLSArgs(opts, playlist)

	cmd := exec.CommandContext(ctx, t.ffmpegBin, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("HLS 转码失败(%s): %w: %s", hwAccel, err, trimFFmpegLog(string(out)))
	}
	return playlist, nil
}

func (t *Transcoder) buildHLSArgs(opts HLSOptions, playlist string) []string {
	if opts.Bitrate == "" {
		opts.Bitrate = "2500k"
	}
	if opts.AudioBitrate == "" {
		opts.AudioBitrate = "128k"
	}
	if opts.SegmentTime <= 0 {
		opts.SegmentTime = 4
	}
	if opts.Preset == "" {
		opts.Preset = "ultrafast"
	}

	args := []string{"-i", opts.Input}

	if opts.CopyVideo {
		args = append(args, "-c:v", "copy")
		if opts.CopyAudio {
			args = append(args, "-c:a", "copy")
		} else {
			args = append(args, "-c:a", "aac", "-b:a", opts.AudioBitrate)
		}
	} else {
		if t.hwAccel == "qsv" {
			args = append(args, "-hwaccel", "qsv")
		}
		args = append(args, "-c:v", t.videoEncoder())
		if t.hwAccel == "none" {
			args = append(args, "-preset", opts.Preset)
		}
		args = append(args, "-b:v", opts.Bitrate)
		if opts.CopyAudio {
			args = append(args, "-c:a", "copy")
		} else {
			args = append(args, "-c:a", "aac", "-b:a", opts.AudioBitrate)
		}
		args = append(args, "-vf", t.scaleFilter(opts.Width, opts.Height))
	}

	args = append(args,
		"-hls_time", strconv.Itoa(opts.SegmentTime),
		"-hls_list_size", "0",
		"-hls_flags", "independent_segments+temp_file",
		"-hls_segment_filename", filepath.Join(opts.OutputDir, "seg_%03d.ts"),
		"-max_muxing_queue_size", "1024",
		"-f", "hls",
		"-y",
		playlist,
	)
	return args
}

func trimFFmpegLog(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 800 {
		return s[len(s)-800:]
	}
	return s
}

// videoEncoder 选编码器
func (t *Transcoder) videoEncoder() string {
	switch t.hwAccel {
	case "qsv":
		return "h264_qsv"
	case "nvenc":
		return "h264_nvenc"
	default:
		return "libx264"
	}
}

func (t *Transcoder) scaleFilter(w, h int) string {
	if w > 0 && h > 0 {
		return fmt.Sprintf("scale=%d:%d", w, h)
	}
	if w > 0 {
		return fmt.Sprintf("scale=%d:-2", w)
	}
	if h > 0 {
		return fmt.Sprintf("scale=-2:%d", h)
	}
	return "scale=-2:480"
}

// probeAudioIsAAC 检测首个音频流是否为 AAC（可 copy 跳过重编码）
func (t *Transcoder) probeAudioIsAAC(ctx context.Context, file string) bool {
	ffprobe := strings.Replace(t.ffmpegBin, "ffmpeg", "ffprobe", 1)
	args := []string{
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=nw=1",
		file,
	}
	cmd := exec.CommandContext(ctx, ffprobe, args...)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	codec := strings.ToLower(strings.TrimSpace(string(out)))
	return strings.Contains(codec, "aac") || strings.Contains(codec, "mp4a")
}

func (t *Transcoder) probeDuration(ctx context.Context, file string) (int, error) {
	args := []string{"-i", file}
	cmd := exec.CommandContext(ctx, t.ffmpegBin, args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0, fmt.Errorf("无法探测时长")
	}
	// ffmpeg 在 stderr 输出 Duration 信息（用 CombinedOutput 模拟）
	outStr := string(out)
	idx := strings.Index(outStr, "Duration: ")
	if idx < 0 {
		return 0, fmt.Errorf("未找到时长信息")
	}
	end := strings.Index(outStr[idx:], ",")
	if end < 0 {
		return 0, fmt.Errorf("时长格式异常")
	}
	durStr := outStr[idx+10 : idx+end]
	parts := strings.Split(durStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("时长格式异常: %s", durStr)
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.ParseFloat(parts[2], 64)
	return h*3600 + m*60 + int(s), nil
}
