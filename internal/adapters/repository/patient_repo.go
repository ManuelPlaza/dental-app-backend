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
