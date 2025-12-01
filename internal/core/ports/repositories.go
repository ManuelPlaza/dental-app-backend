package ports

import "dental-app/internal/core/domain"

// PatientRepository define qué se puede hacer con la base de datos
type PatientRepository interface {
	Save(patient *domain.Patient) error
	GetAll() ([]domain.Patient, error)
}
