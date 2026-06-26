package scraper

import (
	"regexp"
	"strings"
)

var (
	bracketTagRe   = regexp.MustCompile(`\[[^\]]*\]`)
	yearInParensRe = regexp.MustCompile(`[\(（]\s*(19|20)\d{2}\s*[\)）]`)
	qualityNoiseRe = regexp.MustCompile(`(?i)(4k|3d|2160p|1080p|720p|480p|60fps|120fps|\d+fps|\d+帧|出屏|国配|双语|原声|国语|粤语|台语|国粤台|web-dl|bluray|x264|x265|h265|hevc|10bit|aac|dts|hdr|sbs|half|加长版|白星|修复版)`)
)

// SearchQueries 生成 TMDB 搜索词列表（主标题 + 去副标题变体）
func SearchQueries(title string) []string {
	t := normalizeSearchTitle(title)
	if t == "" {
		return nil
	}
	seen := map[string]struct{}{t: {}}
	out := []string{t}

	add := func(s string) {
		s = normalizeSearchTitle(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	// 「加勒比海盗5：死无对证」→ 也尝试「加勒比海盗5」
	for _, sep := range []string{"：", ":", " - ", "－", "—"} {
		if i := strings.Index(t, sep); i > 0 {
			add(t[:i])
		}
	}

	// 「唐朝诡事录之西行」→ 也尝试主剧名「唐朝诡事录」
	if i := strings.LastIndex(t, "之"); i > 2 {
		add(t[:i])
	}

	// 提取英文片名片段（如 Frozen II）
	if eng := extractLatinTitle(title); eng != "" {
		add(eng)
	}

	return out
}

func extractLatinTitle(title string) string {
	re := regexp.MustCompile(`[A-Za-z][A-Za-z0-9.'\s:-]{2,}`)
	matches := re.FindAllString(title, -1)
	best := ""
	bestWords := 0
	for _, m := range matches {
		words := len(strings.Fields(m))
		if words > bestWords {
			best = strings.TrimSpace(m)
			bestWords = words
		}
	}
	if bestWords >= 2 {
		return best
	}
	return ""
}

func normalizeSearchTitle(title string) string {
	t := strings.TrimSpace(bracketTagRe.ReplaceAllString(title, ""))
	t = strings.ReplaceAll(t, "丨", "")
	t = strings.ReplaceAll(t, "·", "")
	t = strings.ReplaceAll(t, "•", "")
	t = yearInParensRe.ReplaceAllString(t, "")
	t = qualityNoiseRe.ReplaceAllString(t, " ")
	t = strings.Join(strings.Fields(t), " ")
	return strings.TrimSpace(t)
}
