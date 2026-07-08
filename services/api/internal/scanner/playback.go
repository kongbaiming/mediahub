package scanner

import (
	"fmt"
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

// StreamTrack 单条音视频/字幕流（ffprobe streams）
type StreamTrack struct {
	Index     int    `json:"index"`
	Codec     string `json:"codec"`
	Language  string `json:"language,omitempty"`
	Label     string `json:"label,omitempty"`
	Channels  int    `json:"channels,omitempty"`
	Title     string `json:"title,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

// PlaybackHint 播放策略（供 stream probe 与前端决策）
type PlaybackHint struct {
	VideoCodec            string        `json:"video_codec"`
	AudioCodec            string        `json:"audio_codec"`
	Width                 int           `json:"width"`
	Height                int           `json:"height"`
	Resolution            string        `json:"resolution"`
	Container             string        `json:"container"`
	DirectPlayable        bool          `json:"direct_playable"`
	HLSCopyable           bool          `json:"hls_copyable"`
	Recommended           PlaybackMode  `json:"recommended"`
	AudioTracks           []StreamTrack `json:"audio_tracks,omitempty"`
	EmbeddedSubtitleTracks []StreamTrack `json:"embedded_subtitle_tracks,omitempty"`
	DefaultAudioIndex     int           `json:"default_audio_index,omitempty"`
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

	audioN := 0
	defaultAudio := -1
	for _, s := range p.Streams {
		switch s.CodecType {
		case "audio":
			track := streamTrackFromProbe(s, audioN)
			h.AudioTracks = append(h.AudioTracks, track)
			if track.IsDefault {
				defaultAudio = track.Index
			}
			audioN++
		case "subtitle":
			h.EmbeddedSubtitleTracks = append(h.EmbeddedSubtitleTracks, streamTrackFromProbe(s, len(h.EmbeddedSubtitleTracks)))
		}
	}
	if defaultAudio >= 0 {
		h.DefaultAudioIndex = defaultAudio
	} else if len(h.AudioTracks) > 0 {
		h.DefaultAudioIndex = h.AudioTracks[0].Index
	}
	return h
}

func streamTrackFromProbe(s ProbeStream, ordinal int) StreamTrack {
	lang := ""
	title := ""
	isDefault := false
	if s.Tags != nil {
		lang = s.Tags["language"]
		title = s.Tags["title"]
		if s.Tags["default"] == "1" {
			isDefault = true
		}
	}
	label := title
	if label == "" && lang != "" {
		label = strings.ToUpper(lang)
	}
	if label == "" {
		label = fmt.Sprintf("轨道 %d", ordinal+1)
	}
	return StreamTrack{
		Index:     s.Index,
		Codec:     s.CodecName,
		Language:  lang,
		Label:     label,
		Channels:  s.Channels,
		Title:     title,
		IsDefault: isDefault,
	}
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
