package recommend

import (
	"context"
	"sort"
	"strconv"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/history"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scraper"

	"github.com/google/uuid"
)

// GuessYouLike 猜你喜欢：观影历史 + 口味画像 + TMDB 相似推荐
func (e *Engine) GuessYouLike(ctx context.Context, profileID string, limit, discoverLimit int) ([]Item, error) {
	if limit <= 0 {
		limit = 20
	}
	if discoverLimit < 0 {
		discoverLimit = 0
	}
	if discoverLimit > limit/2 {
		discoverLimit = limit / 2
	}

	watched := map[uuid.UUID]bool{}
	genreWeight := map[string]float64{}
	typeWeight := map[common.MediaType]float64{}

	var seeds []history.History
	if profileID != "" && profileID != "anonymous" {
		hs, err := e.histRepo.ListByProfile(ctx, profileID, 50)
		if err != nil {
			return nil, err
		}
		seeds = hs
		for _, h := range hs {
			watched[h.MediaID] = true
			if h.Media == nil {
				continue
			}
			w := historyWeight(h)
			typeWeight[h.Media.Type] += w
			for _, g := range h.Media.Genres {
				genreWeight[g] += w
			}
		}
	}

	scores := map[string]itemScore{} // key: local uuid or tmdb:123

	addLocal := func(m *media.Media, score float64) {
		if m == nil || watched[m.ID] {
			return
		}
		key := "local:" + m.ID.String()
		if cur, ok := scores[key]; !ok || score > cur.score {
			scores[key] = itemScore{local: m, score: score}
		}
	}

	addTMDB := func(entry scraper.SearchEntry, score float64) {
		if entry.ID <= 0 {
			return
		}
		key := "tmdb:" + strconv.Itoa(entry.ID)
		if cur, ok := scores[key]; ok && score <= cur.score {
			return
		}
		if local, err := e.mediaRepo.GetByTMDBID(ctx, entry.ID); err == nil && local != nil {
			if watched[local.ID] {
				return
			}
			addLocal(local, score+0.2)
			return
		}
		if discoverLimit <= 0 || e.tmdb == nil {
			return
		}
		scores[key] = itemScore{tmdb: &entry, score: score, external: true}
	}

	// 1) 基于最近观看种子的 TMDB 推荐 + 本地内容相似
	seedMedias := pickSeedMedias(seeds, 3)
	for i, m := range seedMedias {
		seedBoost := 1.0 - float64(i)*0.15
		if cb, err := e.ContentBased(ctx, m.ID.String(), limit); err == nil {
			for j, c := range cb {
				w := seedBoost * (1.0 - float64(j)/float64(len(cb)+1))
				addLocal(&c, w*0.8)
			}
		}
		if m.TMDBID != nil && e.tmdb != nil {
			e.fetchTMDBRecs(ctx, *m.TMDBID, m.Type, seedBoost, addTMDB)
		}
	}

	// 2) 口味画像：偏好类型 + 标签
	preferredType := preferredMediaType(typeWeight)
	if len(genreWeight) > 0 {
		topGenres := topKeys(genreWeight, 3)
		f := repository.MediaFilter{
			Sort:       "rating",
			SortDesc:   true,
			MinRating:  ptrF(6.0),
			ExcludeAdult: false,
		}
		if preferredType != "" {
			f.Type = string(preferredType)
		}
		candidates, _, err := e.mediaRepo.List(ctx, f, limit*3, 0)
		if err == nil {
			for _, c := range candidates {
				if watched[c.ID] {
					continue
				}
				gScore := sharedGenreScore(c.Genres, topGenres)
				if gScore <= 0 && preferredType != "" && c.Type != preferredType {
					continue
				}
				addLocal(&c, gScore*0.6+c.Rating/20.0)
			}
		}
	}

	// 3) 无历史：TMDB 热门 + 库内高分
	if len(seeds) == 0 {
		if e.tmdb != nil {
			for _, mt := range []string{"tv", "movie"} {
				trend, err := e.tmdb.GetTrending(ctx, mt, "week")
				if err != nil {
					continue
				}
				for j, entry := range trend.Results {
					addTMDB(entry, 1.0-float64(j)*0.03)
				}
			}
		}
		items, _, err := e.mediaRepo.List(ctx, repository.MediaFilter{
			Sort: "rating", SortDesc: true, MinRating: ptrF(7.0),
		}, limit, 0)
		if err == nil {
			for j, m := range items {
				addLocal(&m, 0.5-float64(j)*0.02)
			}
		}
	}

	ranked := rankScores(scores, limit, discoverLimit)
	if len(ranked) == 0 {
		ranked = e.guessFallback(ctx, watched, limit, discoverLimit)
	}
	return e.scoresToItems(ctx, ranked), nil
}

// guessFallback 无评分/未刮削媒资时仍给出库内推荐
func (e *Engine) guessFallback(ctx context.Context, watched map[uuid.UUID]bool, limit, discoverLimit int) []itemScore {
	if limit <= 0 {
		limit = 20
	}
	scores := map[string]itemScore{}

	addLocal := func(m *media.Media, score float64) {
		if m == nil || watched[m.ID] {
			return
		}
		key := "local:" + m.ID.String()
		if cur, ok := scores[key]; !ok || score > cur.score {
			scores[key] = itemScore{local: m, score: score}
		}
	}

	addTMDB := func(entry scraper.SearchEntry, score float64) {
		if entry.ID <= 0 || e.tmdb == nil || discoverLimit <= 0 {
			return
		}
		key := "tmdb:" + strconv.Itoa(entry.ID)
		if cur, ok := scores[key]; ok && score <= cur.score {
			return
		}
		if local, err := e.mediaRepo.GetByTMDBID(ctx, entry.ID); err == nil && local != nil {
			if !watched[local.ID] {
				addLocal(local, score+0.2)
			}
			return
		}
		scores[key] = itemScore{tmdb: &entry, score: score, external: true}
	}

	items, _, err := e.mediaRepo.List(ctx, repository.MediaFilter{
		Sort:     "created_at",
		SortDesc: true,
	}, limit*3, 0)
	if err == nil {
		for j, m := range items {
			addLocal(&m, 0.8-float64(j)*0.02)
		}
	}

	if e.tmdb != nil {
		for _, mt := range []string{"tv", "movie"} {
			trend, err := e.tmdb.GetTrending(ctx, mt, "week")
			if err != nil {
				continue
			}
			for j, entry := range trend.Results {
				addTMDB(entry, 0.6-float64(j)*0.02)
			}
		}
	}

	return rankScores(scores, limit, discoverLimit)
}

type itemScore struct {
	local    *media.Media
	tmdb     *scraper.SearchEntry
	score    float64
	external bool
}

func (e *Engine) fetchTMDBRecs(ctx context.Context, tmdbID int, mt common.MediaType, boost float64, add func(scraper.SearchEntry, float64)) {
	var res *scraper.SearchResult
	var err error
	switch mt {
	case common.MediaTypeMovie:
		res, err = e.tmdb.GetMovieRecommendations(ctx, tmdbID)
	default:
		res, err = e.tmdb.GetTVRecommendations(ctx, tmdbID)
	}
	if err != nil || res == nil {
		return
	}
	for j, entry := range res.Results {
		if entry.MediaType == "person" {
			continue
		}
		w := boost * (1.0 - float64(j)/float64(len(res.Results)+1))
		add(entry, w)
	}
}

func pickSeedMedias(seeds []history.History, n int) []*media.Media {
	out := make([]*media.Media, 0, n)
	seen := map[uuid.UUID]bool{}
	for _, h := range seeds {
		if h.Media == nil || seen[h.MediaID] {
			continue
		}
		if h.Progress <= 0 && !h.Completed {
			continue
		}
		seen[h.MediaID] = true
		m := h.Media
		out = append(out, m)
		if len(out) >= n {
			break
		}
	}
	return out
}

func historyWeight(h history.History) float64 {
	if h.Completed {
		return 2.0
	}
	if h.Duration > 0 && float64(h.Progress)/float64(h.Duration) > 0.5 {
		return 1.5
	}
	if h.Progress > 0 {
		return 1.0
	}
	return 0.3
}

func preferredMediaType(w map[common.MediaType]float64) common.MediaType {
	var best common.MediaType
	var max float64
	for t, v := range w {
		if v > max {
			max = v
			best = t
		}
	}
	return best
}

func topKeys(m map[string]float64, n int) []string {
	type kv struct {
		k string
		v float64
	}
	arr := make([]kv, 0, len(m))
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	if len(arr) > n {
		arr = arr[:n]
	}
	out := make([]string, len(arr))
	for i, x := range arr {
		out[i] = x.k
	}
	return out
}

func sharedGenreScore(genres, preferred []string) float64 {
	if len(preferred) == 0 {
		return 0.1
	}
	set := map[string]bool{}
	for _, g := range genres {
		set[g] = true
	}
	var score float64
	for _, p := range preferred {
		if set[p] {
			score += 1.0
		}
	}
	return score / float64(len(preferred))
}

func rankScores(scores map[string]itemScore, limit, discoverLimit int) []itemScore {
	arr := make([]itemScore, 0, len(scores))
	for _, v := range scores {
		arr = append(arr, v)
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].score > arr[j].score })

	out := make([]itemScore, 0, limit)
	externalCount := 0
	for _, it := range arr {
		if len(out) >= limit {
			break
		}
		if it.external {
			if externalCount >= discoverLimit {
				continue
			}
			externalCount++
		}
		out = append(out, it)
	}
	return out
}

func (e *Engine) scoresToItems(ctx context.Context, ranked []itemScore) []Item {
	out := make([]Item, 0, len(ranked))
	for _, r := range ranked {
		if r.local != nil {
			out = append(out, Item{Media: r.local, External: false})
			continue
		}
		if r.tmdb == nil || e.tmdb == nil {
			continue
		}
		entry := *r.tmdb
		title := entry.Title
		if title == "" {
			title = entry.Name
		}
		mediaType := "movie"
		if entry.MediaType == "tv" || entry.FirstAirDate != "" {
			mediaType = "tvshow"
		}
		var year *int
		dateStr := entry.ReleaseDate
		if dateStr == "" {
			dateStr = entry.FirstAirDate
		}
		if len(dateStr) >= 4 {
			if y, err := strconv.Atoi(dateStr[:4]); err == nil {
				year = &y
			}
		}
		out = append(out, Item{
			TMDBID:        entry.ID,
			Title:         title,
			PosterURL:     e.tmdb.PosterURL(entry.PosterPath, "w342"),
			BackdropURL:   e.tmdb.BackdropURL(entry.BackdropPath, "w780"),
			MediaType:     mediaType,
			Rating:        entry.VoteAverage,
			Year:          year,
			Overview:      entry.Overview,
			External:      true,
		})
	}
	_ = ctx
	return out
}
