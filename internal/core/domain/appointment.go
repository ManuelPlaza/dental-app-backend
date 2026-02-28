package domain

import "time"

type Appointment struct {
	ID uint `json:"id"`

	PatientID          uint       `json:"patient_id"`
	Patient            Patient    `json:"patient" gorm:"foreignKey:PatientID"`
	SpecialistID       uint       `json:"specialist_id"`
	Specialist         Specialist `json:"specialist" gorm:"foreignKey:SpecialistID"`
	ServiceID          uint       `json:"service_id"`
	StartTime          time.Time  `json:"start_time"`
	EndTime            time.Time  `json:"end_time"`
	Status             string     `json:"status"`
	HistoricalPrice    float64    `json:"historical_price"`
	ModificationCount  int        `json:"modification_count"`
	Notes              string     `json:"notes"`
	CancellationReason string     `json:"cancellation_reason"` // <--- NUEVO
	CancellationNotes  string     `json:"cancellation_notes"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// AdminUpdateRequest estructura para actualización admin sin restricciones
type AdminUpdateRequest struct {
	Status             string     `json:"status"`
	SpecialistID       *uint      `json:"specialist_id"`
	ServiceID          *uint      `json:"service_id"`
	StartTime          *time.Time `json:"start_time"`
	EndTime            *time.Time `json:"end_time"`
	CancellationReason string     `json:"cancellation_reason"` // <--- NUEVO
	CancellationNotes  string     `json:"cancellation_notes"`  // <--- NUEVO
}

func (Appointment) TableName() string {
	return "\"Appointments\""
}
