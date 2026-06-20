package scraper

import "strings"

// SearchQueries 生成 TMDB 搜索词列表（主标题 + 去副标题变体）
func SearchQueries(title string) []string {
	t := strings.TrimSpace(title)
	if t == "" {
		return nil
	}
	seen := map[string]struct{}{t: {}}
	out := []string{t}

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

	// 「加勒比海盗5：死无对证」→ 也尝试「加勒比海盗5」
	for _, sep := range []string{"：", ":", " - ", "－", "—"} {
		if i := strings.Index(t, sep); i > 0 {
			add(t[:i])
		}
	}

	return out
}
