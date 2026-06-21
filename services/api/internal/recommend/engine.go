// Package recommend 提供推荐引擎
//
// 三种算法：
//   - Content-based：基于内容相似度（标签 + 类型 + 评分）
//   - CF（Collaborative Filtering）：基于用户-物品矩阵
//   - Hybrid：加权融合
//
// 推荐结果缓存到 recommendations 表，避免每次请求重算。
package recommend

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/mediahub/api/internal/domain/history"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scraper"

	"github.com/google/uuid"
)

// Engine 推荐引擎
type Engine struct {
	mediaRepo *repository.MediaRepo
	histRepo  *repository.HistoryRepo
	tmdb      *scraper.TMDBClient
}

// NewEngine 构造
func NewEngine(m *repository.MediaRepo, h *repository.HistoryRepo, tmdb *scraper.TMDBClient) *Engine {
	return &Engine{mediaRepo: m, histRepo: h, tmdb: tmdb}
}

// ---- Content-based ----

// ContentBased 基于内容相似度推荐
// 相似度 = 类型匹配 + 标签 Jaccard + 评分权重
func (e *Engine) ContentBased(ctx context.Context, seedMediaID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 20
	}
	seed, err := e.mediaRepo.GetByID(ctx, seedMediaID)
	if err != nil {
		return nil, err
	}

	// 拉取候选（同类目 + 未看过）
	candidates, _, err := e.mediaRepo.List(ctx, repository.MediaFilter{
		Type:       string(seed.Type),
		Sort:       "rating",
		SortDesc:   true,
		ExcludeIDs: []string{seed.ID.String()},
		MinRating:  ptrF(6.0),
	}, limit*4, 0)
	if err != nil {
		return nil, err
	}

	// 计算相似度
	type scored struct {
		m     media.Media
		score float64
	}
	scoredItems := make([]scored, len(candidates))
	for i, c := range candidates {
		scoredItems[i] = scored{
			m:     c,
			score: contentSimilarity(seed, &c),
		}
	}

	// 按相似度排序
	sort.Slice(scoredItems, func(i, j int) bool {
		if scoredItems[i].score != scoredItems[j].score {
			return scoredItems[i].score > scoredItems[j].score
		}
		return scoredItems[i].m.Rating > scoredItems[j].m.Rating
	})

	if len(scoredItems) > limit {
		scoredItems = scoredItems[:limit]
	}

	out := make([]media.Media, len(scoredItems))
	for i, s := range scoredItems {
		out[i] = s.m
	}
	return out, nil
}

// contentSimilarity 计算两个媒资的内容相似度（0-1）
func contentSimilarity(a, b *media.Media) float64 {
	if a.ID == b.ID {
		return 1.0
	}

	// 类型一致：+0.3
	typeMatch := 0.0
	if a.Type == b.Type {
		typeMatch = 0.3
	}

	// 年份接近：+0.1（10 年内）
	yearMatch := 0.0
	if a.Year != nil && b.Year != nil {
		diff := math.Abs(float64(*a.Year - *b.Year))
		if diff <= 10 {
			yearMatch = 0.1 * (1 - diff/10)
		}
	}

	// 标签 Jaccard：+0.4
	tagJaccard := jaccard(a.Genres, b.Genres) * 0.4

	// 评分接近：+0.2（分差 ≤ 2）
	ratingMatch := 0.0
	ratingDiff := math.Abs(a.Rating - b.Rating)
	if ratingDiff <= 2 {
		ratingMatch = 0.2 * (1 - ratingDiff/2)
	}

	return typeMatch + yearMatch + tagJaccard + ratingMatch
}

// jaccard 集合 Jaccard 相似度
func jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	set := make(map[string]bool)
	for _, x := range a {
		set[x] = true
	}
	intersection := 0
	for _, y := range b {
		if set[y] {
			intersection++
		}
	}
	union := len(set) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// ---- Collaborative Filtering ----

// Collaborative 基于协同过滤推荐
// 简化版：找到看过 seed 的其他用户，看他们还看过什么
func (e *Engine) Collaborative(ctx context.Context, profileID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 20
	}

	// 1. 当前用户的播放历史（取最近 N 个看过且完成的）
	myHistory, err := e.histRepo.ListByProfile(ctx, profileID, 50)
	if err != nil {
		return nil, err
	}
	if len(myHistory) == 0 {
		return nil, nil
	}

	// 2. 找出看过类似媒资的其他 profile
	candidateProfiles, err := e.findSimilarProfiles(ctx, myHistory, profileID, 30)
	if err != nil {
		return nil, err
	}
	if len(candidateProfiles) == 0 {
		return nil, nil
	}

	// 3. 这些 profile 看过的、我没看过的媒资
	seenIDs := map[uuid.UUID]bool{}
	for _, h := range myHistory {
		seenIDs[h.MediaID] = true
	}

	scores := map[uuid.UUID]float64{} // mediaID -> score
	for _, pid := range candidateProfiles {
		theirHistory, err := e.histRepo.ListByProfile(ctx, pid, 100)
		if err != nil {
			continue
		}
		for _, h := range theirHistory {
			if seenIDs[h.MediaID] {
				continue
			}
			// 评分 = 用户相似度 * 完成度
			score := 1.0
			if h.Completed {
				score = 1.0
			} else if h.Progress > 0 {
				score = 0.6
			}
			scores[h.MediaID] += score
		}
	}

	// 4. 取 top N
	type scored struct {
		id    uuid.UUID
		score float64
	}
	ranked := make([]scored, 0, len(scores))
	for id, s := range scores {
		ranked = append(ranked, scored{id: id, score: s})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	out := make([]media.Media, 0, len(ranked))
	for _, r := range ranked {
		m, err := e.mediaRepo.GetByID(ctx, r.id.String())
		if err != nil {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

// findSimilarProfiles 找出和当前用户口味相似的 profile
func (e *Engine) findSimilarProfiles(ctx context.Context, myHistory []history.History, excludeProfileID string, limit int) ([]string, error) {
	// 当前用户看过的媒资集合
	myMediaIDs := make(map[uuid.UUID]bool, len(myHistory))
	for _, h := range myHistory {
		myMediaIDs[h.MediaID] = true
	}

	// 全局统计：每个 profile 和当前用户的 Jaccard
	allHistory, _, err := e.mediaRepo.List(ctx, repository.MediaFilter{
		Sort: "rating",
		SortDesc: true,
	}, 100, 0)
	if err != nil {
		return nil, err
	}
	_ = allHistory // 用不到

	// 简化版：直接查询所有历史（不去 join users/profiles），按 profile_id 分组
	// 这里用 ListByProfile 单个查，性能不佳，但家庭场景数据量小
	// W4 优化：加 SQL 聚合查询

	// 由于没有全局 profile 历史查询 API，这里退化：取最近创建的 N 个 profile 做对比
	// W4 用 SQL 优化
	return []string{}, nil
}

// ---- Hybrid ----

// Hybrid 混合推荐（加权融合）
func (e *Engine) Hybrid(ctx context.Context, profileID string, seedMediaID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 20
	}

	// 无种子时从观影历史取最近观看
	if seedMediaID == "" && profileID != "" && profileID != "anonymous" {
		if seeds := pickSeedMediasFromProfile(ctx, e.histRepo, profileID, 1); len(seeds) > 0 {
			seedMediaID = seeds[0].ID.String()
		}
	}

	scores := map[uuid.UUID]float64{}

	// Content-based：基于内容相似度
	if seedMediaID != "" {
		cb, err := e.ContentBased(ctx, seedMediaID, limit)
		if err == nil {
			for i, m := range cb {
				// 排名权重：越靠前分越高
				weight := 1.0 - float64(i)/float64(len(cb))
				scores[m.ID] += weight * 0.5 // 50% 权重
			}
		}
	}

	// CF：基于协同过滤
	if profileID != "" {
		cf, err := e.Collaborative(ctx, profileID, limit)
		if err == nil {
			for i, m := range cf {
				weight := 1.0 - float64(i)/float64(len(cf))
				scores[m.ID] += weight * 0.5 // 50% 权重
			}
		}
	}

	// 加分项：评分加成
	for id := range scores {
		m, err := e.mediaRepo.GetByID(ctx, id.String())
		if err != nil {
			continue
		}
		scores[id] += m.Rating / 20.0 // 评分加成最多 +0.5
	}

	// 排序
	type scored struct {
		id    uuid.UUID
		score float64
	}
	ranked := make([]scored, 0, len(scores))
	for id, s := range scores {
		ranked = append(ranked, scored{id: id, score: s})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	out := make([]media.Media, 0, len(ranked))
	for _, r := range ranked {
		m, err := e.mediaRepo.GetByID(ctx, r.id.String())
		if err != nil {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func pickSeedMediasFromProfile(ctx context.Context, histRepo *repository.HistoryRepo, profileID string, n int) []*media.Media {
	hs, err := histRepo.ListByProfile(ctx, profileID, 20)
	if err != nil {
		return nil
	}
	return pickSeedMedias(hs, n)
}

// ---- Cache ----

// CacheRecommendation 缓存推荐结果
func (e *Engine) CacheRecommendation(ctx context.Context, algo string, profileID string, mediaIDs []uuid.UUID, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	rec := struct {
		algo      string
		profileID string
		mediaIDs  []uuid.UUID
		expiresAt time.Time
	}{
		algo:      algo,
		profileID: profileID,
		mediaIDs:  mediaIDs,
		expiresAt: time.Now().Add(ttl),
	}
	_ = rec
	// TODO W3.D5: 写库到 recommendations 表
	return nil
}

// ptrF float64 helper (in service.go)

