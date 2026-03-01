package handler

import (
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	service ports.DashboardService
}

func NewDashboardHandler(service ports.DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}