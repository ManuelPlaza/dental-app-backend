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
func (r *gormPaymentRepo) GetByID(id uint) (*domain.Payment, error) {
	var payment domain.Payment
	err := r.db.Table("\"Payments\"").First(&payment, id).Error
	return &payment, err
}

func (r *gormPaymentRepo) Update(payment *domain.Payment) error {
	return r.db.Table("\"Payments\"").Save(payment).Error
}

// GetMonthlyIncome trae ingresos agrupados por mes (últimos 6 meses)
func (r *gormPaymentRepo) GetMonthlyIncome() ([]map[string]interface{}, error) {
	var results []struct {
		Month string
		Total float64
	}

	err := r.db.Table("\"Payments\"").
		Select("TO_CHAR(DATE_TRUNC('month', payment_date), 'YYYY-MM') as month, SUM(amount) as total").
		Where("payment_date >= NOW() - INTERVAL '6 months'").
		Where("amount > 0").
		Group("month").
		Order("month ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(results))
	for i, r := range results {
		data[i] = map[string]interface{}{
			"month": r.Month,
			"total": r.Total,
		}
	}
	return data, nil
}

// GetRecent trae los últimos N pagos con info del paciente y servicio
func (r *gormPaymentRepo) GetRecent(limit int) ([]domain.Payment, error) {
	var payments []domain.Payment
	err := r.db.Table("\"Payments\"").
		Preload("Appointment").
		Preload("Appointment.Patient").
		Preload("Appointment.Specialist").
		Where("amount > 0").
		Order("payment_date DESC").
		Limit(limit).
		Find(&payments).Error
	return payments, err
}
