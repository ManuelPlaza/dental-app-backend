package repository

import (
	"strings"

	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormPatientRepo struct {
	db *gorm.DB
}

func NewGormPatientRepo(db *gorm.DB) ports.PatientRepository {
	return &gormPatientRepo{db: db}
}

// Save maneja duplicados de document_number
func (r *gormPatientRepo) Save(patient *domain.Patient) error {
	err := r.db.Table("\"Patient\"").Create(patient).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return domain.ErrPatientAlreadyExists
		}
		return err
	}
	return nil
}

// GetAll obtiene todos los pacientes
func (r *gormPatientRepo) GetAll() ([]domain.Patient, error) {
	var patients []domain.Patient
	err := r.db.Table("\"Patient\"").Find(&patients).Error
	return patients, err
}

// FindByDocumentNumber busca un paciente por número de documento
func (r *gormPatientRepo) FindByDocumentNumber(doc string) (*domain.Patient, error) {
	var patient domain.Patient
	err := r.db.Table("\"Patient\"").Where("document_number = ?", doc).First(&patient).Error
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

// Update actualiza solo los campos permitidos del paciente
func (r *gormPatientRepo) Update(id uint, patient *domain.Patient) error {
	return r.db.Table("\"Patient\"").Where("id = ?", id).Updates(map[string]interface{}{
		"phone":                          patient.Phone,
		"email":                          patient.Email,
		"emergency_contact_name":         patient.EmergencyContactName,
		"emergency_contact_relationship": patient.EmergencyContactRelationship,
		"emergency_contact_phone":        patient.EmergencyContactPhone,
	}).Error
}
func (r *gormPatientRepo) FindByID(id uint) (*domain.Patient, error) {
	var patient domain.Patient
	err := r.db.Table("\"Patient\"").First(&patient, id).Error
	if err != nil {
		return nil, err
	}
	return &patient, nil
}
