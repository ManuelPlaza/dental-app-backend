package handler

import (
	"net/http"
	"strconv"

	"dental-app/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	svc ports.ReferralService
}

func NewReferralHandler(svc ports.ReferralService) *ReferralHandler {
	return &ReferralHandler{svc: svc}
}

// TopReferrers maneja GET /api/v1/admin/referrals/top
func (h *ReferralHandler) TopReferrers(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	referrers, err := h.svc.TopReferrers(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error obteniendo referidos"})
		return
	}
	c.JSON(http.StatusOK, referrers)
}
