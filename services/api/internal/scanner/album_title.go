package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	bracketTagRe     = regexp.MustCompile(`\[[^\]]*\]`)
	leadingHanTitle  = regexp.MustCompile(`^[\p{Han}]+`)
	seriesNoSuffixRe = regexp.MustCompile(`^\d+[\p{Han}]+`)
	yearInParensRe   = regexp.MustCompile(`[\(（]\s*(19|20)\d{2}\s*[\)）]\.?$`)
)

// IsVideoFileName 是否为视频文件名（含扩展名）
func IsVideoFileName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mkv", ".mp4", ".m4v", ".avi", ".mov", ".webm", ".ts":
		return true
	default:
		return false
	}
}

// AlbumFolderName 从 storage_path 提取专辑文件夹名（剧集 album 为目录本身，电影为父目录）
func AlbumFolderName(storagePath string) string {
	base := filepath.Base(storagePath)
	if IsVideoFileName(base) {
		return filepath.Base(filepath.Dir(storagePath))
	}
	return base
}

// SeriesFolderTitle 从专辑文件夹名提取 TMDB 搜索用剧名（去掉 Emby 画质/发布组后缀）
func SeriesFolderTitle(folderName string) string {
	return cleanSeriesAlbumTitle(folderName)
}

// SeriesFolderYear 从文件夹名括号中提取年份（如 「西行(2024)」→ 2024）
func SeriesFolderYear(folderName string) *int {
	m := yearInParensRe.FindStringSubmatch(strings.TrimSpace(folderName))
	if len(m) < 2 {
		return nil
	}
	full := m[0]
	digits := regexp.MustCompile(`(19|20)\d{2}`).FindString(full)
	if digits == "" {
		return nil
	}
	y, err := strconv.Atoi(digits)
	if err != nil || y <= 0 {
		return nil
	}
	return &y
}

func normalizeSeriesFolderName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".")
	name = strings.ReplaceAll(name, "丨", "")
	name = strings.ReplaceAll(name, "·", "")
	name = strings.ReplaceAll(name, "•", "")
	name = yearInParensRe.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

func cleanSeriesAlbumTitle(name string) string {
	name = normalizeSeriesFolderName(name)
	if name == "" {
		return ""
	}

	name = bracketTagRe.ReplaceAllString(name, "")
	name = strings.TrimSpace(name)

	// Emby 前缀：T唐朝诡事录0509
	if strings.HasPrefix(name, "T") {
		if r, _ := utf8.DecodeRuneInString(strings.TrimPrefix(name, "T")); unicode.Is(unicode.Han, r) {
			name = strings.TrimPrefix(name, "T")
		}
	}

	if m := leadingHanTitle.FindString(name); m != "" {
		rest := strings.TrimPrefix(name, m)
		if len(rest) > 0 && len(rest) <= 2 && isAllDigits(rest) {
			return m + rest
		}
		if sub := seriesNoSuffixRe.FindString(rest); sub != "" {
			return m + sub
		}
		return m
	}

	// 英文文件夹：Calabash.Brothers.II.1991... → Calabash Brothers II
	if i := strings.Index(name, "."); i > 0 {
		return cleanTitle(name[:i])
	}

	return cleanTitle(name)
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func albumSeriesTitle(seriesName string) string {
	if t := cleanSeriesAlbumTitle(seriesName); t != "" {
		return t
	}
	return cleanTitle(seriesName)
}
