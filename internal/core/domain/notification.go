package domain

import "time"

type NotificationType string
type NotificationStatus string

const (
	NotificationConfirmation  NotificationType = "CONFIRMATION_REQUEST"
	NotificationReminder24H   NotificationType = "REMINDER_24H"
	NotificationReminderFinal NotificationType = "REMINDER_FINAL"

	NotificationPending NotificationStatus = "pending"
	NotificationSent    NotificationStatus = "sent"
	NotificationFailed  NotificationStatus = "failed"
	NotificationSkipped NotificationStatus = "skipped"
)

type NotificationQueue struct {
	ID            uint               `json:"id"`
	AppointmentID uint               `json:"appointment_id"`
	Appointment   Appointment        `json:"appointment" gorm:"foreignKey:AppointmentID"`
	PatientID     uint               `json:"patient_id"`
	Patient       Patient            `json:"patient" gorm:"foreignKey:PatientID"`
	Type          NotificationType   `json:"type"`
	Status        NotificationStatus `json:"status"`
	ScheduledAt   time.Time          `json:"scheduled_at"`
	SentAt        *time.Time         `json:"sent_at"`
	Attempts      int                `json:"attempts"`
	MaxAttempts   int                `json:"max_attempts"`
	Token         string             `json:"token"`
	ErrorLog      string             `json:"error_log"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func (NotificationQueue) TableName() string {
	return "\"NotificationQueue\""
}

type NotificationLog struct {
	ID             uint      `json:"id"`
	NotificationID uint      `json:"notification_id"`
	Event          string    `json:"event"`
	Detail         string    `json:"detail"`
	CreatedAt      time.Time `json:"created_at"`
}

func (NotificationLog) TableName() string {
	return "\"NotificationLogs\""
}
