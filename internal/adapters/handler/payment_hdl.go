package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"
	"strconv" // <--- ¡AQUÍ ESTÁ LO QUE FALTABA!

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service ports.PaymentService
}

func NewPaymentHandler(service ports.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) Create(c *gin.Context) {
	var payment domain.Payment

	if err := c.ShouldBindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido: " + err.Error()})
		return
	}

	if err := h.service.Process(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}
func (h *PaymentHandler) GetAll(c *gin.Context) {
	payments, err := h.service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, payments)
}
func (h *PaymentHandler) GetBalance(c *gin.Context) {
	idStr := c.Param("id") // ID de la CITA
	// (Recuerda importar "strconv" arriba si no está)
	// Aquí hacemos una conversión rápida (puedes mejorarla luego)
	// Asegúrate de tener: import "strconv"
	idUint64, _ := strconv.ParseUint(idStr, 10, 32)
	appID := uint(idUint64)

	total, paid, pending, err := h.service.GetBalance(appID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"appointment_id":  appID,
		"total_cost":      total,
		"total_paid":      paid,
		"pending_balance": pending,
		"status":          getItemStatus(pending), // Una funcioncita visual
	})
}

func getItemStatus(pending float64) string {
	if pending <= 0 {
		return "PAID_IN_FULL" // Pagado total
	}
	return "PARTIAL" // Debe dinero
}
