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

// PrependEmbeddedCandidates 内嵌标题优先插入搜索候选（去重）
func PrependEmbeddedCandidates(candidates []string, emb *EmbeddedMeta) []string {
	if emb == nil {
		return candidates
	}
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
	for _, s := range []string{emb.Title, emb.Show} {
		add(s)
		for _, alias := range movieTitleAliases(s) {
			add(alias)
		}
		if eng := extractEnglishTitle(s); eng != "" {
			add(eng)
		}
	}
	for _, c := range candidates {
		add(c)
	}
	return out
}
