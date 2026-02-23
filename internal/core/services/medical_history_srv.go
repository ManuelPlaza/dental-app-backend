package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"time"
)

type medicalHistoryService struct {
	repo            ports.MedicalHistoryRepository
	appointmentRepo ports.AppointmentRepository // <--- NUEVO
}

func NewMedicalHistoryService(repo ports.MedicalHistoryRepository, appointmentRepo ports.AppointmentRepository) ports.MedicalHistoryService {
	return &medicalHistoryService{repo: repo, appointmentRepo: appointmentRepo}
}

func (s *medicalHistoryService) Create(history *domain.MedicalHistory) error {
	// Validación: la cita debe existir y estar completada
	app, err := s.appointmentRepo.GetByID(history.AppointmentID)
	if err != nil {
		return errors.New("cita no encontrada")
	}
	if app.Status != "completed" {
		return errors.New("solo se puede crear historia clínica para citas completadas")
	}

	// Tomar el patient_id de la cita automáticamente
	history.PatientID = app.PatientID

	// Validaciones médicas
	if history.Diagnosis == "" {
		return errors.New("el campo diagnóstico es obligatorio")
	}
	if history.Treatment == "" {
		return errors.New("debe especificar el tratamiento realizado")
	}

	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now()
	}

	// Guardar la historia clínica
	if err := s.repo.Save(history); err != nil {
		return err
	}

	// Si viene próxima cita → crearla automáticamente
	if history.NextAppointmentDate != nil {
		nextApp := &domain.Appointment{
			PatientID:    app.PatientID,
			SpecialistID: app.SpecialistID,
			ServiceID:    app.ServiceID,
			StartTime:    *history.NextAppointmentDate,
			EndTime:      history.NextAppointmentDate.Add(time.Duration(60) * time.Minute),
			Status:       "pending",
			Notes:        "Cita generada automáticamente desde historia clínica",
		}
		s.appointmentRepo.Save(nextApp)
	}

	return nil
}

func (s *medicalHistoryService) GetHistoryByPatient(patientID uint) ([]domain.MedicalHistory, error) {
	return s.repo.GetByPatientID(patientID)
}

func (s *medicalHistoryService) GetAll() ([]domain.MedicalHistory, error) {
	return s.repo.GetAll()
}
