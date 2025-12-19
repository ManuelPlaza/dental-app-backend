package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
)

type patientService struct{ repo ports.PatientRepository }

func NewPatientService(repo ports.PatientRepository) ports.PatientService {
	return &patientService{repo: repo}
}
func (s *patientService) Create(patient *domain.Patient) error {
	if patient.DocumentNumber == "" {
		return domain.ErrDocumentRequired
	}
	return s.repo.Save(patient)
}
func (s *patientService) List() ([]domain.Patient, error) {
	return s.repo.GetAll()
}
