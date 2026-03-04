package handler

import (
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ServiceCategoryHandler struct {
	service ports.ServiceCategoryService
}

func NewServiceCategoryHandler(service ports.ServiceCategoryService) *ServiceCategoryHandler {
	return &ServiceCategoryHandler{service: service}
}

// GetAll maneja GET /service-categories (público, sin auth)
func (h *ServiceCategoryHandler) GetAll(c *gin.Context) {
	categories, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, categories)
}
