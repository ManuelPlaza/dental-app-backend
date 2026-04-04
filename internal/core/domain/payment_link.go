package domain

import "time"

type PaymentLinkStatus string

const (
	PaymentLinkPending   PaymentLinkStatus = "pending"
	PaymentLinkPaid      PaymentLinkStatus = "paid"
	PaymentLinkExpired   PaymentLinkStatus = "expired"
	PaymentLinkRejected  PaymentLinkStatus = "rejected"
	PaymentLinkCancelled PaymentLinkStatus = "cancelled"
)

type PaymentLink struct {
	ID             uint              `json:"id"`
	AppointmentID  uint              `json:"appointment_id"`
	Appointment    Appointment       `json:"appointment,omitempty" gorm:"foreignKey:AppointmentID"`
	Amount         float64           `json:"amount"`
	Token          string            `json:"token"`
	NequiTxnID     string            `json:"nequi_txn_id,omitempty"`
	Status         PaymentLinkStatus `json:"status"`
	PhoneNumber    string            `json:"phone_number"`
	ExpiresAt      time.Time         `json:"expires_at"`
	PaidAt         *time.Time        `json:"paid_at,omitempty"`
	CreatedBy      uint              `json:"created_by"`
	WebhookPayload string            `json:"-"` // solo para auditoría interna
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (PaymentLink) TableName() string {
	return "\"PaymentLinks\""
}

// GeneratePaymentLinkRequest payload que envía el admin para crear un link
type GeneratePaymentLinkRequest struct {
	Amount      float64 `json:"amount"       binding:"required,gt=0"`
	PhoneNumber string  `json:"phone_number" binding:"required"`
}
