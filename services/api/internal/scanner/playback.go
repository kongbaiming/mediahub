package scanner

import (
	"path/filepath"
	"strings"
)

// PlaybackMode 推荐播放方式
type PlaybackMode string

const (
	PlaybackDirect        PlaybackMode = "direct"
	PlaybackHLSCopy       PlaybackMode = "hls_copy"
	PlaybackHLSTranscode  PlaybackMode = "hls_transcode"
)

// PlaybackHint 播放策略（供 stream probe 与前端决策）
type PlaybackHint struct {
	VideoCodec     string       `json:"video_codec"`
	AudioCodec     string       `json:"audio_codec"`
	Width          int          `json:"width"`
	Height         int          `json:"height"`
	Resolution     string       `json:"resolution"`
	Container      string       `json:"container"`
	DirectPlayable bool         `json:"direct_playable"`
	HLSCopyable    bool         `json:"hls_copyable"`
	Recommended    PlaybackMode `json:"recommended"`
}

// PlaybackHint 从探测结果推断浏览器/HLS 播放策略
func (p *ProbeResult) PlaybackHint(filePath string) PlaybackHint {
	info := p.Extract()
	h := PlaybackHint{
		VideoCodec: info.VideoCodec,
		AudioCodec: info.AudioCodec,
		Width:      0,
		Height:     0,
		Resolution: info.Resolution,
		Container:  p.Format.FormatName,
	}

	for _, s := range p.Streams {
		if s.CodecType == "video" && s.Height > 0 {
			h.Width = s.Width
			h.Height = s.Height
			break
		}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	h.DirectPlayable = isDirectPlayable(ext, h.VideoCodec)
	h.HLSCopyable = isHLSCopyable(h.VideoCodec)

	switch {
	case h.DirectPlayable:
		h.Recommended = PlaybackDirect
	case h.HLSCopyable:
		h.Recommended = PlaybackHLSCopy
	default:
		h.Recommended = PlaybackHLSTranscode
	}
	return h
}

func isDirectPlayable(ext, videoCodec string) bool {
	switch ext {
	case ".mp4", ".m4v", ".mov":
		return videoCodec == "h264" || videoCodec == "hevc" || videoCodec == "h265"
	case ".webm":
		return videoCodec == "vp8" || videoCodec == "vp9" || videoCodec == "av1"
	default:
		return false
	}
}

func isHLSCopyable(videoCodec string) bool {
	switch videoCodec {
	case "h264", "hevc", "h265", "mpeg4", "mpeg2video":
		return true
	default:
		return false
	}
}
