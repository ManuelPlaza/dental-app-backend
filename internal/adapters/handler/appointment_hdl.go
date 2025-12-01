package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AppointmentHandler struct {
	service ports.AppointmentService
}

func NewAppointmentHandler(service ports.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{service: service}
}

// --- 1. AGENDAR CITA (POST) ---
func (h *AppointmentHandler) Create(c *gin.Context) {
	var app domain.Appointment

	// BindJSON convierte el texto JSON a la estructura Go
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	if err := h.service.Schedule(&app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, app)
}

// --- 2. MODIFICAR CITA (PUT) ---
// Estructura auxiliar solo para recibir las nuevas fechas
type ModifyRequest struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (h *AppointmentHandler) Modify(c *gin.Context) {
	// A. Leer el ID de la URL (ej: /appointments/1)
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cita inválido"})
		return
	}
	id := uint(id64)

	// B. Leer el cuerpo JSON con las nuevas fechas
	var req ModifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido: " + err.Error()})
		return
	}

	// C. Llamar al servicio (que tiene las reglas de negocio de la hora límite)
	if err := h.service.Modify(id, req.StartTime, req.EndTime); err != nil {
		// Retornamos 409 Conflict porque viola una regla de negocio
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cita modificada con éxito"})
}

// --- 3. CANCELAR CITA (PATCH) ---
func (h *AppointmentHandler) Cancel(c *gin.Context) {
	// A. Leer el ID de la URL
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cita inválido"})
		return
	}
	id := uint(id64)

	// B. Llamar al servicio (Regla de 2 horas)
	if err := h.service.Cancel(id); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cita cancelada exitosamente"})
}
func (h *AppointmentHandler) GetAll(c *gin.Context) {
	apps, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, apps)
}
