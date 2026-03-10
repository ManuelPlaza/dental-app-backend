package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"time"

	"gorm.io/gorm"
)

type gormNotificationRepo struct {
	db *gorm.DB
}

func NewGormNotificationRepo(db *gorm.DB) ports.NotificationRepository {
	return &gormNotificationRepo{db: db}
}

func (r *gormNotificationRepo) Save(n *domain.NotificationQueue) error {
	return r.db.Table("\"NotificationQueue\"").Create(n).Error
}

func (r *gormNotificationRepo) GetPending() ([]domain.NotificationQueue, error) {
	var notifications []domain.NotificationQueue
	err := r.db.Table("\"NotificationQueue\"").
		Preload("Appointment").
		Preload("Appointment.Patient").
		Preload("Appointment.Specialist").
		Where("status = ?", domain.NotificationPending).
		Where("scheduled_at <= ?", time.Now()).
		Where("attempts < max_attempts").
		Find(&notifications).Error
	return notifications, err
}

func (r *gormNotificationRepo) Update(n *domain.NotificationQueue) error {
	return r.db.Table("\"NotificationQueue\"").Save(n).Error
}

func (r *gormNotificationRepo) FindByToken(token string) (*domain.NotificationQueue, error) {
	var n domain.NotificationQueue
	err := r.db.Table("\"NotificationQueue\"").
		Preload("Appointment").
		Preload("Appointment.Patient").
		Where("token = ?", token).
		First(&n).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *gormNotificationRepo) FindByAppointmentAndType(appointmentID uint, nType domain.NotificationType) (*domain.NotificationQueue, error) {
	var n domain.NotificationQueue
	err := r.db.Table("\"NotificationQueue\"").
		Where("appointment_id = ? AND type = ?", appointmentID, nType).
		First(&n).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *gormNotificationRepo) SaveLog(l *domain.NotificationLog) error {
	return r.db.Table("\"NotificationLogs\"").Create(l).Error
}
