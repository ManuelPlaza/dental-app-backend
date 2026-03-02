package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"time"
)

type paymentService struct {
	payRepo     ports.PaymentRepository
	appointRepo ports.AppointmentRepository // <--- Necesitamos esto para ver el precio
}

// Actualizamos el constructor para pedir ambos repositorios
func NewPaymentService(payRepo ports.PaymentRepository, appointRepo ports.AppointmentRepository) ports.PaymentService {
	return &paymentService{
		payRepo:     payRepo,
		appointRepo: appointRepo,
	}
}

// ... (El método Process queda igual, pégalo aquí abajo) ...
func (s *paymentService) Process(payment *domain.Payment) error {
	// Permitir amount=0 solo si es un pago pendiente generado automáticamente
	if payment.Amount < 0 {
		return errors.New("el monto del pago no puede ser negativo")
	}
	if payment.Amount == 0 && payment.Method != "pending" {
		return errors.New("el monto del pago debe ser mayor a cero")
	}

	// Nequi requiere código de referencia
	if payment.Method == "nequi" && payment.ReferenceCode == "" {
		return errors.New("para pagos con Nequi, el código de referencia es obligatorio")
	}

	if payment.PaymentDate.IsZero() {
		payment.PaymentDate = time.Now()
	}

	return s.payRepo.Save(payment)
}

// ... (Método List queda igual) ...
func (s *paymentService) List() ([]domain.Payment, error) {
	return s.payRepo.GetAll()
}

// === NUEVA FUNCIÓN: CALCULAR SALDO ===
func (s *paymentService) GetBalance(appID uint) (float64, float64, float64, error) {
	// 1. Obtener la cita para saber cuánto costaba
	app, err := s.appointRepo.GetByID(appID)
	if err != nil {
		return 0, 0, 0, errors.New("cita no encontrada")
	}

	// 2. Obtener todos los pagos hechos a esa cita
	payments, err := s.payRepo.GetByAppointmentID(appID)
	if err != nil {
		return 0, 0, 0, err
	}

	// 3. Sumar lo pagado
	var totalPaid float64 = 0
	for _, p := range payments {
		totalPaid += p.Amount
	}

	// 4. Calcular deuda
	totalCost := app.HistoricalPrice
	remaining := totalCost - totalPaid

	return totalCost, totalPaid, remaining, nil
}
func (s *paymentService) UpdatePayment(id uint, payment *domain.Payment) error {
	// Buscar el registro existente
	existing, err := s.payRepo.GetByID(id)
	if err != nil {
		return errors.New("registro de pago no encontrado")
	}

	// Validar monto
	if payment.Amount < 0 {
		return errors.New("el monto no puede ser negativo")
	}

	// Nequi requiere código de referencia
	if payment.Method == "nequi" && payment.ReferenceCode == "" {
		return errors.New("para pagos con Nequi, el código de referencia es obligatorio")
	}

	// Actualizar solo los campos que vienen
	if payment.Amount > 0 {
		existing.Amount = payment.Amount
	}
	if payment.Method != "" {
		existing.Method = payment.Method
	}
	if payment.ReferenceCode != "" {
		existing.ReferenceCode = payment.ReferenceCode
	}
	if payment.Notes != "" {
		existing.Notes = payment.Notes
	}

	return s.payRepo.Update(existing)
}
func (s *paymentService) GetByAppointment(appID uint) ([]domain.Payment, error) {
	return s.payRepo.GetByAppointmentID(appID)
}
