package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"time"

	"gorm.io/gorm"
)

type gormPaymentLinkRepo struct {
	db *gorm.DB
}

func NewGormPaymentLinkRepo(db *gorm.DB) ports.PaymentLinkRepository {
	return &gormPaymentLinkRepo{db: db}
}

func (r *gormPaymentLinkRepo) Save(link *domain.PaymentLink) error {
	return r.db.Table(`"PaymentLinks"`).Create(link).Error
}

func (r *gormPaymentLinkRepo) GetByToken(token string) (*domain.PaymentLink, error) {
	var link domain.PaymentLink
	err := r.db.Table(`"PaymentLinks"`).
		Preload("Appointment").
		Preload("Appointment.Patient").
		Preload("Appointment.Service").
		Where("token = ?", token).
		First(&link).Error
	return &link, err
}

func (r *gormPaymentLinkRepo) GetByNequiTxnID(txnID string) (*domain.PaymentLink, error) {
	var link domain.PaymentLink
	err := r.db.Table(`"PaymentLinks"`).
		Where("nequi_txn_id = ?", txnID).
		First(&link).Error
	return &link, err
}

func (r *gormPaymentLinkRepo) GetByAppointmentID(appointmentID uint) ([]domain.PaymentLink, error) {
	var links []domain.PaymentLink
	err := r.db.Table(`"PaymentLinks"`).
		Where("appointment_id = ?", appointmentID).
		Order("created_at DESC").
		Find(&links).Error
	return links, err
}

func (r *gormPaymentLinkRepo) UpdateStatus(id uint, status domain.PaymentLinkStatus, paidAt *time.Time, webhookPayload string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	if webhookPayload != "" {
		updates["webhook_payload"] = webhookPayload
	}
	return r.db.Table(`"PaymentLinks"`).Where("id = ?", id).Updates(updates).Error
}

func (r *gormPaymentLinkRepo) UpdateNequiTxnID(id uint, txnID string) error {
	return r.db.Table(`"PaymentLinks"`).Where("id = ?", id).
		Updates(map[string]interface{}{
			"nequi_txn_id": txnID,
			"updated_at":   time.Now(),
		}).Error
}

func (r *gormPaymentLinkRepo) CancelPendingByAppointment(appointmentID uint) error {
	return r.db.Table(`"PaymentLinks"`).
		Where("appointment_id = ? AND status = ?", appointmentID, domain.PaymentLinkPending).
		Updates(map[string]interface{}{
			"status":     domain.PaymentLinkCancelled,
			"updated_at": time.Now(),
		}).Error
}

func (r *gormPaymentLinkRepo) ExpireStale() error {
	return r.db.Table(`"PaymentLinks"`).
		Where("status = ? AND expires_at < ?", domain.PaymentLinkPending, time.Now()).
		Updates(map[string]interface{}{
			"status":     domain.PaymentLinkExpired,
			"updated_at": time.Now(),
		}).Error
}
