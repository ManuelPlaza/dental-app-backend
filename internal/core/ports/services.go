package ports

import "dental-app/internal/core/domain"

// PatientService define la lógica de negocio disponible para el mundo exterior.
type PatientService interface {
	Create(patient *domain.Patient) error
	List() ([]domain.Patient, error)
}