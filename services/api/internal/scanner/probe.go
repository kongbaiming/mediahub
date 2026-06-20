package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ProbeResult ffprobe 探测结果
type ProbeResult struct {
	Format    ProbeFormat     `json:"format"`
	Streams   []ProbeStream   `json:"streams"`
	Chapters  []ProbeChapter  `json:"chapters,omitempty"`
}

// ProbeFormat 容器信息
type ProbeFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// ProbeStream 流信息（视频/音频/字幕）
type ProbeStream struct {
	Index        int               `json:"index"`
	CodecType    string            `json:"codec_type"` // video | audio | subtitle
	CodecName    string            `json:"codec_name"`
	CodecLongName string           `json:"codec_long_name,omitempty"`
	Width        int               `json:"width,omitempty"`
	Height       int               `json:"height,omitempty"`
	Duration     string            `json:"duration,omitempty"`
	BitRate      string            `json:"bit_rate,omitempty"`
	Channels     int               `json:"channels,omitempty"`
	SampleRate   string            `json:"sample_rate,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// ProbeChapter 章节
type ProbeChapter struct {
	ID        int               `json:"id"`
	TimeBase  string            `json:"time_base"`
	Start     int               `json:"start"`
	End       int               `json:"end"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// Probe 调 ffprobe 探测文件
func Probe(ctx context.Context, ffmpegBin, filePath string) (*ProbeResult, error) {
	if ffmpegBin == "" {
		ffmpegBin = "ffprobe"
	}
	cmd := exec.CommandContext(ctx,
		ffmpegBin,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_chapters",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 执行失败: %w", err)
	}

	var result ProbeResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("ffprobe 输出解析失败: %w", err)
	}
	return &result, nil
}

// MediaInfo 抽取的核心媒体信息
type MediaInfo struct {
	Duration    int    // 秒
	VideoCodec  string
	AudioCodec  string
	Resolution  string
	HasSubtitle bool
	BitRate     int64
}

// Extract 从 ProbeResult 抽取关键信息
func (p *ProbeResult) Extract() MediaInfo {
	info := MediaInfo{}
	if d, err := strconv.ParseFloat(p.Format.Duration, 64); err == nil {
		info.Duration = int(d)
	}
	if br, err := strconv.ParseInt(p.Format.BitRate, 10, 64); err == nil {
		info.BitRate = br
	}
	for _, s := range p.Streams {
		switch s.CodecType {
		case "video":
			info.VideoCodec = strings.ToLower(s.CodecName)
			res := fmt.Sprintf("%dx%d", s.Width, s.Height)
			info.Resolution = res
			// 标准分辨率归一化
			switch s.Height {
			case 2160:
				info.Resolution = "4K"
			case 1080:
				info.Resolution = "1080p"
			case 720:
				info.Resolution = "720p"
			}
		case "audio":
			info.AudioCodec = strings.ToLower(s.CodecName)
		case "subtitle":
			info.HasSubtitle = true
		}
	}
	return info
}
