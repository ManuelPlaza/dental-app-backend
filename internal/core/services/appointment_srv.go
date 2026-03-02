package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"time"
)

var bogotaLoc, _ = time.LoadLocation("America/Bogota")

// 1. DEFINICIÓN DE LA ESTRUCTURA
type appointmentService struct {
	repo        ports.AppointmentRepository
	patientRepo ports.PatientRepository
	serviceRepo ports.ServiceRepository
}

// 2. CONSTRUCTOR
func NewAppointmentService(repo ports.AppointmentRepository, patientRepo ports.PatientRepository, serviceRepo ports.ServiceRepository) ports.AppointmentService {
	return &appointmentService{repo: repo, patientRepo: patientRepo, serviceRepo: serviceRepo}
}

// Helper: validar horarios de atención en hora de Bogotá
func validBusinessHours(t time.Time) error {
	// Convertir a hora de Bogotá para validar
	local := t.In(bogotaLoc)
	weekday := local.Weekday()
	timeInMinutes := local.Hour()*60 + local.Minute()

	// Domingo → cerrado
	if weekday == time.Sunday {
		return errors.New("no atendemos los domingos")
	}

	// Sábado → 8am a 12pm
	if weekday == time.Saturday {
		if timeInMinutes < 8*60 || timeInMinutes >= 12*60 {
			return errors.New("Los sábados atendemos de 8:00 a.m. a 12:00 p.m.")
		}
		return nil
	}

	// Lunes a Viernes → 8am a 6pm
	if timeInMinutes < 8*60 || timeInMinutes >= 18*60 {
		return errors.New("El horario de atención es de 8:00 a.m. a 6:00 p.m.")
	}

	return nil
}

// 3. MÉTODO AGENDAR (Schedule) — con validación de horarios
func (s *appointmentService) Schedule(app *domain.Appointment) error {
	// Convertir tiempos a Bogotá
	startBogota := app.StartTime.Time.In(bogotaLoc)
	endBogota := app.EndTime.Time.In(bogotaLoc)

	// Regla: Hora fin debe ser después de hora inicio
	if endBogota.Before(startBogota) {
		return errors.New("la hora de fin no puede ser antes de la hora de inicio")
	}

	// Regla: Validar horarios de atención
	if err := validBusinessHours(startBogota); err != nil {
		return err
	}
	if err := validBusinessHours(endBogota); err != nil {
		return err
	}

	// Regla: No se puede agendar en el pasado
	nowBogota := time.Now().In(bogotaLoc)
	if startBogota.Before(nowBogota) {
		return errors.New("no puedes agendar una cita en una fecha y hora pasada")
	}

	// Si no viene patient_id, buscar o crear paciente por documento
	if app.PatientID == 0 {
		if app.Patient.DocumentNumber == "" {
			return errors.New("debes enviar patient_id o los datos del paciente con document_number")
		}

		existing, err := s.patientRepo.FindByDocumentNumber(app.Patient.DocumentNumber)
		if err != nil {
			if err := s.patientRepo.Save(&app.Patient); err != nil {
				return errors.New("error creando paciente: " + err.Error())
			}
		} else {
			app.Patient = *existing
		}
		app.PatientID = app.Patient.ID
	}

	// Regla: Verificar conflicto de especialista
	if app.SpecialistID != 0 {
		conflict, err := s.repo.HasSpecialistConflict(app.SpecialistID, app.StartTime.Time, app.EndTime.Time, 0)
		if err != nil {
			return errors.New("error verificando disponibilidad del especialista")
		}
		if conflict {
			return errors.New("el especialista ya tiene una cita programada en ese horario")
		}
	}

	// Guardar precio histórico del servicio
	if app.ServiceID != 0 && app.HistoricalPrice == 0 {
		service, err := s.serviceRepo.GetByID(app.ServiceID)
		if err == nil {
			app.HistoricalPrice = service.Price
		}
	}

	// Estado por defecto
	if app.Status == "" {
		app.Status = "pending"
	}

	return s.repo.Save(app)
}

// 4. MÉTODO MODIFICAR (Modify)
func (s *appointmentService) Modify(id uint, newStart, newEnd time.Time) error {
	app, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	// REGLA: Solo se puede modificar 1 vez
	if app.ModificationCount >= 1 {
		return errors.New("esta cita ya fue modificada una vez y no se permiten más cambios")
	}

	// REGLA: Mínimo 1 hora de antelación en hora de Bogotá
	deadline := time.Now().In(bogotaLoc).Add(1 * time.Hour)
	if deadline.After(app.StartTime.Time.In(bogotaLoc)) {
		return errors.New("ya es muy tarde para modificar la cita (mínimo 1 hora antes)")
	}

	// Aplicar cambios en hora de Bogotá
	app.StartTime = domain.BogotaTime{Time: newStart.In(bogotaLoc)}
	app.EndTime = domain.BogotaTime{Time: newEnd.In(bogotaLoc)}
	app.ModificationCount++

	return s.repo.Update(app)
}

// 5. MÉTODO CANCELAR (Cancel)
func (s *appointmentService) Cancel(id uint) error {
	app, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	// REGLA: Mínimo 2 horas de antelación en hora de Bogotá
	deadline := time.Now().In(bogotaLoc).Add(2 * time.Hour)
	if deadline.After(app.StartTime.Time.In(bogotaLoc)) {
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

// AdminUpdate — con todas las reglas de negocio para el admin
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

	// Regla: Las citas completadas o canceladas no se pueden modificar
	if app.Status == "completed" || app.Status == "cancelled" {
		return errors.New("no se puede modificar una cita " + app.Status)
	}

	// Validar motivo si se va a cancelar
	if req.Status == "cancelled" {
		if req.CancellationReason == "" {
			return errors.New("debes seleccionar un motivo de cancelación")
		}
		validReasons := map[string]bool{
			"no_show": true, "patient_request": true,
			"auto_expired": true, "emergency": true,
			"scheduling_conflict": true, "specialist_unavailable": true,
			"clinic_decision": true, "other": true,
		}
		if !validReasons[req.CancellationReason] {
			return errors.New("motivo de cancelación inválido")
		}
		app.CancellationReason = req.CancellationReason
		app.CancellationNotes = req.CancellationNotes
	}

	// Regla: La nueva fecha no puede ser en el pasado (en hora Bogotá)
	if req.StartTime != nil && req.StartTime.Time.In(bogotaLoc).Before(time.Now().In(bogotaLoc)) {
		return errors.New("no puedes reprogramar una cita a una fecha y hora pasada")
	}

	// Determinar tiempos para validación de conflicto
	startTime := app.StartTime.Time
	endTime := app.EndTime.Time
	if req.StartTime != nil {
		startTime = req.StartTime.Time.In(bogotaLoc)
	}
	if req.EndTime != nil {
		endTime = req.EndTime.Time.In(bogotaLoc)
	}

	// Regla: Verificar conflicto de especialista si cambia especialista o fechas
	specialistID := app.SpecialistID
	if req.SpecialistID != nil {
		specialistID = *req.SpecialistID
	}

	if req.SpecialistID != nil || req.StartTime != nil || req.EndTime != nil {
		conflict, err := s.repo.HasSpecialistConflict(specialistID, startTime, endTime, id)
		if err != nil {
			return errors.New("error verificando disponibilidad del especialista")
		}
		if conflict {
			return errors.New("el especialista ya tiene una cita programada en ese horario")
		}
	}

	// Aplicar cambios
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
		app.StartTime = domain.BogotaTime{Time: req.StartTime.Time.In(bogotaLoc)}
	}
	if req.EndTime != nil {
		app.EndTime = domain.BogotaTime{Time: req.EndTime.Time.In(bogotaLoc)}
	}

	return s.repo.Update(app)
}
