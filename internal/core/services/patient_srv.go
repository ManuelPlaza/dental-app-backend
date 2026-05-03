package services

import (
	"crypto/sha256"
	"fmt"
	"time"

	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
)

const retentionYears = 2

type patientService struct {
	repo        ports.PatientRepository
	consentRepo ports.DataConsentRepository
}

func NewPatientService(repo ports.PatientRepository, consentRepo ports.DataConsentRepository) ports.PatientService {
	return &patientService{repo: repo, consentRepo: consentRepo}
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

func (s *patientService) FindByDocument(documentNumber string) (*domain.Patient, error) {
	return s.repo.FindByDocumentNumber(documentNumber)
}

func (s *patientService) Update(id uint, patient *domain.Patient) error {
	return s.repo.Update(id, patient)
}

// GetConsentStatus retorna el estado de consentimiento y retención del paciente.
func (s *patientService) GetConsentStatus(patientID uint) (*domain.ConsentStatus, error) {
	patient, err := s.repo.FindByID(patientID)
	if err != nil {
		return nil, fmt.Errorf("paciente no encontrado")
	}

	status := &domain.ConsentStatus{
		IsAnonymized: patient.AnonymizedAt != nil,
	}
	if status.IsAnonymized {
		return status, nil
	}

	// Verificar consentimiento registrado
	consent, _ := s.consentRepo.FindByDocument(patient.DocumentNumber)
	if consent != nil {
		status.HasConsent = true
		t := consent.AceptadoAt
		status.ConsentDate = &t
	}

	// Último contacto — última cita no cancelada, o fecha de registro como fallback
	lastContact, err := s.repo.GetLastContactDate(patientID)
	if err != nil {
		return nil, err
	}
	if lastContact == nil {
		lastContact = &patient.CreatedAt
	}
	status.LastContactDate = lastContact

	// Expiración según política de privacidad: 2 años desde último contacto
	expiry := lastContact.AddDate(retentionYears, 0, 0)
	status.ExpiresAt = &expiry
	days := int(time.Until(expiry).Hours() / 24)
	status.DaysUntilExpiry = &days

	return status, nil
}

// RevokeConsent elimina el consentimiento del paciente (Ley 1581 art. 8 lit. a).
func (s *patientService) RevokeConsent(patientID uint) error {
	patient, err := s.repo.FindByID(patientID)
	if err != nil {
		return fmt.Errorf("paciente no encontrado")
	}
	return s.consentRepo.RevokeByDocument(patient.DocumentNumber)
}

// AnonymizeExpired anonimiza pacientes cuyo período de retención de 2 años venció.
func (s *patientService) AnonymizeExpired() (int64, error) {
	cutoff := time.Now().AddDate(-retentionYears, 0, 0)
	patients, err := s.repo.GetExpiringPatients(cutoff)
	if err != nil {
		return 0, err
	}

	var count int64
	for _, p := range patients {
		// SHA-256 del documento original para mantener unicidad sin revelar PII
		h := sha256.Sum256([]byte(p.DocumentNumber))
		anonDoc := fmt.Sprintf("ANON-%x", h[:6])

		if err := s.repo.AnonymizePatient(p.ID, anonDoc); err != nil {
			continue
		}
		_ = s.consentRepo.RevokeByDocument(p.DocumentNumber)
		count++
	}
	return count, nil
}
