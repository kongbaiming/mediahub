package scraper

import (
	"regexp"
	"strings"
)

var (
	bracketTagRe   = regexp.MustCompile(`\[[^\]]*\]`)
	yearInParensRe = regexp.MustCompile(`[\(（]\s*(19|20)\d{2}\s*[\)）]`)
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

	return out
}

func normalizeSearchTitle(title string) string {
	t := strings.TrimSpace(bracketTagRe.ReplaceAllString(title, ""))
	t = strings.ReplaceAll(t, "丨", "")
	t = strings.ReplaceAll(t, "·", "")
	t = strings.ReplaceAll(t, "•", "")
	t = yearInParensRe.ReplaceAllString(t, "")
	return strings.TrimSpace(t)
}
