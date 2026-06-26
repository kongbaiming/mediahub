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
	episodeInNameRe = regexp.MustCompile(`(?i)(?:^|[.\s_-])S(?P<s>\d{1,2})E(?P<e>\d{1,2})(?:[.\s_-]|$)`)
	episodeMarkerRe = regexp.MustCompile(`(?i)(?:^|[._-])E(?P<e>\d{1,3})(?:[._-]|$)`)
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

	// 纯数字文件名 + 专辑文件夹 → 剧集单集（电影续集文件夹除外，如 冰河世纪4/053.mp4）
	if numericEpRe.MatchString(strings.TrimSpace(baseName)) && isSeriesAlbumDir(parentDir, seriesName) {
		if isMovieAlbumFolder(parentDir, seriesName) {
			p.Type = "movie"
			p.Title = MovieFolderTitle(seriesName)
			if p.Title == "" {
				p.Title = albumSeriesTitle(seriesName)
			}
			refineMovieTitleFromPath(fullPath, p)
			return p
		}
		p.Type = "episode"
		p.Title = albumSeriesTitle(seriesName)
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
		p.Title = albumSeriesTitle(seriesName)
		if s, err := strconv.Atoi(m[1]); err == nil {
			p.Season = &s
		}
		if e, err := strconv.Atoi(m[2]); err == nil {
			p.Episode = &e
		}
		return p
	}

	// S02E01.4K / Detective.Chinatown.S02E01... 等 Emby 命名
	if m := episodeInNameRe.FindStringSubmatch(baseName); len(m) == 3 && isSeriesAlbumDir(parentDir, seriesName) {
		p.Type = "episode"
		p.Title = albumSeriesTitle(seriesName)
		if s, err := strconv.Atoi(m[1]); err == nil {
			p.Season = &s
		}
		if e, err := strconv.Atoi(m[2]); err == nil {
			p.Episode = &e
		}
		return p
	}

	// 葫芦小金刚...E04... 等 E 集数命名
	if m := episodeMarkerRe.FindStringSubmatch(baseName); len(m) == 2 && isSeriesAlbumDir(parentDir, seriesName) {
		p.Type = "episode"
		p.Title = albumSeriesTitle(seriesName)
		if ep, err := strconv.Atoi(m[1]); err == nil && ep > 0 {
			p.Episode = &ep
		}
		if p.Season == nil {
			s := 1
			p.Season = &s
		}
		return p
	}

	// 小品集/综艺等：同文件夹内多视频合并为一部剧集专辑
	if isCollectionAlbumDir(seriesName) && isSeriesAlbumDir(parentDir, seriesName) {
		p.Type = "episode"
		p.Title = albumSeriesTitle(seriesName)
		if p.Season == nil {
			s := 1
			p.Season = &s
		}
		return p
	}

	// SxxExx 但剧名不可靠（如只有数字）→ 用文件夹名
	if p.Type == "episode" && isSeriesAlbumDir(parentDir, seriesName) {
		p.Title = albumSeriesTitle(seriesName)
	} else if p.Type == "episode" && isWeakSeriesTitle(p.Title) && isSeriesAlbumDir(parentDir, seriesName) {
		p.Title = albumSeriesTitle(seriesName)
	}

	if p.Type == "movie" {
		refineMovieTitleFromPath(fullPath, p)
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

// isCollectionAlbumDir 小品集、综艺等同文件夹多视频应合并为一部专辑
func isCollectionAlbumDir(folderName string) bool {
	hints := []string{"小品集", "综艺", "选集", "专场", "片集", "相声", "脱口秀"}
	for _, h := range hints {
		if strings.Contains(folderName, h) {
			return true
		}
	}
	return false
}

// collectionEpisodeTitle 从 Emby 风格文件名提取单集标题（如「吃面.mp4 5678」→「吃面」）
func collectionEpisodeTitle(filePath string) string {
	base := filepath.Base(filePath)
	lower := strings.ToLower(base)
	for _, ext := range []string{".mp4", ".mkv", ".m4v", ".avi", ".mov", ".webm", ".ts"} {
		if i := strings.Index(lower, ext); i > 0 {
			return strings.TrimSpace(base[:i])
		}
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return cleanTitle(name)
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

	// 默认：/media/专辑名/单集.ext 视为剧集专辑（分类目录名已在 skipNames 排除）
	return true
}

var movieSequelFolderRe = regexp.MustCompile(`^[\p{Han}]+(?:之[\p{Han}]+)*\d+$`)

// isMovieAlbumFolder 纯数字文件名时，判断文件夹是否更像电影（如 冰河世纪4/053.mp4）而非剧集
func isMovieAlbumFolder(parentDir, folderName string) bool {
	if isMovieLibraryPath(parentDir) {
		return true
	}
	title := MovieFolderTitle(folderName)
	if title == "" {
		title = albumSeriesTitle(folderName)
	}
	if movieSequelFolderRe.MatchString(title) {
		return true
	}
	return englishMovieSequelRe.MatchString(title)
}

func isMovieLibraryPath(dir string) bool {
	hints := []string{"movies", "movie", "films", "film", "电影"}
	for _, seg := range strings.Split(filepath.ToSlash(dir), "/") {
		lower := strings.ToLower(strings.TrimSpace(seg))
		for _, h := range hints {
			if lower == h {
				return true
			}
		}
	}
	return false
}

var englishMovieSequelRe = regexp.MustCompile(`(?i)^[a-z0-9][a-z0-9\s:.'-]*\d+$`)

// IsEpisodeFile 是否为应归入专辑的剧集单集
func IsEpisodeFile(p *ParsedFile, mtype string) bool {
	if p == nil || p.Type != "episode" {
		return false
	}
	return mtype == "tvshow" || mtype == "anime"
}
