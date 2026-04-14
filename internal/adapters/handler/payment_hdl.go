package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"math"
	"net/http"
	"strconv"

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

	// pending_balance nunca negativo (por si total_paid > total_cost por error de datos)
	pendingBalance := math.Max(0, pending)

	c.JSON(http.StatusOK, gin.H{
		"appointment_id":  appID,
		"total_cost":      total,
		"total_paid":      paid,
		"pending_balance": pendingBalance,
		"status":          getPaymentStatus(total, paid, pendingBalance),
	})
}

// getPaymentStatus deriva el estado del pago exclusivamente de los montos calculados.
// Nunca lee campos de estado de la BD para evitar desincronización.
func getPaymentStatus(total, paid, pending float64) string {
	// Sin precio definido aún — no se puede determinar si está pagado
	if total <= 0 {
		return "pendiente"
	}
	// Saldo cero o negativo (sobrepago por redondeo) → completamente pagado
	if pending <= 0 {
		return "pagado"
	}
	// Tiene saldo pero no se ha abonado nada
	if paid == 0 {
		return "pendiente"
	}
	// Pago parcial
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
