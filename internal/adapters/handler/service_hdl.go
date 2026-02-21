package handler

import (
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	service ports.ServiceService
}

func NewServiceHandler(service ports.ServiceService) *ServiceHandler {
	return &ServiceHandler{service: service}
}

// GetAll maneja GET /services
func (h *ServiceHandler) GetAll(c *gin.Context) {
	services, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, services)
}
