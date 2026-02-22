package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"time"
)

// 1. DEFINICIÓN DE LA ESTRUCTURA (Esto es lo que te faltaba)
type appointmentService struct {
	repo        ports.AppointmentRepository
	patientRepo ports.PatientRepository // <--- debe estar esta línea
}

// 2. CONSTRUCTOR
func NewAppointmentService(repo ports.AppointmentRepository, patientRepo ports.PatientRepository) ports.AppointmentService {
	return &appointmentService{repo: repo, patientRepo: patientRepo}
}

// 3. MÉTODO AGENDAR (Schedule)
func (s *appointmentService) Schedule(app *domain.Appointment) error {
	// Regla: Hora fin debe ser después de hora inicio
	if app.EndTime.Before(app.StartTime) {
		return errors.New("la hora de fin no puede ser antes de la hora de inicio")
	}

	// Si no viene patient_id, buscar o crear paciente por documento
	if app.PatientID == 0 {
		if app.Patient.DocumentNumber == "" {
			return errors.New("debes enviar patient_id o los datos del paciente con document_number")
		}

		existing, err := s.patientRepo.FindByDocumentNumber(app.Patient.DocumentNumber)
		if err != nil {
			// No existe → crearlo con user_id NULL (sin cuenta web)
			if err := s.patientRepo.Save(&app.Patient); err != nil {
				return errors.New("error creando paciente: " + err.Error())
			}
		} else {
			// Ya existe → reutilizarlo
			app.Patient = *existing
		}
		app.PatientID = app.Patient.ID // <--- Esta línea es la clave
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

// 6. MÉTODO ADMIN: Cambiar estado sin restricciones
func (s *appointmentService) AdminUpdateStatus(id uint, status string) error {
	// Validar que el status sea un valor permitido
	validStatuses := map[string]bool{
		"pending":   true,
		"scheduled": true,
		"completed": true,
		"cancelled": true,
	}
	if !validStatuses[status] {
		return errors.New("estado inválido, valores permitidos: pending, scheduled, completed, cancelled")
	}

	app, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	// Sin validación de modification_count ni tiempo
	app.Status = status
	return s.repo.Update(app)
}

// 7. MÉTODO LISTAR PAGINADO
func (s *appointmentService) ListPaginated(page, limit int, status string) ([]domain.Appointment, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return s.repo.GetPaginated(page, limit, status)
}

// GetSummary retorna conteos para las cards
func (s *appointmentService) GetSummary() (map[string]int64, error) {
	return s.repo.GetSummary()
}

// AdminUpdate actualiza cualquier campo sin restricciones
func (s *appointmentService) AdminUpdate(id uint, req domain.AdminUpdateRequest) error {
	validStatuses := map[string]bool{
		"pending": true, "scheduled": true,
		"completed": true, "cancelled": true,
	}

	if req.Status != "" && !validStatuses[req.Status] {
		return errors.New("estado inválido: pending, scheduled, completed, cancelled")
	}

	app, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	// Actualizar solo los campos que vienen en el request
	if req.Status != "" {
		app.Status = req.Status
	}
	if req.SpecialistID != nil {
		app.SpecialistID = *req.SpecialistID
	}
	if req.ServiceID != nil {
		app.ServiceID = *req.ServiceID
	}
	if req.StartTime != nil {
		app.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		app.EndTime = *req.EndTime
	}

	return s.repo.Update(app)
}
