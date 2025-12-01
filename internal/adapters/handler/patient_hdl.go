package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PatientHandler struct{ service ports.PatientService }

func NewPatientHandler(service ports.PatientService) *PatientHandler {
	return &PatientHandler{service: service}
}
func (h *PatientHandler) Create(c *gin.Context) {
	var p domain.Patient
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}
	if err := h.service.Create(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// GetAll maneja la petición GET /patients
func (h *PatientHandler) GetAll(c *gin.Context) {
	patients, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, patients)
}
