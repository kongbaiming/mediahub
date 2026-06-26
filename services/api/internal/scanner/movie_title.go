package scanner

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	qualityTokenRe = regexp.MustCompile(`(?i)(4k|3d|2160p|1080p|720p|480p|60fps|120fps|\d+fps|\d+帧|出屏|国配|双语|原声|国语|粤语|台语|国粤台|web-dl|bluray|x264|x265|h265|hevc|10bit|aac|dts|hdr|sbs|half|加长版|白星|修复版|收藏版|合集)`)
	englishTitleRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9.'\s:-]{1,}`)
)

// MovieFolderTitle 从电影文件夹名提取 TMDB 搜索用标题
func MovieFolderTitle(folderName string) string {
	return cleanMovieFolderTitle(folderName)
}

func cleanMovieFolderTitle(name string) string {
	name = normalizeSeriesFolderName(name)
	if name == "" {
		return ""
	}
	name = bracketTagRe.ReplaceAllString(name, "")
	name = stripObfuscationLatin(name)
	name = stripQualityNoise(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// 中文片名 + 可选序号：冰雪奇缘2、唐人街探案3
	if m := hanTitleWithNumberRe.FindStringSubmatch(name); len(m) >= 2 {
		title := m[1]
		if len(m) >= 3 && m[2] != "" {
			title += m[2]
		}
		return title
	}

	if m := leadingHanTitle.FindString(name); m != "" {
		rest := strings.TrimPrefix(name, m)
		if sub := seriesNoSuffixRe.FindString(rest); sub != "" {
			return m + sub
		}
		if digits := trailingDigitsRe.FindString(rest); digits != "" {
			return m + digits
		}
		return m
	}

	if i := strings.Index(name, "."); i > 0 {
		return cleanTitle(name[:i])
	}
	return cleanTitle(name)
}

var (
	hanTitleWithNumberRe = regexp.MustCompile(`^([\p{Han}]+(?:之[\p{Han}]+)*)(\d+)?`)
	trailingDigitsRe     = regexp.MustCompile(`^\d+`)
)

func stripObfuscationLatin(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Han, r), unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsLetter(r) && r < 128:
			continue
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), "")
}

// MovieSearchCandidates 生成电影 TMDB 搜索候选（文件名 + 上级文件夹 + 英文片名）
func MovieSearchCandidates(storagePath, parsedTitle string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	add(parsedTitle)
	if eng := extractEnglishTitle(parsedTitle); eng != "" {
		add(eng)
	}
	if han := extractLeadingHanTitle(parsedTitle); han != "" && han != parsedTitle {
		add(han)
	}

	for _, folder := range movieFolderChain(storagePath, 3) {
		ft := MovieFolderTitle(folder)
		if ft != "" {
			add(ft)
		}
		if eng := extractEnglishTitle(folder); eng != "" {
			add(eng)
		}
		if y := SeriesFolderYear(folder); y != nil {
			_ = y
		}
	}

	if isWeakMovieTitle(parsedTitle) && len(out) > 1 {
		// 弱文件名时优先尝试文件夹/英文标题
		reordered := append([]string{}, out[1:]...)
		reordered = append(reordered, parsedTitle)
		return reordered
	}
	return out
}

func movieFolderChain(storagePath string, maxDepth int) []string {
	dir := filepath.Dir(storagePath)
	var names []string
	for i := 0; i < maxDepth && dir != "" && dir != "."; i++ {
		base := filepath.Base(dir)
		if isMovieCategoryFolder(base) {
			break
		}
		names = append(names, base)
		dir = filepath.Dir(dir)
	}
	return names
}

func isMovieCategoryFolder(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch lower {
	case "media", "movies", "movie", "films", "film", "downloads", "download", "电影", "下载", "video", "videos", "root":
		return true
	default:
		return false
	}
}

func isWeakMovieTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return true
	}
	cleaned := stripQualityNoise(title)
	if countHanRunes(cleaned) >= 2 {
		return false
	}
	if latinWordCount(cleaned) >= 2 {
		if !isAcronymLikeTitle(cleaned) {
			return false
		}
	}
	if len(cleaned) <= 6 && !containsHan(cleaned) {
		return true
	}
	lower := strings.ToLower(title)
	weakStarts := []string{"4k", "3d", "1080p", "2160p", "720p", "d c", "dc "}
	for _, p := range weakStarts {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	if qualityTokenRe.ReplaceAllString(title, "") == "" {
		return true
	}
	return len(strings.Fields(cleaned)) <= 1 && !containsHan(cleaned)
}

func isQualityHeavyTitle(title string) bool {
	clean := stripQualityNoise(title)
	return clean != title && len([]rune(clean)) >= 2
}

func stripQualityNoise(title string) string {
	s := qualityTokenRe.ReplaceAllString(title, " ")
	s = bracketTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "  ", " ")
	return strings.TrimSpace(s)
}

func extractEnglishTitle(s string) string {
	if i := strings.Index(s, "["); i >= 0 {
		inner := s[i+1:]
		if j := strings.Index(inner, "]"); j >= 0 {
			inner = inner[:j]
		}
		if latinWordCount(inner) >= 2 {
			return cleanTitle(inner)
		}
	}

	matches := englishTitleRe.FindAllString(s, -1)
	best := ""
	bestWords := 0
	for _, m := range matches {
		m = strings.TrimSpace(m)
		words := latinWordCount(m)
		if words > bestWords || (words == bestWords && len(m) > len(best)) {
			best = m
			bestWords = words
		}
	}
	if bestWords >= 2 {
		if isPlausibleEnglishTitle(best) {
			return cleanTitle(best)
		}
		return ""
	}
	if bestWords == 1 && len(best) >= 4 {
		return cleanTitle(best)
	}
	return ""
}

func extractLeadingHanTitle(s string) string {
	s = stripQualityNoise(s)
	if m := leadingHanTitle.FindString(s); m != "" {
		rest := strings.TrimPrefix(s, m)
		if sub := seriesNoSuffixRe.FindString(rest); sub != "" {
			return m + sub
		}
		return m
	}
	return ""
}

func latinWordCount(s string) int {
	n := 0
	for _, w := range strings.Fields(s) {
		hasLatin := false
		for _, r := range w {
			if unicode.IsLetter(r) && r < 128 {
				hasLatin = true
				break
			}
		}
		if hasLatin {
			n++
		}
	}
	return n
}

func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func countHanRunes(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			n++
		}
	}
	return n
}

func isAcronymLikeTitle(title string) bool {
	words := strings.Fields(title)
	if len(words) < 2 {
		return false
	}
	for _, w := range words {
		if len([]rune(w)) > 2 {
			return false
		}
	}
	return true
}

func isUsableFolderMovieTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	return countHanRunes(title) >= 2 || latinWordCount(title) >= 2
}

func isPlausibleEnglishTitle(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	words := strings.Fields(s)
	latinWords := 0
	for _, w := range words {
		hasLatin := false
		for _, r := range w {
			if unicode.IsLetter(r) && r < 128 {
				hasLatin = true
				break
			}
		}
		if !hasLatin {
			continue
		}
		if len([]rune(w)) >= 4 || latinWords > 0 {
			latinWords++
		}
	}
	return latinWords >= 2
}

func refineMovieTitleFromPath(fullPath string, p *ParsedFile) {
	if p == nil || p.Type != "movie" {
		return
	}
	for _, folder := range movieFolderChain(fullPath, 3) {
		if p.Year == nil {
			if y := SeriesFolderYear(folder); y != nil {
				p.Year = y
			}
		}
	}
	if isWeakMovieTitle(p.Title) || isQualityHeavyTitle(p.Title) {
		for _, folder := range movieFolderChain(fullPath, 3) {
			if ft := MovieFolderTitle(folder); isUsableFolderMovieTitle(ft) {
				p.Title = ft
				break
			}
		}
	}
	if cleaned := stripQualityNoise(p.Title); cleaned != "" && !isWeakMovieTitle(cleaned) {
		if containsHan(cleaned) || latinWordCount(cleaned) >= 2 {
			p.Title = cleaned
		}
	}
	if eng := extractEnglishTitle(filepath.Base(fullPath)); isPlausibleEnglishTitle(eng) && isWeakMovieTitle(p.Title) {
		p.Title = eng
	}
}
