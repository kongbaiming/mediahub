package handler

import (
	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/middleware"
	"github.com/mediahub/api/internal/service"

	"github.com/gin-gonic/gin"
)

// LibraryHandler 个人片库 HTTP
type LibraryHandler struct {
	svc *service.LibraryService
}

func NewLibraryHandler(svc *service.LibraryService) *LibraryHandler {
	return &LibraryHandler{svc: svc}
}

func profileID(c *gin.Context) (string, error) {
	pid := middleware.GetProfileID(c)
	if pid == "" {
		return "", apperr.BadRequest("缺少 profile_id")
	}
	return pid, nil
}

func (h *LibraryHandler) WantToWatch(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	items, err := h.svc.WantToWatch(c.Request.Context(), pid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *LibraryHandler) AddWant(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	_, err = h.svc.AddWant(c.Request.Context(), pid, c.Param("media_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *LibraryHandler) RemoveWant(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	_, err = h.svc.RemoveWant(c.Request.Context(), pid, c.Param("media_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *LibraryHandler) Favorites(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	items, err := h.svc.Favorites(c.Request.Context(), pid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *LibraryHandler) ToggleFavorite(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	added, err := h.svc.ToggleFavorite(c.Request.Context(), pid, c.Param("media_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok", "added": added})
}

func (h *LibraryHandler) Watching(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	items, err := h.svc.Watching(c.Request.Context(), pid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *LibraryHandler) Watched(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	items, err := h.svc.Watched(c.Request.Context(), pid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *LibraryHandler) History(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	items, err := h.svc.History(c.Request.Context(), pid, atoi(c.Query("limit"), 50))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}

func (h *LibraryHandler) ContinueWatching(c *gin.Context) {
	pid, err := profileID(c)
	if err != nil {
		respondError(c, err)
		return
	}
	items, err := h.svc.ContinueWatching(c.Request.Context(), pid, atoi(c.Query("limit"), 12))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": items})
}
