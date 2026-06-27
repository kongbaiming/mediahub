package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mediahub/api/internal/apperr"
)

const maxM3UEntries = 3000

// M3UEntry M3U 频道条目
type M3UEntry struct {
	Title      string
	GroupTitle string
	Logo       string
	StreamURL  string
}

// IsM3UPlaylist 判断内容是否为 M3U 频道列表
func IsM3UPlaylist(content string) bool {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "#EXTM3U") {
		return false
	}
	return strings.Contains(content, "#EXTINF:")
}

// ParseM3U 解析 M3U/M3U8 频道列表
func ParseM3U(content, baseURL string) ([]M3UEntry, error) {
	if !IsM3UPlaylist(content) {
		return nil, apperr.Validation("不是有效的 M3U 频道列表")
	}

	lines := strings.Split(content, "\n")
	out := make([]M3UEntry, 0, 64)
	var pending *M3UEntry

	flush := func() {
		if pending == nil || pending.StreamURL == "" {
			pending = nil
			return
		}
		if pending.Title == "" {
			pending.Title = "未命名频道"
		}
		out = append(out, *pending)
		pending = nil
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF:") {
			flush()
			title, group, logo := parseExtInfLine(line)
			pending = &M3UEntry{Title: title, GroupTitle: group, Logo: logo}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if pending == nil {
			continue
		}
		resolved, err := resolveMediaURL(line, baseURL)
		if err != nil {
			continue
		}
		pending.StreamURL = resolved
		flush()
	}

	if len(out) > maxM3UEntries {
		return nil, apperr.BadRequest(fmt.Sprintf("M3U 频道数量超过上限（%d），请按分组筛选后分批导入", maxM3UEntries))
	}
	if len(out) == 0 {
		return nil, apperr.Validation("M3U 列表中未找到有效频道")
	}
	return out, nil
}

func parseExtInfLine(line string) (title, groupTitle, logo string) {
	rest := strings.TrimPrefix(line, "#EXTINF:")
	commaIdx := strings.LastIndex(rest, ",")
	if commaIdx >= 0 {
		title = strings.TrimSpace(rest[commaIdx+1:])
		rest = rest[:commaIdx]
	} else {
		title = strings.TrimSpace(rest)
	}
	groupTitle = extractM3UAttribute("#"+rest, "group-title")
	logo = extractM3UAttribute("#"+rest, "tvg-logo")
	if logo == "" {
		logo = extractM3UAttribute("#"+rest, "tvg-logo-small")
	}
	return title, groupTitle, logo
}

// M3UGroupStat 分组统计
type M3UGroupStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// SummarizeM3UGroups 统计 M3U 分组
func SummarizeM3UGroups(entries []M3UEntry) []M3UGroupStat {
	counts := make(map[string]int)
	for _, e := range entries {
		name := e.GroupTitle
		if name == "" {
			name = "未分组"
		}
		counts[name]++
	}
	out := make([]M3UGroupStat, 0, len(counts))
	for name, count := range counts {
		out = append(out, M3UGroupStat{Name: name, Count: count})
	}
	return out
}

// FilterM3UByGroup 按分组筛选频道
func FilterM3UByGroup(entries []M3UEntry, groups []string) []M3UEntry {
	if len(groups) == 0 {
		return entries
	}
	allowed := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		allowed[g] = struct{}{}
	}
	out := make([]M3UEntry, 0, len(entries))
	for _, e := range entries {
		name := e.GroupTitle
		if name == "" {
			name = "未分组"
		}
		if _, ok := allowed[name]; ok {
			out = append(out, e)
		}
	}
	return out
}

func validateM3UPlaylistURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.Validation("请填写 M3U 列表地址")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", apperr.Validation("M3U 列表地址格式无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", apperr.Validation("M3U 列表地址须为 http 或 https")
	}
	if !IsSafePublicURL(u) {
		return "", apperr.Validation("M3U 列表地址不允许访问内网地址")
	}
	return normalizeM3UPlaylistURL(u.String())
}

// knownM3UHomepages 常见 IPTV 聚合站首页 → M3U 路径
var knownM3UHomepages = map[string]string{
	"live.zhi35.com": "/iptv.m3u",
}

// normalizeM3UPlaylistURL 将聚合站首页等地址解析为实际 M3U 文件地址
func normalizeM3UPlaylistURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw, nil
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	lower := strings.ToLower(path)
	if path != "" && (strings.HasSuffix(lower, ".m3u") || strings.HasSuffix(lower, ".m3u8")) {
		return parsed.String(), nil
	}
	host := strings.ToLower(parsed.Hostname())
	if suffix, ok := knownM3UHomepages[host]; ok {
		parsed.Path = suffix
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String(), nil
	}
	// 常见约定：站点根路径下提供 iptv.m3u
	if path == "" {
		parsed.Path = "/iptv.m3u"
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String(), nil
	}
	return parsed.String(), nil
}
