package ports

import (
	"dental-app/internal/core/domain"
	"time"
)

// PatientService define la lógica de negocio disponible para el mundo exterior.
type PatientService interface {
	Create(patient *domain.Patient) error
	List() ([]domain.Patient, error)
	FindByDocument(documentNumber string) (*domain.Patient, error) // <--- NUEVO
}

type AppointmentService interface {
	Schedule(appointment *domain.Appointment) error
	Modify(id uint, newStart, newEnd time.Time) error
	Cancel(id uint) error
	List() ([]domain.Appointment, error)
	AdminUpdateStatus(id uint, status string) error
	ListPaginated(page, limit int, status string) ([]domain.Appointment, int64, error) // <--- status agregado
	GetSummary() (map[string]int64, error)                                             // <--- NUEVO
	AdminUpdate(id uint, req domain.AdminUpdateRequest) error                          // <--- NUEVO
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

type SpecialistService interface {
	Create(s *domain.Specialist) error
	List() ([]domain.Specialist, error)
	Inactivate(id uint) error
}

type ServiceService interface {
	List() ([]domain.Service, error)
}
