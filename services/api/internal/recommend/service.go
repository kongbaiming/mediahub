package recommend

import (
	"context"
	"time"

	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"

	"github.com/google/uuid"
)

// Service 推荐业务（带缓存）
type Service struct {
	engine   *Engine
	recCache *repository.RecommendRepo
}

// NewService 构造
func NewService(e *Engine, c *repository.RecommendRepo) *Service {
	return &Service{engine: e, recCache: c}
}

// Hot 全局热门推荐（适合未登录用户）
func (s *Service) Hot(ctx context.Context, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 20
	}

	// 尝试从缓存读取
	cached, err := s.recCache.GetLatest(ctx, nil, "hot")
	if err == nil && cached != nil && time.Now().Before(cached.ExpiresAt) {
		return s.fetchMediaByIDs(ctx, cached.MediaIDs)
	}

	// 重新计算
	items, _, err := s.engine.mediaRepo.List(ctx, repository.MediaFilter{
		Sort:      "rating",
		SortDesc:  true,
		MinRating: ptrF(7.0),
	}, limit, 0)
	if err != nil {
		return nil, err
	}

	// 缓存
	ids := make([]uuid.UUID, len(items))
	for i, m := range items {
		ids[i] = m.ID
	}
	_ = s.recCache.Save(ctx, nil, "hot", ids, time.Now().Add(6*time.Hour))

	return items, nil
}

// ForProfile 个人推荐（Hybrid）
func (s *Service) ForProfile(ctx context.Context, profileID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 20
	}
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return nil, err
	}

	// 尝试缓存
	cached, err := s.recCache.GetLatest(ctx, &pid, "hybrid")
	if err == nil && cached != nil && time.Now().Before(cached.ExpiresAt) {
		return s.fetchMediaByIDs(ctx, cached.MediaIDs)
	}

	// 重新计算 Hybrid
	items, err := s.engine.Hybrid(ctx, profileID, "", limit)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		// 兜底：给全局热门
		return s.Hot(ctx, limit)
	}

	// 缓存
	ids := make([]uuid.UUID, len(items))
	for i, m := range items {
		ids[i] = m.ID
	}
	_ = s.recCache.Save(ctx, &pid, "hybrid", ids, time.Now().Add(12*time.Hour))

	return items, nil
}

// SimilarTo 基于内容的相似推荐
func (s *Service) SimilarTo(ctx context.Context, mediaID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.engine.ContentBased(ctx, mediaID, limit)
}

// GuessYouLike 猜你喜欢（观影习惯 + TMDB）
func (s *Service) GuessYouLike(ctx context.Context, profileID string, limit, discoverLimit int) ([]Item, error) {
	if limit <= 0 {
		limit = 20
	}
	if profileID == "" || profileID == "anonymous" {
		return s.engine.GuessYouLike(ctx, "", limit, discoverLimit)
	}
	pid, err := uuid.Parse(profileID)
	if err != nil {
		return nil, err
	}

	cached, err := s.recCache.GetLatest(ctx, &pid, "for-you")
	if err == nil && cached != nil && time.Now().Before(cached.ExpiresAt) {
		return s.fetchGuessByIDs(ctx, cached.MediaIDs)
	}

	items, err := s.engine.GuessYouLike(ctx, profileID, limit, discoverLimit)
	if err != nil {
		return nil, err
	}

	// 缓存库内媒资 ID（外部 TMDB 项不缓存）
	var ids []uuid.UUID
	for _, it := range items {
		if it.Media != nil {
			ids = append(ids, it.Media.ID)
		}
	}
	if len(ids) > 0 {
		_ = s.recCache.Save(ctx, &pid, "for-you", ids, time.Now().Add(6*time.Hour))
	}
	return items, nil
}

func (s *Service) fetchGuessByIDs(ctx context.Context, ids []uuid.UUID) ([]Item, error) {
	medias, err := s.fetchMediaByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Item, len(medias))
	for i := range medias {
		m := medias[i]
		out[i] = Item{Media: &m}
	}
	return out, nil
}

// fetchMediaByIDs 按 ID 顺序获取媒资
func (s *Service) fetchMediaByIDs(ctx context.Context, ids []uuid.UUID) ([]media.Media, error) {
	out := make([]media.Media, 0, len(ids))
	for _, id := range ids {
		m, err := s.engine.mediaRepo.GetByID(ctx, id.String())
		if err != nil {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func ptrF(v float64) *float64 { return &v }
