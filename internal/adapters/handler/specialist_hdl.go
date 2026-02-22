package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SpecialistHandler struct {
	service ports.SpecialistService
}

func NewSpecialistHandler(service ports.SpecialistService) *SpecialistHandler {
	return &SpecialistHandler{service: service}
}

func (h *SpecialistHandler) Create(c *gin.Context) {
	var s domain.Specialist
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.service.Create(&s); err != nil {
		switch {
		case errors.Is(err, domain.ErrLicenseRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrSpecialistAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, s)
}

func (h *SpecialistHandler) GetAll(c *gin.Context) {
	list, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *SpecialistHandler) Inactivate(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	if err := h.service.Inactivate(uint(id64)); err != nil {
		switch {
		case errors.Is(err, domain.ErrSpecialistNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrInvalidPayload):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "specialist inactivated"})
}

// GetWithoutUser maneja GET /specialists/without-user
func (h *SpecialistHandler) GetWithoutUser(c *gin.Context) {
	list, err := h.service.GetWithoutUser()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, list)
}
func (h *SpecialistHandler) Activate(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	if err := h.service.Activate(uint(id64)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "specialist activated"})
}
