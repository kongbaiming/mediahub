package handler

import (
	"strconv"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// CatalogHandler 内容目录 HTTP
type CatalogHandler struct {
	svc *service.CatalogService
}

func NewCatalogHandler(svc *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) Credits(c *gin.Context) {
	items, err := h.svc.ListCredits(c.Request.Context(), c.Param("id"), c.Query("role"), c.Query("episode_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) Extras(c *gin.Context) {
	items, err := h.svc.ListExtras(c.Request.Context(), c.Param("id"), c.Query("type"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) Ratings(c *gin.Context) {
	items, err := h.svc.ListRatings(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) Subtitles(c *gin.Context) {
	items, err := h.svc.ListSubtitles(c.Request.Context(), c.Param("id"), c.Query("episode_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) NextEpisode(c *gin.Context) {
	ep, err := h.svc.NextEpisode(c.Request.Context(), c.Param("id"), c.Query("after_episode_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	if ep == nil {
		c.JSON(200, gin.H{"data": nil})
		return
	}
	c.JSON(200, gin.H{"data": ep})
}

func (h *CatalogHandler) Availability(c *gin.Context) {
	if err := h.svc.RefreshAvailability(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "refreshed"})
}

func (h *CatalogHandler) ListPersons(c *gin.Context) {
	items, err := h.svc.SearchPersons(c.Request.Context(), c.Query("q"), atoi(c.Query("limit"), 20))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) GetPerson(c *gin.Context) {
	p, err := h.svc.GetPerson(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": p})
}

func (h *CatalogHandler) GetPersonByTMDB(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdb_id"))
	if err != nil || tmdbID <= 0 {
		respondError(c, apperr.Validation(map[string]string{"tmdb_id": "invalid"}))
		return
	}
	p, err := h.svc.EnsurePersonByTMDB(c.Request.Context(), tmdbID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": p})
}

func (h *CatalogHandler) TMDBMediaDetail(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdb_id"))
	if err != nil || tmdbID <= 0 {
		respondError(c, apperr.Validation(map[string]string{"tmdb_id": "invalid"}))
		return
	}
	detail, err := h.svc.TMDBMediaDetail(c.Request.Context(), c.Param("type"), tmdbID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": detail})
}

func (h *CatalogHandler) TMDBSimilar(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("tmdb_id"))
	if err != nil || tmdbID <= 0 {
		respondError(c, apperr.Validation(map[string]string{"tmdb_id": "invalid"}))
		return
	}
	items, err := h.svc.TMDBSimilar(c.Request.Context(), c.Param("type"), tmdbID, atoi(c.Query("limit"), 12))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) Collection(c *gin.Context) {
	items, err := h.svc.Collection(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) PersonWorks(c *gin.Context) {
	items, err := h.svc.PersonWorks(c.Request.Context(), c.Param("id"), c.Query("exclude_media_id"), atoi(c.Query("limit"), 40))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) ListCategories(c *gin.Context) {
	items, err := h.svc.ListCategories(c.Request.Context(), c.Query("kind"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) CategoryWorks(c *gin.Context) {
	p := common.Pagination{Page: atoi(c.Query("page"), 1), PageSize: atoi(c.Query("page_size"), 20)}
	isKid := false
	if pid := middleware.GetProfileID(c); pid != "" {
		// kid filter via feed pattern omitted; use query
	}
	result, err := h.svc.CategoryWorks(c.Request.Context(), c.Param("slug"), p, isKid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, result)
}

func (h *CatalogHandler) TagWorks(c *gin.Context) {
	items, err := h.svc.TagWorks(c.Request.Context(), c.Param("slug"), atoi(c.Query("limit"), 40), false)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) ListAlbums(c *gin.Context) {
	items, err := h.svc.ListAlbums(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) AlbumWorks(c *gin.Context) {
	items, err := h.svc.AlbumWorks(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *CatalogHandler) CreateAlbum(c *gin.Context) {
	var req struct {
		Title    string   `json:"title" binding:"required"`
		Overview string   `json:"overview"`
		MediaIDs []string `json:"media_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(err.Error()))
		return
	}
	a, err := h.svc.CreateAlbum(c.Request.Context(), req.Title, req.Overview, req.MediaIDs)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, gin.H{"data": a})
}
