// Package nequi encapsula la integración con la API de Nequi (docs.conecta.nequi.com.co).
// Mientras no haya credenciales configuradas opera en modo simulación:
// registra la operación en logs pero no llama servicios externos.
package nequi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	prodBaseURL    = "https://api.nequi.com.co"
	sandboxBaseURL = "https://api.sandbox.nequi.com.co"
)

// Client gestiona la comunicación con Nequi API.
type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client
	simMode   bool // true cuando no hay credenciales
}

// NewClient construye el cliente desde variables de entorno:
//
//	NEQUI_API_KEY      — clave de API merchant
//	NEQUI_API_SECRET   — secreto de API merchant
//	NEQUI_SANDBOX=true — usar ambiente de pruebas (default: true)
func NewClient() *Client {
	apiKey := os.Getenv("NEQUI_API_KEY")
	apiSecret := os.Getenv("NEQUI_API_SECRET")

	baseURL := sandboxBaseURL
	if os.Getenv("NEQUI_SANDBOX") == "false" {
		baseURL = prodBaseURL
	}

	simMode := apiKey == "" || apiSecret == ""
	if simMode {
		log.Println("⚠️  [Nequi] NEQUI_API_KEY/NEQUI_API_SECRET no configurados — modo simulación activo")
	} else {
		log.Printf("✅ [Nequi] Cliente configurado — baseURL: %s", baseURL)
	}

	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		simMode:   simMode,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
}

// IsSimMode retorna true cuando no hay credenciales reales.
func (c *Client) IsSimMode() bool { return c.simMode }

// --- Tipos de request/response ---

// PaymentNotificationRequest representa el cobro push a un celular.
type PaymentNotificationRequest struct {
	PhoneNumber string  `json:"phoneNumber"`
	Amount      float64 `json:"amount"`
	Reference   string  `json:"reference"`  // token UUID del PaymentLink
	Description string  `json:"description"`
}

// PaymentNotificationResponse respuesta de Nequi al enviar el cobro.
type PaymentNotificationResponse struct {
	TransactionID string `json:"transactionId"`
	Status        string `json:"status"` // "pending" mientras espera aceptación
}

// WebhookPayload estructura del callback que Nequi envía a nuestro endpoint
// cuando el usuario acepta o rechaza en la app.
type WebhookPayload struct {
	TransactionID string  `json:"transactionId"`
	Reference     string  `json:"reference"` // token UUID que enviamos
	Status        string  `json:"status"`    // "SUCCESS" | "REJECTED" | "EXPIRED"
	Amount        float64 `json:"amount"`
	PhoneNumber   string  `json:"phoneNumber"`
	Timestamp     string  `json:"timestamp"`
}

// SendPaymentNotification envía un cobro push al celular del paciente.
// En modo simulación registra la operación y retorna un ID ficticio.
func (c *Client) SendPaymentNotification(req PaymentNotificationRequest) (*PaymentNotificationResponse, error) {
	if c.simMode {
		simID := fmt.Sprintf("SIM-%s", req.Reference[:8])
		log.Printf("🔔 [Nequi SIM] Cobro $%.0f → cel %s | ref: %s | txn: %s",
			req.Amount, req.PhoneNumber, req.Reference, simID)
		return &PaymentNotificationResponse{
			TransactionID: simID,
			Status:        "pending",
		}, nil
	}

	// --- Llamada real a Nequi API ---
	// Endpoint: POST /payments/v2/-services-notificationPayment-PaymentService-sendPayment
	// Auth: AWS Signature v4 con API Key (migra a OAuth 2.0 en Oct 2025)
	// TODO: implementar AWS Sig v4 cuando Nequi entregue las credenciales merchant

	body, _ := json.Marshal(map[string]interface{}{
		"RequestMessage": map[string]interface{}{
			"RequestHeader": map[string]interface{}{
				"Channel":      "APP",
				"RequestDate":  time.Now().Format(time.RFC3339),
				"MessageID":    req.Reference,
				"ClientID":     c.apiKey,
				"Destination":  map[string]string{"ServiceName": "PaymentService", "ServiceOperation": "sendPayment"},
			},
			"RequestBody": map[string]interface{}{
				"any": map[string]interface{}{
					"phoneNumber": req.PhoneNumber,
					"value":       fmt.Sprintf("%.0f", req.Amount),
					"reference":   req.Reference,
					"description": req.Description,
				},
			},
		},
	})

	httpReq, err := http.NewRequest("POST",
		c.baseURL+"/payments/v2/-services-notificationPayment-PaymentService-sendPayment",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error conectando Nequi API: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nequi API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result PaymentNotificationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.New("error parseando respuesta Nequi")
	}
	return &result, nil
}
