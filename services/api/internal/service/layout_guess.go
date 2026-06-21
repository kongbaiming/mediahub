package service

import (
	"context"
	"strings"

	"github.com/mediahub/api/internal/domain/layout"
)

type guessRowSpec struct {
	rowID        string
	cardStyle    string
	subtitle     string
	limit        int
	discoverLimit int
}

var (
	webGuessSpec = guessRowSpec{
		rowID:         "guess-you-like-web",
		cardStyle:     "poster",
		subtitle:      "根据你的观影习惯推荐",
		limit:         20,
		discoverLimit: 6,
	}
	tvGuessSpec = guessRowSpec{
		rowID:         "guess-you-like-tv",
		cardStyle:     "landscape",
		subtitle:      "根据观影习惯智能推荐",
		limit:         16,
		discoverLimit: 4,
	}
)

// EnsureGuessYouLikeRows 为首页布局补全「猜你喜欢」行（幂等，启动时调用）
func (s *LayoutService) EnsureGuessYouLikeRows(ctx context.Context) error {
	known := map[string]guessRowSpec{
		"d7cedd8e-5598-4f9b-bc82-6dc1a954b362": webGuessSpec,
		"ed9e7c99-5260-4fe4-960f-eabe764ba145": tvGuessSpec,
	}
	for id, spec := range known {
		if err := s.ensureGuessRowForLayout(ctx, id, spec); err != nil {
			return err
		}
	}

	all, err := s.repo.List(ctx, nil, "")
	if err != nil {
		return err
	}
	for _, l := range all {
		if layoutHasGuessYouLike(l) {
			continue
		}
		id := l.ID.String()
		if _, done := known[id]; done {
			continue
		}
		if layoutHasRowID(l, "hero-web") {
			if err := s.ensureGuessRowForLayout(ctx, id, webGuessSpec); err != nil {
				return err
			}
		} else if layoutHasRowID(l, "hero-tv") {
			if err := s.ensureGuessRowForLayout(ctx, id, tvGuessSpec); err != nil {
				return err
			}
		}
	}
	// 启动时强制刷新首页 Feed 缓存（避免布局已改但 Redis 仍是旧数据）
	s.invalidateHomeFeeds(ctx)
	return nil
}

func (s *LayoutService) ensureGuessRowForLayout(ctx context.Context, layoutID string, spec guessRowSpec) error {
	l, err := s.repo.GetByID(ctx, layoutID)
	if err != nil {
		return nil
	}
	if layoutHasGuessYouLike(*l) {
		return nil
	}

	visible := true
	newRow := layout.Row{
		ID:        spec.rowID,
		Type:      "shelf",
		Title:     "猜你喜欢",
		Subtitle:  spec.subtitle,
		CardStyle: spec.cardStyle,
		Visible:   &visible,
		Source: layout.DataSource{
			Type: "guess-you-like",
			Params: map[string]any{
				"limit":          spec.limit,
				"discover_limit": spec.discoverLimit,
			},
		},
	}

	l.Config.Rows = insertGuessYouLikeRow(l.Config.Rows, newRow)
	l.Version++
	if err := s.repo.Update(ctx, l); err != nil {
		return err
	}
	s.invalidateFeedsForLayout(ctx, layoutID)
	return nil
}

func layoutHasGuessYouLike(l layout.Layout) bool {
	for _, r := range l.Config.Rows {
		if r.Source.Type == "guess-you-like" || strings.HasPrefix(r.ID, "guess-you-like") {
			return true
		}
	}
	return false
}

func layoutHasRowID(l layout.Layout, rowID string) bool {
	for _, r := range l.Config.Rows {
		if r.ID == rowID {
			return true
		}
	}
	return false
}

func insertGuessYouLikeRow(rows []layout.Row, newRow layout.Row) []layout.Row {
	for i, r := range rows {
		if strings.HasPrefix(r.ID, "continue-") || r.Source.Type == "continue-watching" {
			out := make([]layout.Row, 0, len(rows)+1)
			out = append(out, rows[:i+1]...)
			out = append(out, newRow)
			out = append(out, rows[i+1:]...)
			return out
		}
	}
	if len(rows) > 0 && rows[0].Type == "hero-banner" {
		out := make([]layout.Row, 0, len(rows)+1)
		out = append(out, rows[0], newRow)
		out = append(out, rows[1:]...)
		return out
	}
	return append([]layout.Row{newRow}, rows...)
}
