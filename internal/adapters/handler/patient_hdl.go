package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"
	"errors"
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

        switch {
        case errors.Is(err, domain.ErrDocumentRequired):
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return

        case errors.Is(err, domain.ErrPatientAlreadyExists):
            c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
            return

        default:
            // No filtrar error crudo de DB
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
            return
        }
    }

    c.JSON(http.StatusCreated, p)
}

// GetAll maneja la petición GET /patients
func (h *PatientHandler) GetAll(c *gin.Context) {
	patients, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, patients)
}
