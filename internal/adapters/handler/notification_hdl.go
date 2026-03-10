package handler

import (
	"dental-app/internal/core/ports"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service ports.NotificationService
}

func NewNotificationHandler(service ports.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// GET /api/v1/appointments/confirm?token=xxxxx  (ruta pública)
func (h *NotificationHandler) ConfirmAppointment(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token requerido"})
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3001"
	}

	if err := h.service.ConfirmAppointment(token); err != nil {
		c.Redirect(http.StatusFound, frontendURL+"/confirmar-cita?error="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, frontendURL+"/confirmar-cita?success=true")
}
