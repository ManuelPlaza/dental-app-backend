package services

import (
	"context"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var bogotaLoc, _ = time.LoadLocation("America/Bogota")

// 1. DEFINICIÓN DE LA ESTRUCTURA
type appointmentService struct {
	repo            ports.AppointmentRepository
	patientRepo     ports.PatientRepository
	serviceRepo     ports.ServiceRepository
	specialistRepo  ports.SpecialistRepository
	notificationSrv ports.NotificationService
	consentRepo     ports.DataConsentRepository
	metaClient      ports.MetaCAPIClient
}

// 2. CONSTRUCTOR
func NewAppointmentService(
	repo ports.AppointmentRepository,
	patientRepo ports.PatientRepository,
	serviceRepo ports.ServiceRepository,
	specialistRepo ports.SpecialistRepository,
	notificationSrv ports.NotificationService,
	consentRepo ports.DataConsentRepository,
	metaClient ports.MetaCAPIClient,
) ports.AppointmentService {
	return &appointmentService{
		repo:            repo,
		patientRepo:     patientRepo,
		serviceRepo:     serviceRepo,
		specialistRepo:  specialistRepo,
		notificationSrv: notificationSrv,
		consentRepo:     consentRepo,
		metaClient:      metaClient,
	}
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

	if err := s.repo.Save(app); err != nil {
		return err
	}

	go func() {
		if err := s.notificationSrv.ScheduleConfirmation(app); err != nil {
			log.Printf("⚠️ Error confirmación cita %d: %s", app.ID, err.Error())
		}
		if err := s.notificationSrv.ScheduleReminders(app); err != nil {
			log.Printf("⚠️ Error recordatorios cita %d: %s", app.ID, err.Error())
		}
	}()

	return nil
}

// ScheduleFromWeb crea una cita desde la landing page pública.
// Reglas de seguridad:
//   - status siempre "pending" (ignorado del request)
//   - specialist_id nunca asignado (admin lo asigna después)
//   - end_time calculado automáticamente desde duration_minutes del servicio
func (s *appointmentService) ScheduleFromWeb(req *domain.PublicAppointmentRequest) (*domain.Appointment, error) {
	// 1. Verificar consentimiento Ley 1581 — PRIMERA validación, bloquea todo lo demás.
	// Si el paciente ya aceptó en una cita anterior, no se le exige volver a aceptar
	// a menos que la versión de la política haya cambiado.
	existingConsent, _ := s.consentRepo.FindByDocument(req.DocumentNumber)
	needsNewConsent := existingConsent == nil

	if needsNewConsent {
		// Paciente nuevo o sin consentimiento previo: debe aceptar explícitamente
		if !req.DatosAceptados {
			return nil, errors.New("debe aceptar el tratamiento de datos personales para continuar (Ley 1581 de 2012)")
		}
		if req.DatosAceptadosAt.IsZero() {
			return nil, errors.New("la fecha de aceptación de datos es requerida")
		}
		now := time.Now().UTC()
		diff := now.Sub(req.DatosAceptadosAt.UTC())
		if diff < -5*time.Minute {
			return nil, errors.New("la fecha de aceptación de datos no puede ser futura")
		}
		if diff > 30*time.Minute {
			return nil, errors.New("la sesión de consentimiento ha expirado, por favor recarga el formulario")
		}
	}

	// 2. Validar campos básicos
	if len(req.DocumentNumber) < 5 {
		return nil, errors.New("número de documento inválido")
	}
	if len(req.Phone) < 7 {
		return nil, errors.New("teléfono inválido")
	}

	// 2. Validar horario de atención
	startBogota := req.StartTime.Time.In(bogotaLoc)
	if err := validBusinessHours(startBogota); err != nil {
		return nil, err
	}
	if startBogota.Before(time.Now().In(bogotaLoc)) {
		return nil, errors.New("no puedes agendar una cita en una fecha y hora pasada")
	}

	// 3. Obtener servicio para calcular end_time y precio histórico
	svc, err := s.serviceRepo.GetByID(req.ServiceID)
	if err != nil {
		return nil, errors.New("servicio no encontrado")
	}
	endBogota := startBogota.Add(time.Duration(svc.DurationMinutes) * time.Minute)

	// 4. Buscar o crear paciente
	patient, err := s.patientRepo.FindByDocumentNumber(req.DocumentNumber)
	if err != nil {
		patient = &domain.Patient{
			DocumentNumber: req.DocumentNumber,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Phone:          req.Phone,
			Email:          req.Email,
		}
		if err := s.patientRepo.Save(patient); err != nil {
			return nil, errors.New("error registrando paciente")
		}
	}

	// 5. Obtener especialista default para satisfacer la FK (admin reasigna después)
	defaultSpecialist, err := s.specialistRepo.GetDefault()
	if err != nil {
		return nil, errors.New("no hay un especialista principal configurado; contacta al administrador")
	}

	// 6. Verificar disponibilidad del especialista principal (Opción A — safety net)
	conflict, err := s.repo.HasSpecialistConflict(defaultSpecialist.ID, startBogota, endBogota, 0)
	if err != nil {
		return nil, errors.New("error verificando disponibilidad")
	}
	if conflict {
		return nil, errors.New("el horario seleccionado ya está ocupado. Por favor elige otro horario disponible")
	}

	// 7. Construir la cita — status y specialist SIEMPRE forzados internamente
	app := &domain.Appointment{
		PatientID:       patient.ID,
		ServiceID:       req.ServiceID,
		SpecialistID:    defaultSpecialist.ID, // FK satisfecha; admin reasigna desde el panel
		Status:          "pending",             // Siempre pending, nunca del request
		StartTime:       domain.BogotaTime{Time: startBogota},
		EndTime:         domain.BogotaTime{Time: endBogota},
		HistoricalPrice: svc.Price,
		Notes:           req.Notes,
		Fbclid:          req.Fbclid,
	}

	// 7. Construir consentimiento solo si es necesario
	var consent *domain.DataConsent
	if needsNewConsent {
		consent = &domain.DataConsent{
			DocumentoTitular: req.DocumentNumber,
			Aceptado:         true,
			AceptadoAt:       req.DatosAceptadosAt.UTC(),
			IPAddress:        req.IPAddress,
			VersionPolitica:  "1.0",
			UserAgent:        req.UserAgent,
		}
	}

	// 8. Persistir cita (+ consentimiento si aplica) en una sola transacción
	if needsNewConsent {
		if err := s.repo.SaveWithConsent(app, consent); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.Save(app); err != nil {
			return nil, err
		}
	}

	// 9. Notificación de confirmación (async)
	app.Patient = *patient
	go func() {
		if err := s.notificationSrv.ScheduleConfirmation(app); err != nil {
			log.Printf("⚠️ Error confirmación cita web %d: %s", app.ID, err.Error())
		}
	}()

	// 10. Meta CAPI Lead event (async — no bloquea la respuesta)
	go func() {
		landingURL := os.Getenv("LANDING_URL")
		if landingURL == "" {
			landingURL = "https://tecnicadentaljc.com"
		}
		if err := s.metaClient.SendLeadEvent(context.Background(), ports.MetaLeadEvent{
			EventTime:      time.Now().Unix(),
			EventSourceURL: landingURL,
			Fbclid:         req.Fbclid,
			Phone:          req.Phone,
			Email:          req.Email,
			ExternalID:     req.DocumentNumber,
		}); err != nil {
			log.Printf("⚠️ Meta CAPI Lead cita %d: %s", app.ID, err.Error())
		}
	}()

	return app, nil
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

// AutoCancelExpired cancela citas pending cuyo plazo de confirmación venció.
// Lee APPOINTMENT_CONFIRM_HOURS del entorno (default: 1).
func (s *appointmentService) AutoCancelExpired() (int64, error) {
	confirmHours := 1
	if v := os.Getenv("APPOINTMENT_CONFIRM_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			confirmHours = n
		}
	}
	return s.repo.AutoCancelExpired(confirmHours)
}

// GetAvailableSlots (Opción B) — retorna los horarios libres para un servicio en una fecha.
// Considera citas pending+scheduled del especialista principal como bloqueadas.
func (s *appointmentService) GetAvailableSlots(dateStr string, serviceID uint) (*domain.AvailableSlots, error) {
	// Validar y parsear fecha
	dateStr = strings.TrimSpace(dateStr)
	date, err := time.ParseInLocation("2006-01-02", dateStr, bogotaLoc)
	if err != nil {
		return nil, errors.New("fecha inválida, usa formato YYYY-MM-DD")
	}

	// No mostrar slots para fechas pasadas
	today := time.Now().In(bogotaLoc)
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, bogotaLoc)
	if date.Before(todayDate) {
		return &domain.AvailableSlots{Date: dateStr, ServiceID: serviceID, Slots: []string{}}, nil
	}

	// Domingo → cerrado
	if date.Weekday() == time.Sunday {
		return &domain.AvailableSlots{Date: dateStr, ServiceID: serviceID, Slots: []string{}}, nil
	}

	// Obtener servicio
	svc, err := s.serviceRepo.GetByID(serviceID)
	if err != nil {
		return nil, errors.New("servicio no encontrado")
	}
	svcDuration := time.Duration(svc.DurationMinutes) * time.Minute

	// Obtener especialista principal
	specialist, err := s.specialistRepo.GetDefault()
	if err != nil {
		return nil, errors.New("no hay especialista principal configurado")
	}

	// Obtener rangos ocupados para ese día
	occupied, err := s.repo.GetOccupiedRanges(specialist.ID, date)
	if err != nil {
		return nil, errors.New("error consultando disponibilidad")
	}

	// Calcular inicio y fin del día laboral en Bogotá
	y, m, d := date.Date()
	var dayStart, dayEnd time.Time
	if date.Weekday() == time.Saturday {
		dayStart = time.Date(y, m, d, 8, 0, 0, 0, bogotaLoc)
		dayEnd = time.Date(y, m, d, 12, 0, 0, 0, bogotaLoc)
	} else {
		dayStart = time.Date(y, m, d, 8, 0, 0, 0, bogotaLoc)
		dayEnd = time.Date(y, m, d, 18, 0, 0, 0, bogotaLoc)
	}

	// Intervalo entre slots (configurable, default 30 min)
	slotInterval := 30
	if v := os.Getenv("SLOT_INTERVAL_MINUTES"); v != "" {
		if n, err2 := strconv.Atoi(v); err2 == nil && n > 0 {
			slotInterval = n
		}
	}
	step := time.Duration(slotInterval) * time.Minute

	// Buffer mínimo desde ahora para el día de hoy (30 min)
	minStart := time.Now().In(bogotaLoc).Add(30 * time.Minute)

	var slots []string
	for slot := dayStart; !slot.Add(svcDuration).After(dayEnd); slot = slot.Add(step) {
		// Filtrar slots en el pasado (para hoy)
		if date.Equal(todayDate) && slot.Before(minStart) {
			continue
		}

		slotEnd := slot.Add(svcDuration)

		// Verificar solapamiento con citas existentes
		available := true
		for _, occ := range occupied {
			// Solapamiento: slot empieza antes de que termine la ocupada
			// Y termina después de que empieza la ocupada
			if slot.Before(occ[1]) && slotEnd.After(occ[0]) {
				available = false
				break
			}
		}

		if available {
			slots = append(slots, slot.Format("15:04"))
		}
	}

	if slots == nil {
		slots = []string{}
	}

	return &domain.AvailableSlots{
		Date:            dateStr,
		ServiceID:       serviceID,
		DurationMinutes: svc.DurationMinutes,
		Slots:           slots,
	}, nil
}
