package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PublicHandler expone endpoints sin autenticación para el landing page.
type PublicHandler struct {
	serviceSvc     ports.ServiceService
	patientSvc     ports.PatientService
	appointmentSvc ports.AppointmentService
}

func NewPublicHandler(serviceSvc ports.ServiceService, patientSvc ports.PatientService, appointmentSvc ports.AppointmentService) *PublicHandler {
	return &PublicHandler{
		serviceSvc:     serviceSvc,
		patientSvc:     patientSvc,
		appointmentSvc: appointmentSvc,
	}
}

// GetServices maneja GET /api/v1/public/services
// Retorna únicamente los servicios activos para mostrar en el landing page.
func (h *PublicHandler) GetServices(c *gin.Context) {
	services, err := h.serviceSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, services)
}

// FindPatientByDocument maneja GET /api/v1/public/patients/document/:document_number
// Permite al landing page verificar si un paciente ya existe para pre-rellenar el formulario de cita.
func (h *PublicHandler) FindPatientByDocument(c *gin.Context) {
	documentNumber := c.Param("document_number")

	patient, err := h.patientSvc.FindByDocument(documentNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "paciente no encontrado"})
		return
	}

	c.JSON(http.StatusOK, patient)
}

// RequestAppointment maneja POST /api/v1/public/appointments
// Crea una solicitud de cita desde la landing page sin autenticación.
// Seguridad: status siempre "pending", specialist nunca asignado, end_time calculado automáticamente.
func (h *PublicHandler) RequestAppointment(c *gin.Context) {
	var req domain.PublicAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos incompletos: " + err.Error()})
		return
	}

	app, err := h.appointmentSvc.ScheduleFromWeb(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Solicitud de cita recibida. Te contactaremos para confirmar.",
		"appointment_id": app.ID,
		"start_time":     app.StartTime,
		"end_time":       app.EndTime,
		"status":         app.Status,
	})
}
