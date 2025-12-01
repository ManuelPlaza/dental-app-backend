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
