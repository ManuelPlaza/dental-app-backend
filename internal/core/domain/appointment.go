package domain

import (
	"strings"
	"time"
)

// BogotaTime es un tipo personalizado que siempre convierte a hora de Bogotá
type BogotaTime struct {
	time.Time
}

var bogotaLoc, _ = time.LoadLocation("America/Bogota")

func (bt *BogotaTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "null" || s == "" {
		return nil
	}

	// Intentar parsear con zona horaria (ISO 8601 con Z o offset)
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Intentar parsear sin zona horaria (naive datetime)
		t, err = time.ParseInLocation("2006-01-02T15:04:05", s, bogotaLoc)
		if err != nil {
			return err
		}
		bt.Time = t
		return nil
	}

	// Convertir a hora de Bogotá
	bt.Time = t.In(bogotaLoc)
	return nil
}

func (bt BogotaTime) MarshalJSON() ([]byte, error) {
	t := bt.Time.In(bogotaLoc)
	return []byte(`"` + t.Format("2006-01-02T15:04:05") + `"`), nil
}

type Appointment struct {
	ID                 uint       `json:"id"`
	PatientID          uint       `json:"patient_id"`
	Patient            Patient    `json:"patient" gorm:"foreignKey:PatientID"`
	SpecialistID       uint       `json:"specialist_id"`
	Specialist         Specialist `json:"specialist" gorm:"foreignKey:SpecialistID"`
	ServiceID          uint       `json:"service_id"`
	StartTime          BogotaTime `json:"start_time"`
	EndTime            BogotaTime `json:"end_time"`
	Status             string     `json:"status"`
	HistoricalPrice    float64    `json:"historical_price"`
	ModificationCount  int        `json:"modification_count"`
	Notes              string     `json:"notes"`
	CancellationReason string     `json:"cancellation_reason"`
	CancellationNotes  string     `json:"cancellation_notes"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// AdminUpdateRequest estructura para actualización admin sin restricciones
type AdminUpdateRequest struct {
	Status             string      `json:"status"`
	SpecialistID       *uint       `json:"specialist_id"`
	ServiceID          *uint       `json:"service_id"`
	StartTime          *BogotaTime `json:"start_time"` // ← *BogotaTime no *time.Time
	EndTime            *BogotaTime `json:"end_time"`   // ← *BogotaTime no *time.Time
	CancellationReason string      `json:"cancellation_reason"`
	CancellationNotes  string      `json:"cancellation_notes"`
}

func (Appointment) TableName() string {
	return "\"Appointments\""
}
