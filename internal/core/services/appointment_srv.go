package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"time"
)

// 1. DEFINICIÓN DE LA ESTRUCTURA (Esto es lo que te faltaba)
type appointmentService struct {
	repo ports.AppointmentRepository
}

// 2. CONSTRUCTOR
func NewAppointmentService(repo ports.AppointmentRepository) ports.AppointmentService {
	return &appointmentService{repo: repo}
}

// 3. MÉTODO AGENDAR (Schedule)
func (s *appointmentService) Schedule(app *domain.Appointment) error {
	// Regla: Hora fin debe ser después de hora inicio
	if app.EndTime.Before(app.StartTime) {
		return errors.New("la hora de fin no puede ser antes de la hora de inicio")
	}

	// Estado por defecto
	if app.Status == "" {
		app.Status = "pending"
	}

	return s.repo.Save(app)
}

// 4. MÉTODO MODIFICAR (Modify) - Nueva lógica
func (s *appointmentService) Modify(id uint, newStart, newEnd time.Time) error {
	// Buscar la cita original
	app, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	// REGLA: Solo se puede modificar 1 vez
	if app.ModificationCount >= 1 {
		return errors.New("esta cita ya fue modificada una vez y no se permiten más cambios")
	}

	// REGLA: Mínimo 1 hora de antelación
	deadline := time.Now().Add(1 * time.Hour)
	if deadline.After(app.StartTime) {
		return errors.New("ya es muy tarde para modificar la cita (mínimo 1 hora antes)")
	}

	// Aplicar cambios
	app.StartTime = newStart
	app.EndTime = newEnd
	app.ModificationCount++

	return s.repo.Update(app)
}

// 5. MÉTODO CANCELAR (Cancel) - Nueva lógica
func (s *appointmentService) Cancel(id uint) error {
	app, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	// REGLA: Mínimo 2 horas de antelación para cancelar
	deadline := time.Now().Add(2 * time.Hour)
	if deadline.After(app.StartTime) {
		return errors.New("no se puede cancelar con menos de 2 horas de antelación")
	}

	app.Status = "cancelled"
	return s.repo.Update(app)
}
func (s *appointmentService) List() ([]domain.Appointment, error) {
	return s.repo.GetAll()
}
