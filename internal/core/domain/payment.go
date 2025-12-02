package domain

import "time"

type Payment struct {
	ID            uint        `json:"id"`
	AppointmentID uint        `json:"appointment_id"`
	Appointment   Appointment `json:"appointment" gorm:"foreignKey:AppointmentID"` // <--- NUEVO
	Amount        float64     `json:"amount"`
	Method        string      `json:"method"` // cash, nequi, loyalty_points
	ReferenceCode string      `json:"reference_code"`
	Notes         string      `json:"notes"`
	PaymentDate   time.Time   `json:"payment_date"`
}

// TableName le dice a GORM que use la tabla "Payments" (con comillas)
func (Payment) TableName() string {
	return "\"Payments\""
}
