package service

import (
	"context"
	"strconv"
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

// PersonWorkItem 影人参演作品（库内可播或 TMDB 库外参考）
type PersonWorkItem struct {
	MediaSummary
	External bool `json:"external,omitempty"`
	TMDBID   int  `json:"tmdb_id,omitempty"`
}

func (s *CatalogService) PersonWorks(ctx context.Context, personID, excludeMediaID string, limit int) ([]PersonWorkItem, error) {
	if limit <= 0 {
		limit = 40
	}
	localItems, err := s.catalog.ListWorksByPerson(ctx, personID, limit*2)
	if err != nil {
		return nil, err
	}
	localByTMDB := map[int]*media.Media{}
	for i := range localItems {
		m := &localItems[i]
		if m.TMDBID != nil && *m.TMDBID > 0 {
			localByTMDB[*m.TMDBID] = m
		}
	}

	var exclude uuid.UUID
	if excludeMediaID != "" {
		exclude, _ = uuid.Parse(excludeMediaID)
	}

	seen := map[string]bool{}
	var out []PersonWorkItem
	add := func(item PersonWorkItem) {
		if exclude != uuid.Nil && item.ID == exclude {
			return
		}
		key := item.ID.String()
		if item.External || item.ID == uuid.Nil {
			key = "tmdb:" + strconv.Itoa(item.TMDBID)
		}
		if key == "" || key == "tmdb:0" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, item)
	}

	p, _ := s.catalog.GetPerson(ctx, personID)
	if p != nil && p.TMDBPersonID != nil && *p.TMDBPersonID > 0 && s.tmdb != nil {
		if cc, err := s.tmdb.GetPersonCombinedCredits(ctx, *p.TMDBPersonID); err == nil && cc != nil {
			for _, c := range cc.Cast {
				if len(out) >= limit {
					break
				}
				if c.ID <= 0 {
					continue
				}
				if lm := localByTMDB[c.ID]; lm != nil {
					item := PersonWorkItem{MediaSummary: toSummary(lm)}
					if lm.TMDBID != nil {
						item.TMDBID = *lm.TMDBID
					}
					add(item)
					continue
				}
				title := c.Title
				if title == "" {
					title = c.Name
				}
				if title == "" {
					continue
				}
				mt := common.MediaTypeMovie
				if c.MediaType == "tv" {
					mt = common.MediaTypeTVShow
				}
				item := PersonWorkItem{
					MediaSummary: MediaSummary{
						Title:     title,
						Year:      parseCreditYear(c.ReleaseDate, c.FirstAirDate),
						Type:      mt,
						Rating:    c.VoteAverage,
						PosterURL: s.tmdb.PosterURL(c.PosterPath, "w500"),
					},
					External: true,
					TMDBID:   c.ID,
				}
				add(item)
			}
		}
	}

	for i := range localItems {
		if len(out) >= limit {
			break
		}
		m := &localItems[i]
		item := PersonWorkItem{MediaSummary: toSummary(m)}
		if m.TMDBID != nil {
			item.TMDBID = *m.TMDBID
		}
		add(item)
	}

	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func parseCreditYear(releaseDate, firstAirDate string) *int {
	for _, d := range []string{releaseDate, firstAirDate} {
		if len(d) >= 4 {
			if y, err := strconv.Atoi(d[:4]); err == nil && y > 1800 {
				return &y
			}
		}
	}
	return nil
}

// TMDBCastPersonDTO 演职员（含 TMDB ID，便于库外详情跳转影人页）
type TMDBCastPersonDTO struct {
	ID           string `json:"id,omitempty"`
	TMDBPersonID int    `json:"tmdb_person_id,omitempty"`
	Name         string `json:"name"`
	ProfileURL   string `json:"profile_url,omitempty"`
}

// TMDBCastCreditDTO 库外详情演职员
type TMDBCastCreditDTO struct {
	ID            string            `json:"id"`
	CharacterName string            `json:"character_name,omitempty"`
	Person        TMDBCastPersonDTO `json:"person"`
}

// TMDBMediaDetailDTO TMDB 库外作品详情
type TMDBMediaDetailDTO struct {
	External     bool                `json:"external"`
	TMDBID       int                 `json:"tmdb_id"`
	LocalMediaID *uuid.UUID          `json:"local_media_id,omitempty"`
	Type         common.MediaType    `json:"type"`
	Title        string              `json:"title"`
	OriginalTitle string             `json:"original_title,omitempty"`
	Year         *int                `json:"year,omitempty"`
	Overview     string              `json:"overview,omitempty"`
	PosterURL    string              `json:"poster_url,omitempty"`
	BackdropURL  string              `json:"backdrop_url,omitempty"`
	Rating       float64             `json:"rating"`
	Runtime      int                 `json:"runtime,omitempty"`
	Genres       []string            `json:"genres"`
	Credits      []TMDBCastCreditDTO `json:"credits,omitempty"`
}

func (s *CatalogService) EnsurePersonByTMDB(ctx context.Context, tmdbPersonID int) (*catalog.Person, error) {
	if tmdbPersonID <= 0 {
		return nil, apperr.Validation(map[string]string{"tmdb_person_id": "invalid"})
	}
	if p, err := s.catalog.GetPersonByTMDB(ctx, tmdbPersonID); err == nil {
		s.enrichPerson(p)
		s.refreshPersonBio(ctx, p)
		return p, nil
	}
	p, err := s.upsertPersonFromTMDB(ctx, tmdbPersonID, "", "", "")
	if err != nil {
		return nil, err
	}
	s.enrichPerson(p)
	return p, nil
}

func (s *CatalogService) TMDBMediaDetail(ctx context.Context, mediaType string, tmdbID int) (*TMDBMediaDetailDTO, error) {
	if tmdbID <= 0 {
		return nil, apperr.Validation(map[string]string{"tmdb_id": "invalid"})
	}
	if local, _ := s.media.GetByTMDBID(ctx, tmdbID); local != nil {
		return &TMDBMediaDetailDTO{LocalMediaID: &local.ID}, nil
	}
	if s.tmdb == nil {
		return nil, apperr.ExternalAPI(nil, "TMDB 未配置")
	}

	out := &TMDBMediaDetailDTO{
		External: true,
		TMDBID:   tmdbID,
		Genres:   []string{},
	}

	switch mediaType {
	case "movie":
		out.Type = common.MediaTypeMovie
		m, err := s.tmdb.GetMovie(ctx, tmdbID)
		if err != nil {
			return nil, apperr.ExternalAPI(err, "拉取 TMDB 电影详情失败")
		}
		out.Title = m.Title
		out.OriginalTitle = m.OriginalTitle
		out.Overview = m.Overview
		out.Rating = m.VoteAverage
		out.Runtime = m.Runtime
		out.Year = parseCreditYear(m.ReleaseDate, "")
		out.PosterURL = s.tmdb.PosterURL(m.PosterPath, "w500")
		out.BackdropURL = s.tmdb.PosterURL(m.BackdropPath, "w1280")
		for _, g := range m.Genres {
			out.Genres = append(out.Genres, g.Name)
		}
		if cr, err := s.tmdb.GetMovieCredits(ctx, tmdbID); err == nil && cr != nil {
			out.Credits = s.tmdbCastCredits(ctx, cr.Cast)
		}
	case "tvshow", "tv":
		out.Type = common.MediaTypeTVShow
		tv, err := s.tmdb.GetTVShow(ctx, tmdbID)
		if err != nil {
			return nil, apperr.ExternalAPI(err, "拉取 TMDB 剧集详情失败")
		}
		out.Title = tv.Name
		out.OriginalTitle = tv.OriginalName
		out.Overview = tv.Overview
		out.Rating = tv.VoteAverage
		if len(tv.EpisodeRunTime) > 0 {
			out.Runtime = tv.EpisodeRunTime[0]
		}
		out.Year = parseCreditYear("", tv.FirstAirDate)
		out.PosterURL = s.tmdb.PosterURL(tv.PosterPath, "w500")
		out.BackdropURL = s.tmdb.PosterURL(tv.BackdropPath, "w1280")
		for _, g := range tv.Genres {
			out.Genres = append(out.Genres, g.Name)
		}
		if cr, err := s.tmdb.GetTVCredits(ctx, tmdbID); err == nil && cr != nil {
			out.Credits = s.tmdbCastCredits(ctx, cr.Cast)
		}
	default:
		return nil, apperr.Validation(map[string]string{"type": "unsupported media type"})
	}

	if out.Title == "" {
		return nil, apperr.NotFound("TMDB 作品不存在")
	}
	return out, nil
}

func (s *CatalogService) tmdbCastCredits(ctx context.Context, cast []scraper.TMDBCastMember) []TMDBCastCreditDTO {
	var out []TMDBCastCreditDTO
	for i, c := range cast {
		if i >= 20 {
			break
		}
		person := TMDBCastPersonDTO{
			TMDBPersonID: c.ID,
			Name:         c.Name,
			ProfileURL:   s.tmdb.PosterURL(c.ProfilePath, "w185"),
		}
		if c.ID > 0 {
			if p, err := s.catalog.GetPersonByTMDB(ctx, c.ID); err == nil {
				person.ID = p.ID.String()
			}
		}
		out = append(out, TMDBCastCreditDTO{
			ID:            "tmdb-cast-" + strconv.Itoa(c.ID),
			CharacterName: c.Character,
			Person:        person,
		})
	}
	return out
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
