package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"
	"strings"
	"unicode/utf8"

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

// maskedPatientResponse es la respuesta pública con PII enmascarada (ISO 27001 A.8.11).
// No expone ID interno ni documento completo.
type maskedPatientResponse struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Exists    bool   `json:"exists"`
}

// maskName enmascara un nombre dejando el primer y último carácter visibles.
// "Manuel" → "M****l", "Ana" → "A*a", "Jo" → "J*"
func maskName(s string) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return string(runes[0])
	}
	if n == 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + strings.Repeat("*", n-2) + string(runes[n-1])
}

// maskPhone enmascara un teléfono dejando los primeros 3 y últimos 2 dígitos.
// "3226520483" → "322****83"
func maskPhone(s string) string {
	n := utf8.RuneCountInString(s)
	if n <= 5 {
		return strings.Repeat("*", n)
	}
	runes := []rune(s)
	middle := strings.Repeat("*", n-5)
	return string(runes[:3]) + middle + string(runes[n-2:])
}

// maskEmail enmascara el email dejando los primeros 2 chars del usuario y el dominio completo.
// "usuario@gmail.com" → "us***@gmail.com"
func maskEmail(s string) string {
	at := strings.Index(s, "@")
	if at < 0 {
		return "***"
	}
	user := []rune(s[:at])
	domain := s[at:] // incluye el @
	if len(user) <= 2 {
		return string(user) + "***" + domain
	}
	return string(user[:2]) + strings.Repeat("*", len(user)-2) + domain
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
// Retorna datos enmascarados para confirmar identidad sin exponer PII completa (ISO 27001 A.8.11).
func (h *PublicHandler) FindPatientByDocument(c *gin.Context) {
	documentNumber := c.Param("document_number")

	patient, err := h.patientSvc.FindByDocument(documentNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "paciente no encontrado"})
		return
	}

	c.JSON(http.StatusOK, maskedPatientResponse{
		FirstName: maskName(patient.FirstName),
		LastName:  maskName(patient.LastName),
		Phone:     maskPhone(patient.Phone),
		Email:     maskEmail(patient.Email),
		Exists:    true,
	})
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
