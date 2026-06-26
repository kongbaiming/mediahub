package scanner

import (
	"regexp"
	"strconv"
	"strings"
)

// EmbeddedMeta 视频文件内嵌元数据（ffprobe tags）
type EmbeddedMeta struct {
	Title      string
	Show       string
	Season     *int
	Episode    *int
	IMDBID     string
	Year       *int
	DurationSec int
}

var imdbTagRe = regexp.MustCompile(`(?i)^tt\d{7,8}$`)

// ExtractEmbeddedMeta 从 ffprobe 结果提取内嵌片名/剧集/IMDB 等
func ExtractEmbeddedMeta(p *ProbeResult) EmbeddedMeta {
	if p == nil {
		return EmbeddedMeta{}
	}
	meta := EmbeddedMeta{DurationSec: p.Extract().Duration}
	tags := mergeProbeTags(p)
	meta.Title = firstTag(tags,
		"title", "TITLE", "name", "NAME", "movie", "movie_name", "Movie",
		"track", "Track", "TrackTitle",
	)
	meta.Show = firstTag(tags,
		"show", "SHOW", "series", "SERIES", "tvshow", "tv_show",
		"series_name", "show_name", "album", "ALBUM",
	)
	if meta.Show == "" && meta.Title != "" && (meta.Season != nil || tagInt(tags, "season_number", "SEASON", "season") != nil) {
		meta.Show = meta.Title
	}
	if s := tagInt(tags, "season_number", "SEASON", "season", "SeasonNumber"); s != nil {
		meta.Season = s
	}
	if e := tagInt(tags, "episode_id", "EPISODE", "episode", "EpisodeNumber", "episode_sort", "PART_NUMBER"); e != nil {
		meta.Episode = e
	}
	meta.IMDBID = normalizeIMDBID(firstTag(tags, "imdb", "IMDB", "imdb_id", "IMDB_ID", "IMDBID"))
	if meta.Year == nil {
		meta.Year = yearFromTags(tags)
	}
	return meta
}

func mergeProbeTags(p *ProbeResult) map[string]string {
	out := map[string]string{}
	for k, v := range p.Format.Tags {
		out[k] = v
	}
	for _, s := range p.Streams {
		for k, v := range s.Tags {
			if _, ok := out[k]; !ok {
				out[k] = v
			}
		}
	}
	return out
}

func firstTag(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		for k, v := range tags {
			if strings.EqualFold(k, key) {
				v = strings.TrimSpace(v)
				if v != "" && !isNoiseTagValue(v) {
					return cleanTitle(v)
				}
			}
		}
	}
	return ""
}

func tagInt(tags map[string]string, keys ...string) *int {
	s := firstTagRaw(tags, keys...)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return nil
	}
	return &n
}

func firstTagRaw(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		for k, v := range tags {
			if strings.EqualFold(k, key) {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func yearFromTags(tags map[string]string) *int {
	for _, key := range []string{"date", "DATE", "year", "YEAR", "DATE_RELEASED", "creation_time", "CREATION_TIME"} {
		for k, v := range tags {
			if !strings.EqualFold(k, key) {
				continue
			}
			if y := parseYearFromTag(v); y != nil {
				return y
			}
		}
	}
	return nil
}

func parseYearFromTag(v string) *int {
	v = strings.TrimSpace(v)
	if len(v) >= 4 {
		if y, err := strconv.Atoi(v[:4]); err == nil && y >= 1900 && y <= 2100 {
			return &y
		}
	}
	return nil
}

func normalizeIMDBID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(s), "tt") {
		if digits := regexp.MustCompile(`\d{7,8}`).FindString(s); digits != "" {
			s = "tt" + digits
		}
	}
	if imdbTagRe.MatchString(s) {
		return strings.ToLower(s)
	}
	return ""
}

func isNoiseTagValue(v string) bool {
	lower := strings.ToLower(v)
	noise := []string{"und", "unknown", "ffmpeg", "handbrake", "makemkv", "x264", "x265"}
	for _, n := range noise {
		if lower == n {
			return true
		}
	}
	return false
}

// SearchCandidateOpts 控制 TMDB 搜索候选顺序与过滤
type SearchCandidateOpts struct {
	PreferFolderOverEmbedded bool   // true 时文件夹/标题优先，内嵌元数据靠后
	ReferenceTitle           string // 用于判断内嵌标题是否可靠（如 CMS 手动标题）
}

// PrependEmbeddedCandidates 内嵌标题优先插入搜索候选（去重）
func PrependEmbeddedCandidates(candidates []string, emb *EmbeddedMeta) []string {
	return MergeEmbeddedCandidates(candidates, emb, nil)
}

// MergeEmbeddedCandidates 合并内嵌元数据搜索候选
func MergeEmbeddedCandidates(candidates []string, emb *EmbeddedMeta, opts *SearchCandidateOpts) []string {
	if emb == nil {
		return candidates
	}
	ref := ""
	preferFolder := false
	if opts != nil {
		ref = strings.TrimSpace(opts.ReferenceTitle)
		preferFolder = opts.PreferFolderOverEmbedded
	}

	seen := map[string]struct{}{}
	var embedded, base []string
	addSeen := func(list *[]string, s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		*list = append(*list, s)
	}
	addEmbedded := func(s string) {
		if isUnreliableEmbeddedTitle(s, ref) {
			return
		}
		addSeen(&embedded, s)
		for _, alias := range movieTitleAliases(s) {
			addSeen(&embedded, alias)
		}
		if eng := extractEnglishTitle(s); eng != "" && !isUnreliableEmbeddedTitle(eng, ref) {
			addSeen(&embedded, eng)
		}
	}
	for _, s := range []string{emb.Show, emb.Title} {
		addEmbedded(s)
	}
	for _, c := range candidates {
		addSeen(&base, c)
	}
	if preferFolder {
		return append(base, embedded...)
	}
	return append(embedded, base...)
}

var episodeEmbeddedTitleRe = regexp.MustCompile(`^第\s*\d+\s*集$`)

// isUnreliableEmbeddedTitle 内嵌标题与已知剧名/文件夹明显不符时跳过（如 Mandarin 误匹配）
func isUnreliableEmbeddedTitle(embedded, reference string) bool {
	embedded = strings.TrimSpace(embedded)
	reference = strings.TrimSpace(reference)
	if embedded == "" {
		return true
	}
	if episodeEmbeddedTitleRe.MatchString(embedded) {
		return true
	}
	refHan := countHanRunes(reference)
	embHan := countHanRunes(embedded)
	if refHan >= 2 && embHan == 0 {
		return true
	}
	lower := strings.ToLower(embedded)
	if refHan >= 2 && strings.Contains(lower, "mandarin") {
		return true
	}
	if refHan >= 2 && latinWordCount(embedded) >= 1 && embHan == 0 {
		if !strings.Contains(strings.ToLower(reference), lower) {
			return true
		}
	}
	return false
}
