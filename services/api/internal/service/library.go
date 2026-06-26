package service

import (
	"context"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/history"
)

// LibraryService 个人片库（想看/收藏/历史 语义别名）
type LibraryService struct {
	history *HistoryService
}

func NewLibraryService(h *HistoryService) *LibraryService {
	return &LibraryService{history: h}
}

func (s *LibraryService) WantToWatch(ctx context.Context, profileID string) ([]history.Favorite, error) {
	return s.history.ListFavorites(ctx, profileID, string(common.FavWant))
}

func (s *LibraryService) Favorites(ctx context.Context, profileID string) ([]history.Favorite, error) {
	return s.history.ListFavorites(ctx, profileID, string(common.FavLiked))
}

func (s *LibraryService) Watching(ctx context.Context, profileID string) ([]history.Favorite, error) {
	return s.history.ListFavorites(ctx, profileID, string(common.FavWatching))
}

func (s *LibraryService) Watched(ctx context.Context, profileID string) ([]history.Favorite, error) {
	return s.history.ListFavorites(ctx, profileID, string(common.FavWatched))
}

func (s *LibraryService) AddWant(ctx context.Context, profileID, mediaID string) (bool, error) {
	items, err := s.history.ListFavorites(ctx, profileID, string(common.FavWant))
	if err != nil {
		return false, err
	}
	for _, f := range items {
		if f.MediaID.String() == mediaID {
			return false, nil
		}
	}
	return s.history.ToggleFavorite(ctx, profileID, mediaID, common.FavWant, nil)
}

func (s *LibraryService) RemoveWant(ctx context.Context, profileID, mediaID string) (bool, error) {
	return s.history.ToggleFavorite(ctx, profileID, mediaID, common.FavWant, nil)
}

func (s *LibraryService) ToggleFavorite(ctx context.Context, profileID, mediaID string) (bool, error) {
	return s.history.ToggleFavorite(ctx, profileID, mediaID, common.FavLiked, nil)
}

func (s *LibraryService) History(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	return s.history.GetHistory(ctx, profileID, limit)
}

func (s *LibraryService) ContinueWatching(ctx context.Context, profileID string, limit int) ([]history.History, error) {
	return s.history.GetContinueWatching(ctx, profileID, limit)
}
