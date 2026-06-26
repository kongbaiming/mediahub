package repository

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/catalog"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// CatalogRepo 内容目录仓储
type CatalogRepo struct {
	db *gorm.DB
}

func NewCatalogRepo(db *gorm.DB) *CatalogRepo {
	return &CatalogRepo{db: db}
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Trim(slugRe.ReplaceAllString(s, "-"), "-")
}

// ---- 影人 ----

func (r *CatalogRepo) UpsertPersonByTMDB(ctx context.Context, p *catalog.Person) (*catalog.Person, error) {
	if p.TMDBPersonID == nil {
		return nil, apperr.Validation(map[string]string{"tmdb_person_id": "required"})
	}
	var existing catalog.Person
	err := r.db.WithContext(ctx).Where("tmdb_person_id = ?", *p.TMDBPersonID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "创建影人失败")
		}
		return p, nil
	}
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询影人失败")
	}
	existing.Name = p.Name
	if p.OriginalName != "" {
		existing.OriginalName = p.OriginalName
	}
	if p.ProfilePath != "" {
		existing.ProfilePath = p.ProfilePath
	}
	if p.KnownForDepartment != "" {
		existing.KnownForDepartment = p.KnownForDepartment
	}
	if p.Popularity > 0 {
		existing.Popularity = p.Popularity
	}
	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "更新影人失败")
	}
	return &existing, nil
}

func (r *CatalogRepo) ReplaceCredits(ctx context.Context, mediaID uuid.UUID, credits []catalog.MediaCredit) error {
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Delete(&catalog.MediaCredit{}).Error; err != nil {
		return apperr.Wrap(err, apperr.CodeInternal, "清除演职员失败")
	}
	if len(credits) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(credits, 100).Error
}

func (r *CatalogRepo) ListCredits(ctx context.Context, mediaID string, role string, limit int) ([]catalog.MediaCredit, error) {
	if limit <= 0 {
		limit = 50
	}
	q := r.db.WithContext(ctx).Preload("Person").Where("media_id = ?", mediaID)
	if role != "" {
		q = q.Where("role = ?", role)
	}
	var out []catalog.MediaCredit
	if err := q.Order("billing_order ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询演职员失败")
	}
	return out, nil
}

func (r *CatalogRepo) SearchPersons(ctx context.Context, q string, limit int) ([]catalog.Person, error) {
	if limit <= 0 {
		limit = 20
	}
	var out []catalog.Person
	err := r.db.WithContext(ctx).
		Where("name ILIKE ? OR original_name ILIKE ?", "%"+q+"%", "%"+q+"%").
		Order("popularity DESC").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "搜索影人失败")
	}
	return out, nil
}

func (r *CatalogRepo) GetPerson(ctx context.Context, id string) (*catalog.Person, error) {
	var p catalog.Person
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("影人不存在")
		}
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询影人失败")
	}
	return &p, nil
}

func (r *CatalogRepo) ListWorksByPerson(ctx context.Context, personID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 40
	}
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&catalog.MediaCredit{}).
		Where("person_id = ?", personID).
		Distinct("media_id").
		Pluck("media_id", &ids).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询影人作品失败")
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var items []media.Media
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Limit(limit).Find(&items).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询作品失败")
	}
	return items, nil
}

// ---- 分类 ----

func (r *CatalogRepo) EnsureCategory(ctx context.Context, name, kind string, tmdbID *int) (uuid.UUID, error) {
	slug := Slugify(name)
	if slug == "" {
		slug = "unknown"
	}
	var c catalog.Category
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c = catalog.Category{Name: name, Slug: slug, Kind: kind, TMDBGenreID: tmdbID}
		if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
			return uuid.Nil, err
		}
		return c.ID, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	return c.ID, nil
}

func (r *CatalogRepo) SyncMediaCategories(ctx context.Context, mediaID uuid.UUID, genreNames []string) error {
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Delete(&catalog.MediaCategory{}).Error; err != nil {
		return err
	}
	for _, name := range genreNames {
		if name == "" {
			continue
		}
		cid, err := r.EnsureCategory(ctx, name, "genre", nil)
		if err != nil {
			continue
		}
		_ = r.db.WithContext(ctx).Create(&catalog.MediaCategory{MediaID: mediaID, CategoryID: cid}).Error
	}
	return nil
}

func (r *CatalogRepo) ListCategories(ctx context.Context, kind string) ([]catalog.Category, error) {
	q := r.db.WithContext(ctx).Order("sort_order ASC, name ASC")
	if kind != "" {
		q = q.Where("kind = ?", kind)
	}
	var out []catalog.Category
	if err := q.Find(&out).Error; err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInternal, "查询分类失败")
	}
	return out, nil
}

func (r *CatalogRepo) ListMediaByCategorySlug(ctx context.Context, slug string, limit, offset int, excludeAdult bool) ([]media.Media, int64, error) {
	var cat catalog.Category
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&cat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, apperr.NotFound("分类不存在")
		}
		return nil, 0, err
	}
	q := r.db.WithContext(ctx).
		Joins("JOIN media_categories mc ON mc.media_id = media.id").
		Where("mc.category_id = ?", cat.ID)
	if excludeAdult {
		q = q.Where("media.is_adult = ?", false)
	}
	var total int64
	if err := q.Model(&media.Media{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []media.Media
	if err := q.Order("media.rating DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ---- 标签 ----

func (r *CatalogRepo) EnsureTag(ctx context.Context, name, source string) (uuid.UUID, error) {
	slug := Slugify(name)
	if slug == "" {
		return uuid.Nil, nil
	}
	var t catalog.Tag
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t = catalog.Tag{Name: name, Slug: slug}
		if err := r.db.WithContext(ctx).Create(&t).Error; err != nil {
			return uuid.Nil, err
		}
	} else if err != nil {
		return uuid.Nil, err
	}
	return t.ID, nil
}

func (r *CatalogRepo) LinkMediaTag(ctx context.Context, mediaID, tagID uuid.UUID, source string) error {
	mt := catalog.MediaTag{MediaID: mediaID, TagID: tagID, Source: source}
	return r.db.WithContext(ctx).
		Where("media_id = ? AND tag_id = ?", mediaID, tagID).
		FirstOrCreate(&mt).Error
}

func (r *CatalogRepo) SyncMediaTags(ctx context.Context, mediaID uuid.UUID, names []string, source string) error {
	for _, name := range names {
		tid, err := r.EnsureTag(ctx, name, source)
		if err != nil || tid == uuid.Nil {
			continue
		}
		_ = r.LinkMediaTag(ctx, mediaID, tid, source)
	}
	return nil
}

func (r *CatalogRepo) ListMediaByTagSlug(ctx context.Context, slug string, limit int, excludeAdult bool) ([]media.Media, error) {
	var tag catalog.Tag
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("标签不存在")
		}
		return nil, err
	}
	q := r.db.WithContext(ctx).
		Joins("JOIN media_tags mt ON mt.media_id = media.id").
		Where("mt.tag_id = ?", tag.ID)
	if excludeAdult {
		q = q.Where("media.is_adult = ?", false)
	}
	var items []media.Media
	if err := q.Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ---- 专辑 ----

func (r *CatalogRepo) ListAlbums(ctx context.Context, limit int) ([]catalog.Album, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []catalog.Album
	if err := r.db.WithContext(ctx).Order("sort_order ASC, title ASC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *CatalogRepo) GetAlbum(ctx context.Context, id string) (*catalog.Album, error) {
	var a catalog.Album
	if err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("专辑不存在")
		}
		return nil, err
	}
	return &a, nil
}

func (r *CatalogRepo) ListAlbumMedia(ctx context.Context, albumID string, limit int) ([]media.Media, error) {
	if limit <= 0 {
		limit = 100
	}
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&catalog.AlbumItem{}).
		Where("album_id = ?", albumID).
		Order("sort_order ASC").
		Pluck("media_id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var items []media.Media
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Limit(limit).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CatalogRepo) CreateAlbum(ctx context.Context, a *catalog.Album) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *CatalogRepo) SetAlbumItems(ctx context.Context, albumID uuid.UUID, mediaIDs []uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("album_id = ?", albumID).Delete(&catalog.AlbumItem{}).Error; err != nil {
		return err
	}
	for i, mid := range mediaIDs {
		if err := r.db.WithContext(ctx).Create(&catalog.AlbumItem{
			AlbumID: albumID, MediaID: mid, SortOrder: i,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---- OTT 扩展 ----

func (r *CatalogRepo) ReplaceContentRatings(ctx context.Context, mediaID uuid.UUID, ratings []catalog.ContentRating) error {
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Delete(&catalog.ContentRating{}).Error; err != nil {
		return err
	}
	if len(ratings) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(ratings, 50).Error
}

func (r *CatalogRepo) ListContentRatings(ctx context.Context, mediaID string) ([]catalog.ContentRating, error) {
	var out []catalog.ContentRating
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *CatalogRepo) ReplaceExtras(ctx context.Context, mediaID uuid.UUID, extras []catalog.MediaExtra) error {
	if err := r.db.WithContext(ctx).Where("media_id = ?", mediaID).Delete(&catalog.MediaExtra{}).Error; err != nil {
		return err
	}
	if len(extras) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(extras, 50).Error
}

func (r *CatalogRepo) ListExtras(ctx context.Context, mediaID, extraType string) ([]catalog.MediaExtra, error) {
	q := r.db.WithContext(ctx).Where("media_id = ?", mediaID)
	if extraType != "" {
		q = q.Where("extra_type = ?", extraType)
	}
	var out []catalog.MediaExtra
	if err := q.Order("created_at ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *CatalogRepo) UpsertArtworks(ctx context.Context, mediaID uuid.UUID, poster, backdrop string) error {
	if poster != "" {
		_ = r.db.WithContext(ctx).Where("media_id = ? AND art_type = 'poster'", mediaID).Delete(&catalog.MediaArtwork{}).Error
		_ = r.db.WithContext(ctx).Create(&catalog.MediaArtwork{MediaID: mediaID, ArtType: "poster", URL: poster}).Error
	}
	if backdrop != "" {
		_ = r.db.WithContext(ctx).Where("media_id = ? AND art_type = 'backdrop'", mediaID).Delete(&catalog.MediaArtwork{}).Error
		_ = r.db.WithContext(ctx).Create(&catalog.MediaArtwork{MediaID: mediaID, ArtType: "backdrop", URL: backdrop}).Error
	}
	return nil
}

func (r *CatalogRepo) ListSubtitleTracks(ctx context.Context, mediaID string, episodeID *string) ([]catalog.SubtitleTrack, error) {
	q := r.db.WithContext(ctx).Where("media_id = ?", mediaID)
	if episodeID != nil && *episodeID != "" {
		q = q.Where("episode_id = ? OR episode_id IS NULL", *episodeID)
	}
	var out []catalog.SubtitleTrack
	if err := q.Order("is_default DESC, language ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *CatalogRepo) RegisterSubtitleTrack(ctx context.Context, track *catalog.SubtitleTrack) error {
	return r.db.WithContext(ctx).Create(track).Error
}

func (r *CatalogRepo) GetProfilePolicy(ctx context.Context, profileID string) (*catalog.ProfileContentPolicy, error) {
	var p catalog.ProfileContentPolicy
	err := r.db.WithContext(ctx).First(&p, "profile_id = ?", profileID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &catalog.ProfileContentPolicy{ProfileID: uuid.MustParse(profileID), MaxRatingLevel: 100}, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *CatalogRepo) UpsertProfilePolicy(ctx context.Context, p *catalog.ProfileContentPolicy) error {
	return r.db.WithContext(ctx).Save(p).Error
}

// RefreshAvailability 刷新作品可播状态
func (r *CatalogRepo) RefreshAvailability(ctx context.Context, mediaID uuid.UUID) error {
	var m media.Media
	if err := r.db.WithContext(ctx).First(&m, "id = ?", mediaID).Error; err != nil {
		return err
	}

	status := common.AvailProcessing
	var availableAt *time.Time

	hasFile := false
	if m.IsTV() {
		var n int64
		r.db.WithContext(ctx).Model(&media.Episode{}).
			Where("media_id = ? AND file_path <> ''", mediaID).Count(&n)
		if n == 0 {
			r.db.WithContext(ctx).Model(&media.MediaFile{}).
				Where("media_id = ? AND episode_id IS NOT NULL", mediaID).Count(&n)
		}
		hasFile = n > 0
	} else {
		var f media.MediaFile
		err := r.db.WithContext(ctx).
			Where("media_id = ? AND (episode_id IS NULL OR episode_id = '00000000-0000-0000-0000-000000000000')", mediaID).
			Order("is_primary DESC, height DESC").First(&f).Error
		if err == nil && f.Path != "" {
			if _, statErr := os.Stat(f.Path); statErr == nil {
				hasFile = true
			}
		}
		if !hasFile && m.StoragePath != "" {
			if _, statErr := os.Stat(m.StoragePath); statErr == nil {
				hasFile = true
			}
		}
	}

	switch {
	case hasFile:
		status = common.AvailAvailable
		now := time.Now()
		availableAt = &now
	case m.ScrapeStatus == common.ScrapeStatusDone:
		status = common.AvailMissing
	case m.ScrapeStatus == common.ScrapeStatusPending || m.ScrapeStatus == common.ScrapeStatusScraping:
		status = common.AvailProcessing
	default:
		status = common.AvailMissing
	}

	return r.db.WithContext(ctx).Model(&m).Updates(map[string]any{
		"availability_status": status,
		"available_at":        availableAt,
	}).Error
}

// BackfillCategoriesFromGenres 从 media.genres 数组回填分类关联
func (r *CatalogRepo) BackfillCategoriesFromGenres(ctx context.Context) (int, error) {
	var items []media.Media
	if err := r.db.WithContext(ctx).Find(&items).Error; err != nil {
		return 0, err
	}
	n := 0
	for _, m := range items {
		if len(m.Genres) == 0 {
			continue
		}
		if err := r.SyncMediaCategories(ctx, m.ID, m.Genres); err == nil {
			n++
		}
	}
	return n, nil
}
