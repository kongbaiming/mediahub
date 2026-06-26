package service

import (
	"context"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/catalog"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/internal/repository"
	"github.com/mediahub/api/internal/scraper"

	"github.com/google/uuid"
)

// CatalogService 内容目录业务
type CatalogService struct {
	catalog *repository.CatalogRepo
	media   *repository.MediaRepo
	tmdb    *scraper.TMDBClient
}

func NewCatalogService(catalog *repository.CatalogRepo, media *repository.MediaRepo, tmdb *scraper.TMDBClient) *CatalogService {
	return &CatalogService{catalog: catalog, media: media, tmdb: tmdb}
}

func (s *CatalogService) ListCredits(ctx context.Context, mediaID, role string) ([]catalog.MediaCredit, error) {
	items, err := s.catalog.ListCredits(ctx, mediaID, role, 80)
	if err != nil {
		return nil, err
	}
	for i := range items {
		s.enrichPerson(items[i].Person)
	}
	return items, nil
}

func (s *CatalogService) enrichPerson(p *catalog.Person) {
	if p == nil {
		return
	}
	if s.tmdb != nil && p.ProfilePath != "" {
		p.ProfileURL = s.tmdb.PosterURL(p.ProfilePath, "w185")
	}
}

func (s *CatalogService) GetPerson(ctx context.Context, id string) (*catalog.Person, error) {
	p, err := s.catalog.GetPerson(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichPerson(p)
	s.refreshPersonBio(ctx, p)
	return p, nil
}

func (s *CatalogService) refreshPersonBio(ctx context.Context, p *catalog.Person) {
	if p == nil || strings.TrimSpace(p.Biography) != "" || p.TMDBPersonID == nil || s.tmdb == nil {
		return
	}
	rp, err := s.tmdb.GetPersonRich(ctx, *p.TMDBPersonID)
	if err != nil || rp == nil || strings.TrimSpace(rp.Biography) == "" {
		return
	}
	p.Biography = strings.TrimSpace(rp.Biography)
	if p.PlaceOfBirth == "" {
		p.PlaceOfBirth = strings.TrimSpace(rp.PlaceOfBirth)
	}
	_ = s.catalog.PatchPersonProfile(ctx, p.ID, p.Biography, p.PlaceOfBirth)
}

func (s *CatalogService) PersonWorks(ctx context.Context, personID string, limit int) ([]MediaSummary, error) {
	if limit <= 0 {
		limit = 40
	}
	items, err := s.catalog.ListWorksByPerson(ctx, personID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]MediaSummary, len(items))
	for i := range items {
		out[i] = toSummary(&items[i])
	}
	return out, nil
}

func (s *CatalogService) ListCategories(ctx context.Context, kind string) ([]catalog.Category, error) {
	return s.catalog.ListCategories(ctx, kind)
}

func (s *CatalogService) CategoryWorks(ctx context.Context, slug string, p common.Pagination, excludeAdult bool) (*ListDTO, error) {
	p.Normalize()
	items, total, err := s.catalog.ListMediaByCategorySlug(ctx, slug, p.PageSize, p.Offset(), excludeAdult)
	if err != nil {
		return nil, err
	}
	out := make([]MediaSummary, len(items))
	for i := range items {
		out[i] = toSummary(&items[i])
	}
	return &ListDTO{Items: out, Total: total, Page: p.Page, Size: p.PageSize}, nil
}

func (s *CatalogService) TagWorks(ctx context.Context, slug string, limit int, excludeAdult bool) ([]MediaSummary, error) {
	items, err := s.catalog.ListMediaByTagSlug(ctx, slug, limit, excludeAdult)
	if err != nil {
		return nil, err
	}
	out := make([]MediaSummary, len(items))
	for i := range items {
		out[i] = toSummary(&items[i])
	}
	return out, nil
}

func (s *CatalogService) ListAlbums(ctx context.Context) ([]catalog.Album, error) {
	return s.catalog.ListAlbums(ctx, 100)
}

func (s *CatalogService) AlbumWorks(ctx context.Context, albumID string) ([]MediaSummary, error) {
	items, err := s.catalog.ListAlbumMedia(ctx, albumID, 200)
	if err != nil {
		return nil, err
	}
	out := make([]MediaSummary, len(items))
	for i := range items {
		out[i] = toSummary(&items[i])
	}
	return out, nil
}

func (s *CatalogService) ListExtras(ctx context.Context, mediaID, extraType string) ([]catalog.MediaExtra, error) {
	return s.catalog.ListExtras(ctx, mediaID, extraType)
}

func (s *CatalogService) ListRatings(ctx context.Context, mediaID string) ([]catalog.ContentRating, error) {
	return s.catalog.ListContentRatings(ctx, mediaID)
}

func (s *CatalogService) ListSubtitles(ctx context.Context, mediaID, episodeID string) ([]catalog.SubtitleTrack, error) {
	var ep *string
	if episodeID != "" {
		ep = &episodeID
	}
	return s.catalog.ListSubtitleTracks(ctx, mediaID, ep)
}

func (s *CatalogService) SearchPersons(ctx context.Context, q string, limit int) ([]catalog.Person, error) {
	if q == "" {
		return nil, apperr.BadRequest("missing q")
	}
	return s.catalog.SearchPersons(ctx, q, limit)
}

func (s *CatalogService) NextEpisode(ctx context.Context, mediaID, afterEpisodeID string) (*media.Episode, error) {
	if afterEpisodeID == "" {
		return nil, apperr.BadRequest("missing after_episode_id")
	}
	return s.media.NextEpisode(ctx, mediaID, afterEpisodeID)
}

func (s *CatalogService) RefreshAvailability(ctx context.Context, mediaID string) error {
	id, err := uuid.Parse(mediaID)
	if err != nil {
		return apperr.Validation(map[string]string{"media_id": "格式错误"})
	}
	return s.catalog.RefreshAvailability(ctx, id)
}

// EnrichFromTMDB 刮削后写入演职员/预告/分级/分类
func (s *CatalogService) EnrichFromTMDB(ctx context.Context, m *media.Media) error {
	if s.tmdb == nil || m.TMDBID == nil || *m.TMDBID <= 0 {
		return s.catalog.RefreshAvailability(ctx, m.ID)
	}
	tmdbID := *m.TMDBID

	if len(m.Genres) > 0 {
		_ = s.catalog.SyncMediaCategories(ctx, m.ID, m.Genres)
	}
	if len(m.Tags) > 0 {
		_ = s.catalog.SyncMediaTags(ctx, m.ID, m.Tags, "scanner")
	}
	_ = s.catalog.UpsertArtworks(ctx, m.ID, m.PosterURL, m.BackdropURL)

	if err := s.syncCredits(ctx, m, tmdbID); err != nil {
		// non-fatal
	}
	if err := s.syncExtras(ctx, m.ID, tmdbID, m.IsTV()); err != nil {
		// non-fatal
	}
	if err := s.syncRatings(ctx, m.ID, tmdbID, m.IsTV()); err != nil {
		// non-fatal
	}
	return s.catalog.RefreshAvailability(ctx, m.ID)
}

func (s *CatalogService) syncCredits(ctx context.Context, m *media.Media, tmdbID int) error {
	var cr *scraper.TMDBCredits
	var err error
	if m.IsTV() {
		cr, err = s.tmdb.GetTVCredits(ctx, tmdbID)
	} else {
		cr, err = s.tmdb.GetMovieCredits(ctx, tmdbID)
	}
	if err != nil || cr == nil {
		return err
	}
	var credits []catalog.MediaCredit
	for i, c := range cr.Cast {
		if i >= 30 {
			break
		}
		person, err := s.upsertPersonFromTMDB(ctx, c.ID, c.Name, c.ProfilePath, "")
		if err != nil || person == nil {
			continue
		}
		credits = append(credits, catalog.MediaCredit{
			MediaID: m.ID, PersonID: person.ID, Role: "actor",
			CharacterName: c.Character, BillingOrder: c.Order,
		})
	}
	for _, c := range cr.Crew {
		if c.Job != "Director" && c.Job != "Writer" && c.Department != "Writing" {
			continue
		}
		role := "crew"
		if c.Job == "Director" {
			role = "director"
		} else if c.Job == "Writer" || c.Department == "Writing" {
			role = "writer"
		}
		person, err := s.upsertPersonFromTMDB(ctx, c.ID, c.Name, c.ProfilePath, c.Department)
		if err != nil || person == nil {
			continue
		}
		credits = append(credits, catalog.MediaCredit{
			MediaID: m.ID, PersonID: person.ID, Role: role, BillingOrder: 100,
		})
	}
	return s.catalog.ReplaceCredits(ctx, m.ID, credits)
}

// upsertPersonFromTMDB 按 TMDB_LANGUAGE 拉取影人本地化姓名与简介后入库
func (s *CatalogService) upsertPersonFromTMDB(ctx context.Context, tmdbPersonID int, fallbackName, profilePath, department string) (*catalog.Person, error) {
	name := fallbackName
	originalName := ""
	biography := ""
	placeOfBirth := ""
	var birthday *time.Time
	gender := 0
	popularity := 0.0
	if s.tmdb != nil {
		if p, err := s.tmdb.GetPersonRich(ctx, tmdbPersonID); err == nil && p != nil {
			if p.Name != "" {
				name = p.Name
			}
			originalName = p.OriginalName
			if profilePath == "" && p.ProfilePath != "" {
				profilePath = p.ProfilePath
			}
			if department == "" && p.KnownForDepartment != "" {
				department = p.KnownForDepartment
			}
			biography = strings.TrimSpace(p.Biography)
			placeOfBirth = strings.TrimSpace(p.PlaceOfBirth)
			gender = p.Gender
			popularity = p.Popularity
			if p.Birthday != "" {
				if t, err := time.Parse("2006-01-02", p.Birthday); err == nil {
					birthday = &t
				}
			}
		}
	}
	pid := tmdbPersonID
	return s.catalog.UpsertPersonByTMDB(ctx, &catalog.Person{
		Name:               name,
		OriginalName:       originalName,
		TMDBPersonID:       &pid,
		ProfilePath:        profilePath,
		Biography:          biography,
		Birthday:           birthday,
		PlaceOfBirth:       placeOfBirth,
		Gender:             gender,
		KnownForDepartment: department,
		Popularity:         popularity,
	})
}

func (s *CatalogService) syncExtras(ctx context.Context, mediaID uuid.UUID, tmdbID int, isTV bool) error {
	var v *scraper.TMDBVideos
	var err error
	if isTV {
		v, err = s.tmdb.GetTVVideos(ctx, tmdbID)
	} else {
		v, err = s.tmdb.GetMovieVideos(ctx, tmdbID)
	}
	if err != nil || v == nil {
		return err
	}
	var extras []catalog.MediaExtra
	for _, vid := range v.Results {
		if vid.Site != "YouTube" {
			continue
		}
		et := "clip"
		switch vid.Type {
		case "Trailer":
			et = "trailer"
		case "Teaser":
			et = "teaser"
		case "Behind the Scenes":
			et = "behind_the_scenes"
		}
		extras = append(extras, catalog.MediaExtra{
			MediaID: mediaID, ExtraType: et, Title: vid.Name, Source: "tmdb",
			ExternalKey: vid.Key, ExternalURL: scraper.YouTubeURL(vid.Key),
		})
	}
	return s.catalog.ReplaceExtras(ctx, mediaID, extras)
}

func (s *CatalogService) syncRatings(ctx context.Context, mediaID uuid.UUID, tmdbID int, isTV bool) error {
	var ratings []catalog.ContentRating
	if isTV {
		cr, err := s.tmdb.GetTVContentRatings(ctx, tmdbID)
		if err != nil || cr == nil {
			return err
		}
		for _, r := range cr.Results {
			if r.Rating == "" {
				continue
			}
			ratings = append(ratings, catalog.ContentRating{
				MediaID: mediaID, Country: r.ISO31661, System: "tmdb", Rating: r.Rating,
			})
		}
	} else {
		rd, err := s.tmdb.GetMovieReleaseDates(ctx, tmdbID)
		if err != nil || rd == nil {
			return err
		}
		for _, r := range rd.Results {
			for _, d := range r.ReleaseDates {
				if d.Certification == "" {
					continue
				}
				ratings = append(ratings, catalog.ContentRating{
					MediaID: mediaID, Country: r.ISO31661, System: "tmdb", Rating: d.Certification,
				})
				break
			}
		}
	}
	return s.catalog.ReplaceContentRatings(ctx, mediaID, ratings)
}

func (s *CatalogService) AlbumMediaIDs(ctx context.Context, albumID string) ([]uuid.UUID, error) {
	items, err := s.catalog.ListAlbumMedia(ctx, albumID, 500)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	return ids, nil
}

func (s *CatalogService) CreateAlbum(ctx context.Context, title, overview string, mediaIDs []string) (*catalog.Album, error) {
	a := &catalog.Album{Title: title, Overview: overview, AlbumType: "collection"}
	if err := s.catalog.CreateAlbum(ctx, a); err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for _, s := range mediaIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if err := s.catalog.SetAlbumItems(ctx, a.ID, ids); err != nil {
		return nil, err
	}
	return a, nil
}
