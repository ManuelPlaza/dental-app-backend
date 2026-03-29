package handler

import (
	"dental-app/internal/adapters/sse"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BannerHandler struct {
	service ports.BannerService
	hub     *sse.BannerHub
}

func NewBannerHandler(service ports.BannerService, hub *sse.BannerHub) *BannerHandler {
	return &BannerHandler{service: service, hub: hub}
}

// StreamEvents maneja GET /api/v1/public/banners/events (SSE)
func (h *BannerHandler) StreamEvents(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := h.hub.Subscribe()
	defer h.hub.Unsubscribe(ch)

	c.Stream(func(_ io.Writer) bool {
		select {
		case event, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("message", event)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// GetActive maneja GET /api/v1/public/banners
func (h *BannerHandler) GetActive(c *gin.Context) {
	banners, err := h.service.GetActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, banners)
}

// GetAll maneja GET /api/v1/banners (admin)
func (h *BannerHandler) GetAll(c *gin.Context) {
	banners, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, banners)
}

// GetByID maneja GET /api/v1/banners/:id (admin)
func (h *BannerHandler) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	banner, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "banner no encontrado"})
		return
	}
	c.JSON(http.StatusOK, banner)
}

// Create maneja POST /api/v1/banners (admin)
func (h *BannerHandler) Create(c *gin.Context) {
	var banner domain.Banner
	if err := c.ShouldBindJSON(&banner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.service.Create(&banner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.Broadcast("banners_updated")
	c.JSON(http.StatusCreated, banner)
}

// Update maneja PUT /api/v1/banners/:id (admin)
func (h *BannerHandler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var banner domain.Banner
	if err := c.ShouldBindJSON(&banner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.service.Update(id, &banner); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.Broadcast("banners_updated")
	c.JSON(http.StatusOK, gin.H{"message": "Banner actualizado correctamente"})
}

// Delete maneja DELETE /api/v1/banners/:id (admin)
func (h *BannerHandler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.hub.Broadcast("banners_updated")
	c.JSON(http.StatusOK, gin.H{"message": "Banner eliminado correctamente"})
}

func parseID(c *gin.Context) (uint, error) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	return uint(id64), err
}
