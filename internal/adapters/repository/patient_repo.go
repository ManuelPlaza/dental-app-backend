package repository

import (
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

// Save ahora especifica la tabla exacta "Patient"
func (r *gormPatientRepo) Save(patient *domain.Patient) error {
	return r.db.Table("\"Patient\"").Create(patient).Error
}

// GetAll también busca en "Patient"
func (r *gormPatientRepo) GetAll() ([]domain.Patient, error) {
	var patients []domain.Patient
	err := r.db.Table("\"Patient\"").Find(&patients).Error
	return patients, err
}
