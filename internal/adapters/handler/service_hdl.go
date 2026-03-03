package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	service ports.ServiceService
}

func NewServiceHandler(service ports.ServiceService) *ServiceHandler {
	return &ServiceHandler{service: service}
}

// GetAll maneja GET /services (solo activos, para landing page)
func (h *ServiceHandler) GetAll(c *gin.Context) {
	services, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, services)
}

// GetAllAdmin maneja GET /admin/services (todos, para admin)
func (h *ServiceHandler) GetAllAdmin(c *gin.Context) {
	services, err := h.service.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, services)
}

// Create maneja POST /services
func (h *ServiceHandler) Create(c *gin.Context) {
	var service domain.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.service.Create(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, service)
}

// Update maneja PUT /services/:id
func (h *ServiceHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var service domain.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.service.Update(uint(id64), &service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Servicio actualizado correctamente"})
}

// Activate maneja PATCH /services/:id/activate
func (h *ServiceHandler) Activate(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.service.Activate(uint(id64)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Servicio activado correctamente"})
}

// Inactivate maneja PATCH /services/:id/inactivate
func (h *ServiceHandler) Inactivate(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.service.Inactivate(uint(id64)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Servicio inactivado correctamente"})
}
