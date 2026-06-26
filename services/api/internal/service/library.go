package service

import (
	"context"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/history"
	"github.com/mediahub/api/internal/repository"
)

// LibraryService 个人片库（想看/收藏/历史 语义别名）
type LibraryService struct {
	history *HistoryService
	media   *repository.MediaRepo
}

func NewLibraryService(h *HistoryService, media *repository.MediaRepo) *LibraryService {
	return &LibraryService{history: h, media: media}
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
		if f.MediaID != nil && f.MediaID.String() == mediaID {
			return false, nil
		}
	}
	return s.history.ToggleFavorite(ctx, profileID, mediaID, common.FavWant, nil)
}

func (s *LibraryService) RemoveWant(ctx context.Context, profileID, mediaID string) (bool, error) {
	return s.history.ToggleFavorite(ctx, profileID, mediaID, common.FavWant, nil)
}

func (s *LibraryService) AddWantTMDB(ctx context.Context, profileID string, req AddWantTMDBRequest) (bool, error) {
	if req.TMDBID <= 0 {
		return false, apperr.Validation(map[string]string{"tmdb_id": "invalid"})
	}
	if req.Title == "" {
		return false, apperr.Validation(map[string]string{"title": "required"})
	}
	return s.history.AddWantTMDB(ctx, profileID, req)
}

func (s *LibraryService) RemoveWantTMDB(ctx context.Context, profileID string, tmdbID int) error {
	return s.history.RemoveWantTMDB(ctx, profileID, tmdbID)
}

func (s *LibraryService) IsWantTMDB(ctx context.Context, profileID string, tmdbID int) (bool, error) {
	return s.history.IsWantTMDB(ctx, profileID, tmdbID)
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

// AdminWantItem CMS 想看列表项
type AdminWantItem struct {
	ID           string    `json:"id"`
	ProfileID    string    `json:"profile_id"`
	ProfileName  string    `json:"profile_name"`
	MediaID      *string   `json:"media_id,omitempty"`
	TMDBID       *int      `json:"tmdb_id,omitempty"`
	MediaType    string    `json:"media_type,omitempty"`
	Title        string    `json:"title"`
	Year         *int      `json:"year,omitempty"`
	PosterURL    string    `json:"poster_url,omitempty"`
	InLibrary    bool      `json:"in_library"`
	LocalMediaID *string   `json:"local_media_id,omitempty"`
	External     bool      `json:"external"`
	CreatedAt    time.Time `json:"created_at"`
}

// AdminWantToWatch 全部播放端 Profile 的想看列表
func (s *LibraryService) AdminWantToWatch(ctx context.Context, limit int) ([]AdminWantItem, error) {
	items, err := s.history.ListAllWants(ctx, limit)
	if err != nil {
		return nil, err
	}

	out := make([]AdminWantItem, 0, len(items))
	for _, f := range items {
		item := AdminWantItem{
			ID:        f.ID.String(),
			ProfileID: f.ProfileID.String(),
			CreatedAt: f.CreatedAt,
		}
		if f.Profile != nil {
			item.ProfileName = f.Profile.Name
		}
		if f.MediaID != nil {
			mid := f.MediaID.String()
			item.MediaID = &mid
			item.InLibrary = true
			item.LocalMediaID = &mid
			if f.Media != nil {
				item.Title = f.Media.Title
				item.Year = f.Media.Year
				item.MediaType = string(f.Media.Type)
				item.PosterURL = f.Media.PosterURL
			}
		} else if f.TMDBID != nil {
			item.TMDBID = f.TMDBID
			item.MediaType = f.MediaType
			item.Title = f.Title
			item.Year = f.Year
			item.PosterURL = f.PosterURL
			item.External = true
			if s.media != nil {
				if local, _ := s.media.GetByTMDBID(ctx, *f.TMDBID); local != nil {
					item.InLibrary = true
					lid := local.ID.String()
					item.LocalMediaID = &lid
				}
			}
		}
		if item.Title == "" {
			item.Title = "未知"
		}
		out = append(out, item)
	}
	return out, nil
}
