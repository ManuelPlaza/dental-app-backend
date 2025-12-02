package ports

import (
	"dental-app/internal/core/domain"
	"time"
)

// PatientService define la lógica de negocio disponible para el mundo exterior.
type PatientService interface {
	Create(patient *domain.Patient) error
	List() ([]domain.Patient, error)
}

type AppointmentService interface {
	Schedule(appointment *domain.Appointment) error
	Modify(id uint, newStart, newEnd time.Time) error
	Cancel(id uint) error
	List() ([]domain.Appointment, error) // <--- ¡ESTA FALTABA!
}

// ... (interfaces anteriores)

type PaymentService interface {
	Process(payment *domain.Payment) error
	List() ([]domain.Payment, error) // <--- NUEVO
	GetBalance(appID uint) (float64, float64, float64, error)
}
type MedicalHistoryService interface {
	Create(history *domain.MedicalHistory) error
	GetHistoryByPatient(patientID uint) ([]domain.MedicalHistory, error)
}