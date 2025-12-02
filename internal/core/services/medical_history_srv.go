package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
	"time"
)

type medicalHistoryService struct {
	repo ports.MedicalHistoryRepository
}

func NewMedicalHistoryService(repo ports.MedicalHistoryRepository) ports.MedicalHistoryService {
	return &medicalHistoryService{repo: repo}
}

func (s *medicalHistoryService) Create(history *domain.MedicalHistory) error {
	// Validación Médica Básica
	if history.Diagnosis == "" {
		return errors.New("el campo diagnóstico es obligatorio para la historia clínica")
	}
	if history.Treatment == "" {
		return errors.New("debe especificar el tratamiento realizado")
	}

	if history.CreatedAt.IsZero() {
		history.CreatedAt = time.Now()
	}

	return s.repo.Save(history)
}

func (s *medicalHistoryService) GetHistoryByPatient(patientID uint) ([]domain.MedicalHistory, error) {
	return s.repo.GetByPatientID(patientID)
}
