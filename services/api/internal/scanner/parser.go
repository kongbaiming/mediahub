package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParsedFile 解析后的文件信息
type ParsedFile struct {
	Type          string // movie | episode
	Title         string
	Year          *int
	Season        *int
	Episode       *int
	Resolution    string // 1080p | 4K | 720p
	VideoCodec    string // x264 | x265 | HEVC
	AudioCodec    string // AAC | DTS | TrueHD
	Source        string // BluRay | WEB-DL | HDTV
	Group         string // 字幕组 / 发布组
	Container     string // mkv | mp4
	OriginalName  string
}

// 正则模式（Sonarr 风格命名规范）
var (
	// 电影: Title (2024).mkv  / Title.2024.1080p.BluRay.x264-GROUP.mkv
	movieRe = regexp.MustCompile(`^(?P<title>.+?)[. _\-\(]+(?P<year>(19|20)\d{2})`)
	// 剧集: Title - S01E02.mkv  / Title.S01E02.1080p.mkv  / Show - 1x02.mkv
	episodeRe = regexp.MustCompile(`(?i)(?P<title>.+?)[. _\-\(]+S(?P<s>\d{1,2})E(?P<e>\d{1,2})`)
	// 备用剧集格式: Title 1x02
	episodeAltRe = regexp.MustCompile(`(?P<title>.+?)[. _\-\(]+(?P<s>\d{1,2})x(?P<e>\d{1,2})`)
	// 分辨率
	resolutionRe = regexp.MustCompile(`(?i)(2160p|4k|1080p|720p|480p)`)
	// 视频编码
	videoCodecRe = regexp.MustCompile(`(?i)(x264|h264|x265|h265|hevc|avc|10bit)`)
	// 音频编码
	audioCodecRe = regexp.MustCompile(`(?i)(aac|dts[-. ]?hd|dts|truehd|atmos|ddp|dd5\.1|dd2\.0|opus|flac)`)
	// 来源
	sourceRe = regexp.MustCompile(`(?i)(bluray|blu-ray|web[-.]?dl|webrip|hdtv|pdtv|dvdrip|hdrip)`)
	// 字幕组
	groupRe = regexp.MustCompile(`-([A-Za-z0-9]+)$`)
)

// ParseFileName 解析文件名
func ParseFileName(path string) *ParsedFile {
	base := filepath.Base(path)
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(base), "."))

	p := &ParsedFile{
		Container:    ext,
		OriginalName: base,
	}

	name := strings.TrimSuffix(base, filepath.Ext(base))

	// 1. 尝试剧集
	if m := episodeRe.FindStringSubmatch(name); m != nil {
		p.Type = "episode"
		p.Title = cleanTitle(m[1])
		if s, err := strconv.Atoi(m[2]); err == nil {
			p.Season = &s
		}
		if e, err := strconv.Atoi(m[3]); err == nil {
			p.Episode = &e
		}
	} else if m := episodeAltRe.FindStringSubmatch(name); m != nil {
		p.Type = "episode"
		p.Title = cleanTitle(m[1])
		if s, err := strconv.Atoi(m[2]); err == nil {
			p.Season = &s
		}
		if e, err := strconv.Atoi(m[3]); err == nil {
			p.Episode = &e
		}
	} else if m := movieRe.FindStringSubmatch(name); m != nil {
		// 2. 电影
		p.Type = "movie"
		p.Title = cleanTitle(m[1])
		if y, err := strconv.Atoi(m[2]); err == nil {
			p.Year = &y
		}
	} else {
		// 3. 兜底：当作电影
		p.Type = "movie"
		p.Title = cleanTitle(name)
	}

	// 3. 其他标签
	if m := resolutionRe.FindStringSubmatch(name); m != nil {
		p.Resolution = strings.ToLower(m[1])
	}
	if m := videoCodecRe.FindStringSubmatch(name); m != nil {
		p.VideoCodec = strings.ToUpper(strings.TrimRight(strings.ToLower(m[1]), "10bit"))
	}
	if m := audioCodecRe.FindStringSubmatch(name); m != nil {
		p.AudioCodec = strings.ToUpper(m[1])
	}
	if m := sourceRe.FindStringSubmatch(name); m != nil {
		p.Source = m[1]
	}
	if m := groupRe.FindStringSubmatch(base); m != nil {
		p.Group = m[1]
	}

	return p
}

func cleanTitle(s string) string {
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	// 移除多余空格
	return strings.TrimSpace(joinNonEmpty(strings.Fields(s)))
}

func joinNonEmpty(ss []string) string {
	out := ""
	for _, s := range ss {
		if s == "" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += s
	}
	return out
}

// InferMediaType 从 ParsedFile 推断 MediaType
// 优先看父目录关键词（中文/英文），其次看 p.Type
func InferMediaType(p *ParsedFile, parentDir string) string {
	lower := strings.ToLower(parentDir)

	// 优先级：documentary > anime > 类型推断
	// 用前缀 "documentar" 匹配 documentary / documentaries
	hasDocumentary := strings.Contains(lower, "documentar") || strings.Contains(lower, "纪录")
	hasAnime := strings.Contains(lower, "anime") || strings.Contains(lower, "动画")

	if hasDocumentary {
		return "documentary"
	}
	if hasAnime {
		return "anime"
	}

	// 默认根据 p.Type 推断
	if p.Type == "episode" {
		return "tvshow"
	}
	return "movie"
}
