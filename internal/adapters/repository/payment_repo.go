package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormPaymentRepo struct {
	db *gorm.DB
}

func NewGormPaymentRepo(db *gorm.DB) ports.PaymentRepository {
	return &gormPaymentRepo{db: db}
}

func (r *gormPaymentRepo) Save(payment *domain.Payment) error {
	// Usamos Table("\"Payments\"") para respetar el nombre de DrawSQL
	return r.db.Table("\"Payments\"").Create(payment).Error
}
func (r *gormPaymentRepo) GetAll() ([]domain.Payment, error) {
	var payments []domain.Payment
	// Esto trae el Pago + La Cita + El Paciente de esa cita (Join doble)
	err := r.db.Table("\"Payments\"").
		Preload("Appointment").
		Preload("Appointment.Patient").
		Find(&payments).Error
	return payments, err
}
func (r *gormPaymentRepo) GetByAppointmentID(appID uint) ([]domain.Payment, error) {
	var payments []domain.Payment
	err := r.db.Table("\"Payments\"").Where("appointment_id = ?", appID).Find(&payments).Error
	return payments, err
}
