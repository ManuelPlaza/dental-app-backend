package domain

import "time"

type Appointment struct {
	ID uint `json:"id"`

	PatientID uint    `json:"patient_id"`
	Patient   Patient `json:"patient" gorm:"foreignKey:PatientID"` // <--- Mágia aquí

	SpecialistID uint `json:"specialist_id"`
	// Specialist     Specialist `json:"specialist" gorm:"foreignKey:SpecialistID"` (Descomentar cuando tengas el struct Specialist)

	ServiceID uint `json:"service_id"`
	// Service        Service    `json:"service" gorm:"foreignKey:ServiceID"` (Descomentar cuando tengas el struct Service)

	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Status            string    `json:"status"`
	HistoricalPrice   float64   `json:"historical_price"`
	ModificationCount int       `json:"modification_count"`
	CreatedAt         time.Time `json:"created_at"`
}

// ... (debajo del struct Appointment)

func (Appointment) TableName() string {
	return "\"Appointments\""
}
