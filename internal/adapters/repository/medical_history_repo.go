package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormMedicalHistoryRepo struct {
	db *gorm.DB
}

func NewGormMedicalHistoryRepo(db *gorm.DB) ports.MedicalHistoryRepository {
	return &gormMedicalHistoryRepo{db: db}
}

func (r *gormMedicalHistoryRepo) Save(history *domain.MedicalHistory) error {
	return r.db.Table("\"MedicalHistory\"").Create(history).Error
}

func (r *gormMedicalHistoryRepo) GetByPatientID(patientID uint) ([]domain.MedicalHistory, error) {
	var histories []domain.MedicalHistory

	// PREPARANDO DATOS PARA EL PDF:
	// Traemos la Historia -> La Cita -> El Especialista -> El Usuario (Nombre del doctor)
	err := r.db.Table("\"MedicalHistory\"").
		Preload("Appointment").
		Preload("Appointment.Patient").
		Preload("Appointment.Specialist").
		Preload("Appointment.Service").
		Where("patient_id = ?", patientID).
		Order("created_at desc"). // Lo más reciente primero
		Find(&histories).Error

	return histories, err
}

// GetAll retorna todos los historiales ordenados del más reciente al más viejo
func (r *gormMedicalHistoryRepo) GetAll() ([]domain.MedicalHistory, error) {
	var histories []domain.MedicalHistory
	err := r.db.Table("\"MedicalHistory\"").
		Preload("Appointment").
		Preload("Appointment.Patient").
		Preload("Appointment.Specialist").
		Preload("Appointment.Service").
		Order("created_at DESC").
		Find(&histories).Error
	return histories, err
}
