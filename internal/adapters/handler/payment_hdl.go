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
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno del servidor"})
		return
	}
	c.JSON(http.StatusOK, payments)
}
func (h *PaymentHandler) GetBalance(c *gin.Context) {
	idStr := c.Param("id")
	idUint64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cita inválido"})
		return
	}
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
		"status":          getPaymentStatus(paid, pending),
	})
}

func getPaymentStatus(paid, pending float64) string {
	if pending <= 0 {
		return "pagado"
	}
	if paid == 0 {
		return "pendiente"
	}
	return "parcial"
}

// Update maneja PUT /payments/:id
func (h *PaymentHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de pago inválido"})
		return
	}

	var payment domain.Payment
	if err := c.ShouldBindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.service.UpdatePayment(uint(id64), &payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pago actualizado correctamente"})
}
