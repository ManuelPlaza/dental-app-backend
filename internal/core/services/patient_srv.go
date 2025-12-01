package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
)

type patientService struct{ repo ports.PatientRepository }

func NewPatientService(repo ports.PatientRepository) ports.PatientService {
	return &patientService{repo: repo}
}
func (s *patientService) Create(patient *domain.Patient) error {
	if patient.DocumentNumber == "" {
		return errors.New("documento obligatorio")
	}
	return s.repo.Save(patient)
}
func (s *patientService) List() ([]domain.Patient, error) {
	return s.repo.GetAll()
}
