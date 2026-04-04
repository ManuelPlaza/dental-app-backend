package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaymentLinkHandler struct {
	service ports.PaymentLinkService
}

func NewPaymentLinkHandler(service ports.PaymentLinkService) *PaymentLinkHandler {
	return &PaymentLinkHandler{service: service}
}

// Generate maneja POST /api/v1/appointments/:id/payment-links (admin, JWT requerido)
func (h *PaymentLinkHandler) Generate(c *gin.Context) {
	appointmentID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cita inválido"})
		return
	}

	var req domain.GeneratePaymentLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	// Obtener el user_id del token JWT (lo pone el middleware de auth)
	createdBy := uint(0)
	if uid, exists := c.Get("userID"); exists {
		if v, ok := uid.(uint); ok {
			createdBy = v
		}
	}

	link, err := h.service.Generate(appointmentID, req.Amount, req.PhoneNumber, createdBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           link.ID,
		"token":        link.Token,
		"amount":       link.Amount,
		"phone_number": link.PhoneNumber,
		"status":       link.Status,
		"expires_at":   link.ExpiresAt,
		"payment_url":  fmt.Sprintf("%s/pagar/%s", frontendURL, link.Token),
		"nequi_sim":    link.NequiTxnID == "" || len(link.NequiTxnID) > 3 && link.NequiTxnID[:3] == "SIM",
	})
}

// GetByAppointment maneja GET /api/v1/appointments/:id/payment-links (admin, JWT requerido)
func (h *PaymentLinkHandler) GetByAppointment(c *gin.Context) {
	appointmentID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cita inválido"})
		return
	}

	links, err := h.service.GetByAppointment(appointmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno"})
		return
	}

	c.JSON(http.StatusOK, links)
}

// Cancel maneja DELETE /api/v1/payment-links/:token (admin, JWT requerido)
func (h *PaymentLinkHandler) Cancel(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token requerido"})
		return
	}

	if err := h.service.Cancel(token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "link de pago cancelado"})
}

// GetStatus maneja GET /api/v1/pay/:token (público — sin JWT)
// Solo expone estado y monto, nunca datos del paciente ni de la cita.
func (h *PaymentLinkHandler) GetStatus(c *gin.Context) {
	token := c.Param("token")

	link, err := h.service.GetByToken(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link de pago no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     link.Status,
		"amount":     link.Amount,
		"expires_at": link.ExpiresAt,
		"paid_at":    link.PaidAt,
	})
}

// HandleNequiWebhook maneja POST /api/v1/webhooks/nequi (público, verificado por HMAC)
func (h *PaymentLinkHandler) HandleNequiWebhook(c *gin.Context) {
	// 1. Leer body completo para verificación HMAC
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
		return
	}

	// 2. Verificar firma HMAC-SHA256 si el secreto está configurado
	webhookSecret := os.Getenv("NEQUI_WEBHOOK_SECRET")
	if webhookSecret != "" {
		signature := c.GetHeader("X-Nequi-Signature")
		if !verifyNequiHMAC(body, signature, webhookSecret) {
			log.Printf("⚠️  [Webhook] Firma inválida desde IP: %s", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "firma inválida"})
			return
		}
	}

	// 3. Parsear payload
	var payload ports.NequiWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	log.Printf("📩 [Webhook] Nequi callback — txn: %s | ref: %s | status: %s",
		payload.TransactionID, payload.Reference, payload.Status)

	// 4. Procesar
	if err := h.service.ProcessWebhook(&payload, string(body)); err != nil {
		log.Printf("❌ [Webhook] Error procesando: %v", err)
		// Retornar 200 de todas formas para evitar reintentos de Nequi en errores de lógica
		c.JSON(http.StatusOK, gin.H{"status": "error_logged"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// verifyNequiHMAC verifica la firma HMAC-SHA256 del webhook de Nequi.
func verifyNequiHMAC(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// parseUintParam extrae un parámetro de ruta como uint.
func parseUintParam(c *gin.Context, name string) (uint, error) {
	s := c.Param(name)
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
