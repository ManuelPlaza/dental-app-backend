package services

import (
	"dental-app/internal/adapters/nequi"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type paymentLinkService struct {
	repo        ports.PaymentLinkRepository
	paymentRepo ports.PaymentRepository
	appointRepo ports.AppointmentRepository
	nequiClient *nequi.Client
}

func NewPaymentLinkService(
	repo ports.PaymentLinkRepository,
	paymentRepo ports.PaymentRepository,
	appointRepo ports.AppointmentRepository,
	nequiClient *nequi.Client,
) ports.PaymentLinkService {
	return &paymentLinkService{
		repo:        repo,
		paymentRepo: paymentRepo,
		appointRepo: appointRepo,
		nequiClient: nequiClient,
	}
}

// Generate crea un link de pago (total o parcial) para una cita completada.
func (s *paymentLinkService) Generate(appointmentID uint, amount float64, phone string, createdBy uint) (*domain.PaymentLink, error) {
	// 1. Validar monto
	if amount <= 0 {
		return nil, errors.New("el monto debe ser mayor a cero")
	}
	if len(phone) < 10 {
		return nil, errors.New("número de celular inválido")
	}

	// 2. Verificar que la cita existe y está completada
	app, err := s.appointRepo.GetByID(appointmentID)
	if err != nil {
		return nil, errors.New("cita no encontrada")
	}
	if app.Status != "completed" {
		return nil, errors.New("solo se pueden generar links de pago para citas completadas")
	}

	// 3. Calcular saldo pendiente
	total, paid, pending, err := s.calcBalance(appointmentID)
	if err != nil {
		return nil, err
	}
	if pending <= 0 {
		return nil, fmt.Errorf("la cita ya está completamente pagada (total: $%.0f, pagado: $%.0f)", total, paid)
	}

	// 4. Validar que el monto no supere el saldo pendiente
	if amount > pending+0.01 { // tolerancia de 1 centavo por redondeo
		return nil, fmt.Errorf("el monto solicitado ($%.0f) supera el saldo pendiente ($%.0f)", amount, pending)
	}

	// 5. Cancelar cualquier link pendiente anterior para esta cita
	if err := s.repo.CancelPendingByAppointment(appointmentID); err != nil {
		return nil, fmt.Errorf("error cancelando link anterior: %w", err)
	}

	// 6. Calcular expiración desde variable de entorno
	expiryMinutes := 30 // default: 30 minutos (estándar PSE/Nequi)
	if v := os.Getenv("PAYMENT_LINK_EXPIRY_MINUTES"); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			expiryMinutes = n
		}
	}

	// 7. Generar token UUID v4 criptográficamente seguro
	token := uuid.New().String()

	link := &domain.PaymentLink{
		AppointmentID: appointmentID,
		Amount:        amount,
		Token:         token,
		Status:        domain.PaymentLinkPending,
		PhoneNumber:   phone,
		ExpiresAt:     time.Now().Add(time.Duration(expiryMinutes) * time.Minute),
		CreatedBy:     createdBy,
	}

	// 8. Guardar en BD antes de llamar Nequi (garantiza trazabilidad)
	if err := s.repo.Save(link); err != nil {
		return nil, fmt.Errorf("error guardando link de pago: %w", err)
	}

	// 9. Enviar notificación Nequi de forma asíncrona (no bloquea la respuesta al admin)
	go func() {
		resp, err := s.nequiClient.SendPaymentNotification(nequi.PaymentNotificationRequest{
			PhoneNumber: phone,
			Amount:      amount,
			Reference:   token,
			Description: fmt.Sprintf("Pago servicio dental - Cita #%d", appointmentID),
		})
		if err != nil {
			log.Printf("❌ [PaymentLink] Error enviando notificación Nequi (link %d): %v", link.ID, err)
			return
		}
		if err := s.repo.UpdateNequiTxnID(link.ID, resp.TransactionID); err != nil {
			log.Printf("⚠️  [PaymentLink] No se pudo guardar nequi_txn_id para link %d: %v", link.ID, err)
		}
		log.Printf("✅ [PaymentLink] Notificación enviada — link %d | txn: %s", link.ID, resp.TransactionID)
	}()

	return link, nil
}

// GetByToken retorna el estado de un link por token (endpoint público).
func (s *paymentLinkService) GetByToken(token string) (*domain.PaymentLink, error) {
	if token == "" {
		return nil, errors.New("token inválido")
	}
	link, err := s.repo.GetByToken(token)
	if err != nil {
		return nil, errors.New("link de pago no encontrado")
	}
	return link, nil
}

// GetByAppointment lista todos los links de una cita (admin).
func (s *paymentLinkService) GetByAppointment(appointmentID uint) ([]domain.PaymentLink, error) {
	return s.repo.GetByAppointmentID(appointmentID)
}

// Cancel anula un link pendiente desde el admin.
func (s *paymentLinkService) Cancel(token string) error {
	link, err := s.repo.GetByToken(token)
	if err != nil {
		return errors.New("link de pago no encontrado")
	}
	if link.Status != domain.PaymentLinkPending {
		return fmt.Errorf("el link no se puede cancelar (estado actual: %s)", link.Status)
	}
	return s.repo.UpdateStatus(link.ID, domain.PaymentLinkCancelled, nil, "")
}

// ProcessWebhook procesa el callback de Nequi.
// Si el pago fue aceptado crea automáticamente el registro de pago.
func (s *paymentLinkService) ProcessWebhook(payload *ports.NequiWebhookPayload, rawBody string) error {
	// Buscar el link por token (reference que enviamos a Nequi)
	link, err := s.repo.GetByToken(payload.Reference)
	if err != nil {
		// Intentar por nequi_txn_id como fallback
		link, err = s.repo.GetByNequiTxnID(payload.TransactionID)
		if err != nil {
			return fmt.Errorf("link no encontrado para txn %s / ref %s", payload.TransactionID, payload.Reference)
		}
	}

	// Idempotencia: si ya fue procesado, ignorar
	if link.Status != domain.PaymentLinkPending {
		log.Printf("⚠️  [Webhook] Link %s ya fue procesado (estado: %s) — ignorando", link.Token, link.Status)
		return nil
	}

	switch payload.Status {
	case "SUCCESS":
		now := time.Now()

		// Crear registro de pago automáticamente
		payment := &domain.Payment{
			AppointmentID: link.AppointmentID,
			Amount:        link.Amount,
			Method:        "nequi",
			ReferenceCode: payload.TransactionID,
			Notes:         fmt.Sprintf("Pago automático vía link Nequi — txn: %s", payload.TransactionID),
			PaymentDate:   now,
		}
		if err := s.paymentRepo.Save(payment); err != nil {
			return fmt.Errorf("error registrando pago automático: %w", err)
		}

		// Marcar link como pagado
		if err := s.repo.UpdateStatus(link.ID, domain.PaymentLinkPaid, &now, rawBody); err != nil {
			log.Printf("⚠️  [Webhook] Pago creado pero no se actualizó status del link %d: %v", link.ID, err)
		}

		log.Printf("✅ [Webhook] Pago automático registrado — cita %d | $%.0f | txn: %s",
			link.AppointmentID, link.Amount, payload.TransactionID)

	case "REJECTED":
		if err := s.repo.UpdateStatus(link.ID, domain.PaymentLinkRejected, nil, rawBody); err != nil {
			return err
		}
		log.Printf("❌ [Webhook] Pago rechazado — link %s | txn: %s", link.Token, payload.TransactionID)

	case "EXPIRED":
		if err := s.repo.UpdateStatus(link.ID, domain.PaymentLinkExpired, nil, rawBody); err != nil {
			return err
		}
		log.Printf("⏰ [Webhook] Link expirado por Nequi — link %s", link.Token)

	default:
		log.Printf("⚠️  [Webhook] Estado Nequi desconocido '%s' para link %s", payload.Status, link.Token)
	}

	return nil
}

// ExpireStale marca como expirados los links vencidos (lo llama el worker).
func (s *paymentLinkService) ExpireStale() error {
	return s.repo.ExpireStale()
}

// calcBalance calcula (total, pagado, pendiente) para una cita.
func (s *paymentLinkService) calcBalance(appointmentID uint) (float64, float64, float64, error) {
	app, err := s.appointRepo.GetByID(appointmentID)
	if err != nil {
		return 0, 0, 0, errors.New("cita no encontrada")
	}
	payments, err := s.paymentRepo.GetByAppointmentID(appointmentID)
	if err != nil {
		return 0, 0, 0, err
	}
	var totalPaid float64
	for _, p := range payments {
		totalPaid += p.Amount
	}
	total := app.HistoricalPrice
	return total, totalPaid, total - totalPaid, nil
}
