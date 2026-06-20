package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	seasonFolderRe  = regexp.MustCompile(`(?i)^(?:season[\s._-]*)?(\d{1,2})$`)
	numericEpRe     = regexp.MustCompile(`^\d{1,4}$`)
	episodeOnlyRe   = regexp.MustCompile(`(?i)^S?(?P<s>\d{1,2})E(?P<e>\d{1,2})$`)
)

// ParseFilePath 结合文件路径解析（电视剧专辑名优先取文件夹名）
func ParseFilePath(fullPath string) *ParsedFile {
	p := ParseFileName(fullPath)
	parentDir := filepath.Dir(fullPath)
	seriesName := filepath.Base(parentDir)
	baseName := strings.TrimSuffix(filepath.Base(fullPath), filepath.Ext(fullPath))

	// 父目录 Season 01 / S01
	if sn := seasonFromDir(parentDir); sn != nil {
		if p.Season == nil {
			p.Season = sn
		}
		grandDir := filepath.Dir(parentDir)
		if isSeriesAlbumDir(grandDir, seriesName) {
			seriesName = filepath.Base(grandDir)
		}
	}

	// 纯数字文件名 + 专辑文件夹 → 剧集单集
	if numericEpRe.MatchString(strings.TrimSpace(baseName)) && isSeriesAlbumDir(parentDir, seriesName) {
		p.Type = "episode"
		p.Title = cleanTitle(seriesName)
		if ep, err := strconv.Atoi(strings.TrimSpace(baseName)); err == nil {
			p.Episode = &ep
		}
		if p.Season == nil {
			s := 1
			p.Season = &s
		}
		return p
	}

	// 仅 S01E05 文件名 → 从文件夹取专辑名
	if m := episodeOnlyRe.FindStringSubmatch(strings.TrimSpace(baseName)); len(m) == 3 && isSeriesAlbumDir(parentDir, seriesName) {
		p.Type = "episode"
		p.Title = cleanTitle(seriesName)
		if s, err := strconv.Atoi(m[1]); err == nil {
			p.Season = &s
		}
		if e, err := strconv.Atoi(m[2]); err == nil {
			p.Episode = &e
		}
		return p
	}

	// SxxExx 但剧名不可靠（如只有数字）→ 用文件夹名
	if p.Type == "episode" && isWeakSeriesTitle(p.Title) && isSeriesAlbumDir(parentDir, seriesName) {
		p.Title = cleanTitle(seriesName)
	}

	return p
}

func seasonFromDir(dir string) *int {
	base := filepath.Base(dir)
	if m := seasonFolderRe.FindStringSubmatch(base); len(m) > 1 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return &n
		}
	}
	return nil
}

func isWeakSeriesTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return true
	}
	return numericEpRe.MatchString(title)
}

// isSeriesAlbumDir 判断父目录是否可作为剧集专辑名（排除 movies 等分类目录）
func isSeriesAlbumDir(parentDir, folderName string) bool {
	if folderName == "" || folderName == "." {
		return false
	}
	lower := strings.ToLower(parentDir)
	baseLower := strings.ToLower(folderName)

	skipNames := []string{
		"media", "movies", "movie", "films", "film", "downloads", "download",
		"电影", "下载", "video", "videos", "root",
	}
	for _, s := range skipNames {
		if baseLower == s {
			return false
		}
	}

	// 明确剧集目录
	tvHints := []string{"tvshow", "tvshows", "series", "电视剧", "剧集", "tv", "show"}
	for _, h := range tvHints {
		if strings.Contains(lower, h) {
			return true
		}
	}

	// 排除电影库路径下的单文件
	if strings.Contains(lower, "/movies") || strings.Contains(lower, "\\movies") ||
		strings.Contains(lower, "/movie/") || strings.Contains(lower, "电影") {
		return false
	}

	// 默认：/media/专辑名/单集.ext 视为剧集专辑
	return true
}

// IsEpisodeFile 是否为应归入专辑的剧集单集
func IsEpisodeFile(p *ParsedFile, mtype string) bool {
	if p == nil || p.Type != "episode" {
		return false
	}
	return mtype == "tvshow" || mtype == "anime"
}
