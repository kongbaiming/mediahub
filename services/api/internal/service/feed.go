package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mediahub/api/internal/cache"
	"github.com/mediahub/api/internal/domain/layout"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// FeedService Feed 数据填充业务（核心：把布局 + 数据合并成 Feed）
type FeedService struct {
	media     *repository.MediaRepo
	layout    *repository.LayoutRepo
	history   *repository.HistoryRepo
	users     *repository.UserRepo
	recommend RecommendFetcher
	cache     *cache.Cache // 可选（nil = 禁用）
}

// RecommendFetcher 推荐拉取接口（避免循环依赖）
type RecommendFetcher interface {
	Hot(ctx context.Context, limit int) ([]media.Media, error)
	ForProfile(ctx context.Context, profileID string, limit int) ([]media.Media, error)
	SimilarTo(ctx context.Context, mediaID string, limit int) ([]media.Media, error)
}

// NewFeedService 构造
func NewFeedService(m *repository.MediaRepo, l *repository.LayoutRepo, h *repository.HistoryRepo, u *repository.UserRepo, r RecommendFetcher) *FeedService {
	return &FeedService{media: m, layout: l, history: h, users: u, recommend: r}
}

// WithCache 注入缓存（可选）
func (s *FeedService) WithCache(c *cache.Cache) *FeedService {
	s.cache = c
	return s
}

// BuildFeed 构建 Feed（布局 + 实际数据）
//
// 缓存策略：
//  - Key: feed:{platform}:{profileID}
//  - TTL: 5 分钟（Feed 变化不频繁）
//  - invalidate: 布局发布 / 媒资新增 / 刮削完成时调用
func (s *FeedService) BuildFeed(ctx context.Context, platform string, profileID string) (*layout.Feed, error) {
	// 缓存读穿透
	if s.cache != nil {
		key := "feed:" + platform + ":" + profileID
		cached, err := s.cache.GetOrLoad(ctx, key, 5*time.Minute, func(c context.Context) (any, error) {
			return s.buildFeedInternal(c, platform, profileID)
		})
		if err != nil {
			return nil, err
		}
		return decodeFeed(cached)
	}

	return s.buildFeedInternal(ctx, platform, profileID)
}

// decodeFeed 缓存命中时 JSON 反序列化为 map，需转回 *layout.Feed
func decodeFeed(v any) (*layout.Feed, error) {
	if feed, ok := v.(*layout.Feed); ok {
		return feed, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var feed layout.Feed
	if err := json.Unmarshal(b, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

// buildFeedInternal 实际构建逻辑（被缓存层调用）
func (s *FeedService) buildFeedInternal(ctx context.Context, platform string, profileID string) (*layout.Feed, error) {
	l, err := s.layout.GetPublishedForPlatform(ctx, platform, profileID)
	if err != nil {
		return nil, err
	}

	// 模板继承合并（递归）
	l, err = s.resolveLayoutInheritance(ctx, l)
	if err != nil {
		return nil, err
	}

	return s.BuildFeedFromLayout(ctx, l, platform, profileID)
}

// BuildFeedFromLayout 从已合并的布局构建 Feed（编辑器预览 / 播放端共用）
func (s *FeedService) BuildFeedFromLayout(ctx context.Context, l *layout.Layout, platform, profileID string) (*layout.Feed, error) {
	isKid := s.checkIsKid(ctx, profileID)

	feed := &layout.Feed{
		Version:   l.Version,
		Platform:  platform,
		UpdatedAt: l.UpdatedAt,
		Rows:      make([]layout.FeedRow, 0, len(l.Config.Rows)),
	}

	for _, row := range l.Config.Rows {
		if !layout.RowIsVisible(row) {
			continue
		}
		feedRow := layout.FeedRow{
			ID:        row.ID,
			Type:      row.Type,
			Title:     row.Title,
			Subtitle:  row.Subtitle,
			CardStyle: row.CardStyle,
			Config:    row.Config,
		}

		items, err := s.resolveDataSource(ctx, row.Source, profileID, isKid)
		if err != nil {
			continue
		}
		feedRow.Items = items
		feed.Rows = append(feed.Rows, feedRow)
	}

	return feed, nil
}

// resolveLayoutInheritance 递归合并父布局
func (s *FeedService) resolveLayoutInheritance(ctx context.Context, l *layout.Layout) (*layout.Layout, error) {
	if l.ParentID == nil {
		return l, nil
	}

	visited := map[uuid.UUID]bool{l.ID: true}
	current := l
	for current.ParentID != nil {
		if visited[*current.ParentID] {
			return nil, fmt.Errorf("布局继承存在循环引用")
		}
		visited[*current.ParentID] = true

		parent, err := s.layout.GetByID(ctx, current.ParentID.String())
		if err != nil {
			return nil, fmt.Errorf("加载父布局: %w", err)
		}
		current = mergeLayoutInherit(parent, current)

		if len(visited) > 10 {
			return nil, fmt.Errorf("布局继承深度超过 10 层")
		}
	}
	return current, nil
}

// InvalidateFeed 失效 Feed 缓存（布局发布 / 媒资变更时调用）
func (s *FeedService) InvalidateFeed(ctx context.Context, platform string) error {
	if s.cache == nil {
		return nil
	}
	// 通配符删除该 platform 的所有 profile 缓存
	return s.cache.Invalidate(ctx, "feed:"+platform+":*")
}

// InvalidateAll 失效全部 Feed（重大变更时使用）
func (s *FeedService) InvalidateAll(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Invalidate(ctx, "feed:*")
}

// checkIsKid 检查当前 Profile 是否是儿童
func (s *FeedService) checkIsKid(ctx context.Context, profileID string) bool {
	if profileID == "" || profileID == "anonymous" {
		return false
	}
	p, err := s.users.GetProfile(ctx, profileID)
	if err != nil {
		return false
	}
	return p.IsKid
}

// resolveDataSource 解析数据源 → 媒资列表
func (s *FeedService) resolveDataSource(ctx context.Context, ds layout.DataSource, profileID string, isKid bool) ([]layout.FeedItem, error) {
	switch ds.Type {
	case "manual":
		return s.fromManual(ctx, ds.Params)
	case "library":
		return s.fromLibrary(ctx, ds.Params, isKid)
	case "trending":
		return s.fromTrending(ctx, ds.Params, isKid)
	case "continue-watching":
		return s.fromContinueWatching(ctx, profileID, ds.Params)
	case "recently-added":
		return s.fromRecentlyAdded(ctx, ds.Params, isKid)
	case "similar-to":
		return s.fromSimilar(ctx, ds.Params, isKid)
	case "recommend-algorithm":
		return s.fromRecommend(ctx, profileID, ds.Params, isKid)
	case "tag":
		return s.fromTag(ctx, ds.Params, isKid)
	case "union":
		return s.fromUnion(ctx, ds.Params, profileID, isKid)
	default:
		// 未知类型返回空
		return nil, nil
	}
}

func (s *FeedService) fromManual(ctx context.Context, params map[string]any) ([]layout.FeedItem, error) {
	idsAny, ok := params["ids"].([]any)
	if !ok {
		return nil, nil
	}
	var ids []string
	for _, v := range idsAny {
		if str, ok := v.(string); ok {
			ids = append(ids, str)
		}
	}
	return s.fetchByIDs(ctx, ids)
}

func (s *FeedService) fromLibrary(ctx context.Context, params map[string]any, isKid bool) ([]layout.FeedItem, error) {
	f := repository.MediaFilter{}
	if v, ok := params["type"].(string); ok {
		f.Type = v
	}
	if v, ok := params["genre"].(string); ok {
		f.Genre = v
	}
	if v, ok := params["year"].(float64); ok {
		y := int(v)
		f.Year = &y
	}
	if v, ok := params["min_rating"].(float64); ok {
		f.MinRating = &v
	}
	if v, ok := params["sort"].(string); ok {
		f.Sort = v
	}
	f.SortDesc = true
	f.ExcludeAdult = isKid // 儿童模式过滤成人内容
	limit := 20
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	items, _, err := s.media.List(ctx, f, limit, 0)
	if err != nil {
		return nil, err
	}
	return toFeedItems(items, nil), nil
}

func (s *FeedService) fromTrending(ctx context.Context, params map[string]any, isKid bool) ([]layout.FeedItem, error) {
	f := repository.MediaFilter{
		Sort:         "rating",
		SortDesc:     true,
		MinRating:    ptrF(7.0),
		ExcludeAdult: isKid,
	}
	items, _, err := s.media.List(ctx, f, 20, 0)
	if err != nil {
		return nil, err
	}
	return toFeedItems(items, nil), nil
}

func (s *FeedService) fromContinueWatching(ctx context.Context, profileID string, params map[string]any) ([]layout.FeedItem, error) {
	if profileID == "" {
		return nil, nil
	}
	limit := 12
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	hs, err := s.history.ListInProgress(ctx, profileID, limit)
	if err != nil {
		return nil, err
	}
	var items []media.Media
	progressMap := map[uuid.UUID]int{}
	for _, h := range hs {
		if h.Media != nil {
			items = append(items, *h.Media)
			progressMap[h.Media.ID] = h.Progress
		}
	}
	return toFeedItems(items, progressMap), nil
}

func (s *FeedService) fromRecentlyAdded(ctx context.Context, params map[string]any, isKid bool) ([]layout.FeedItem, error) {
	limit := 20
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	f := repository.MediaFilter{
		Sort:         "created_at",
		SortDesc:     true,
		ExcludeAdult: isKid,
	}
	items, _, err := s.media.List(ctx, f, limit, 0)
	if err != nil {
		return nil, err
	}
	return toFeedItems(items, nil), nil
}

func (s *FeedService) fromSimilar(ctx context.Context, params map[string]any, isKid bool) ([]layout.FeedItem, error) {
	mediaIDStr, _ := params["media_id"].(string)
	if mediaIDStr == "" {
		return nil, nil
	}
	source, err := s.media.GetByID(ctx, mediaIDStr)
	if err != nil {
		return nil, err
	}

	excludeIDs := []string{source.ID.String()}
	limit := 20
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	f := repository.MediaFilter{
		Type:         string(source.Type),
		Sort:         "rating",
		SortDesc:     true,
		ExcludeIDs:   excludeIDs,
		MinRating:    ptrF(6.5),
		ExcludeAdult: isKid,
	}
	items, _, err := s.media.List(ctx, f, limit, 0)
	if err != nil {
		return nil, err
	}

	type scored struct {
		m     media.Media
		score int
	}
	var scoredItems []scored
	for _, m := range items {
		score := sharedCount(m.Genres, source.Genres)
		scoredItems = append(scoredItems, scored{m: m, score: score})
	}
	sort.Slice(scoredItems, func(i, j int) bool {
		if scoredItems[i].score != scoredItems[j].score {
			return scoredItems[i].score > scoredItems[j].score
		}
		return scoredItems[i].m.Rating > scoredItems[j].m.Rating
	})

	medias := make([]media.Media, len(scoredItems))
	for i, s := range scoredItems {
		medias[i] = s.m
	}
	return toFeedItems(medias, nil), nil
}

func (s *FeedService) fromRecommend(ctx context.Context, profileID string, params map[string]any, isKid bool) ([]layout.FeedItem, error) {
	algo, _ := params["algo"].(string)
	if algo == "" {
		algo = "hybrid"
	}
	limit := 20
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	var medias []media.Media
	var err error

	switch algo {
	case "hot":
		medias, err = s.recommend.Hot(ctx, limit)
	case "content", "similar":
		seedID, _ := params["media_id"].(string)
		if seedID != "" {
			medias, err = s.recommend.SimilarTo(ctx, seedID, limit)
		}
	case "cf":
		if profileID != "" {
			medias, err = s.recommend.ForProfile(ctx, profileID, limit)
		}
	case "hybrid", "":
		fallthrough
	default:
		if profileID != "" {
			medias, err = s.recommend.ForProfile(ctx, profileID, limit)
		} else {
			medias, err = s.recommend.Hot(ctx, limit)
		}
	}

	if err != nil {
		return nil, err
	}

	// 儿童模式过滤
	if isKid {
		var filtered []media.Media
		for _, m := range medias {
			if !m.IsAdult {
				filtered = append(filtered, m)
			}
		}
		medias = filtered
	}

	return toFeedItems(medias, nil), nil
}

func (s *FeedService) fromTag(ctx context.Context, params map[string]any, isKid bool) ([]layout.FeedItem, error) {
	tag, _ := params["tag"].(string)
	if tag == "" {
		return nil, nil
	}
	limit := 20
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	f := repository.MediaFilter{
		Sort:         "rating",
		SortDesc:     true,
		ExcludeAdult: isKid,
	}
	items, _, err := s.media.List(ctx, f, limit*3, 0)
	if err != nil {
		return nil, err
	}
	var filtered []media.Media
	for _, m := range items {
		for _, t := range m.Tags {
			if t == tag {
				filtered = append(filtered, m)
				break
			}
		}
		if len(filtered) >= limit {
			break
		}
	}
	return toFeedItems(filtered, nil), nil
}

func (s *FeedService) fromUnion(ctx context.Context, params map[string]any, profileID string, isKid bool) ([]layout.FeedItem, error) {
	srcsAny, ok := params["sources"].([]any)
	if !ok {
		return nil, nil
	}
	seen := map[uuid.UUID]bool{}
	var out []layout.FeedItem
	for _, src := range srcsAny {
		srcMap, ok := src.(map[string]any)
		if !ok {
			continue
		}
		ds := layout.DataSource{
			Type:   getString(srcMap, "type"),
			Params: srcMap,
		}
		items, err := s.resolveDataSource(ctx, ds, profileID, isKid)
		if err != nil {
			continue
		}
		for _, it := range items {
			if !seen[it.MediaID] {
				seen[it.MediaID] = true
				out = append(out, it)
			}
		}
	}
	return out, nil
}

func (s *FeedService) fetchByIDs(ctx context.Context, ids []string) ([]layout.FeedItem, error) {
	var out []layout.FeedItem
	for _, id := range ids {
		m, err := s.media.GetByID(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, toFeedItem(m, nil))
	}
	return out, nil
}

// ---- helpers ----

func toFeedItem(m *media.Media, progress *int) layout.FeedItem {
	year := m.Year
	return layout.FeedItem{
		MediaID:      m.ID,
		Title:        m.Title,
		Year:         year,
		PosterURL:    m.PosterURL,
		BackdropURL:  m.BackdropURL,
		Rating:       m.Rating,
		Type:         string(m.Type),
		Overview:     m.Overview,
		Genres:       m.Genres,
		Progress:     progress,
	}
}

func toFeedItems(items []media.Media, progressMap map[uuid.UUID]int) []layout.FeedItem {
	out := make([]layout.FeedItem, len(items))
	for i, m := range items {
		var p *int
		if v, ok := progressMap[m.ID]; ok {
			p = &v
		}
		out[i] = toFeedItem(&m, p)
	}
	return out
}

func sharedCount(a, b []string) int {
	seen := map[string]bool{}
	for _, x := range a {
		seen[x] = true
	}
	c := 0
	for _, y := range b {
		if seen[y] {
			c++
		}
	}
	return c
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func ptrF(v float64) *float64 { return &v }

// 类型断言 helper（处理 JSON number → int）
func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	}
	return 0
}
