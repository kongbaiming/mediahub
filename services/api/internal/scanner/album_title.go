package scanner

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	bracketTagRe    = regexp.MustCompile(`\[[^\]]*\]`)
	leadingHanTitle = regexp.MustCompile(`^[\p{Han}丨]+`)
	seriesNoSuffixRe = regexp.MustCompile(`^\d+[\p{Han}]+`)
)

// SeriesFolderTitle 从专辑文件夹名提取 TMDB 搜索用剧名（去掉 Emby 画质/发布组后缀）
func SeriesFolderTitle(folderName string) string {
	return cleanSeriesAlbumTitle(folderName)
}

func cleanSeriesAlbumTitle(name string) string {
	name = strings.TrimSpace(name)
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
