package domain

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

var bogotaLoc, _ = time.LoadLocation("America/Bogota")

// BogotaTime es un tipo personalizado que siempre convierte a hora de Bogotá
type BogotaTime struct {
	time.Time
}

// UnmarshalJSON — convierte JSON a BogotaTime
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

// MarshalJSON — convierte BogotaTime a JSON sin Z (hora local)
func (bt BogotaTime) MarshalJSON() ([]byte, error) {
	t := bt.Time.In(bogotaLoc)
	return []byte(`"` + t.Format("2006-01-02T15:04:05-05:00") + `"`), nil
}

// Scan — permite a GORM leer BogotaTime desde la BD ← ESTO FALTABA
func (bt *BogotaTime) Scan(value interface{}) error {
	if value == nil {
		bt.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		// El DB devuelve TIMESTAMP sin zona horaria como UTC
		// pero el valor YA ES hora de Bogotá, solo hay que reinterpretarlo
		bt.Time = time.Date(
			v.Year(), v.Month(), v.Day(),
			v.Hour(), v.Minute(), v.Second(), v.Nanosecond(),
			bogotaLoc, // ← Reinterpretar como Bogotá, no convertir
		)
		return nil
	case []byte:
		t, err := time.ParseInLocation("2006-01-02 15:04:05", string(v), bogotaLoc)
		if err != nil {
			return err
		}
		bt.Time = t
		return nil
	case string:
		t, err := time.ParseInLocation("2006-01-02 15:04:05", v, bogotaLoc)
		if err != nil {
			return err
		}
		bt.Time = t
		return nil
	}

	return fmt.Errorf("no se puede convertir %T a BogotaTime", value)
}

// Value — permite a GORM escribir BogotaTime en la BD ← ESTO FALTABA
func (bt BogotaTime) Value() (driver.Value, error) {
	if bt.Time.IsZero() {
		return nil, nil
	}
	return bt.Time.In(bogotaLoc), nil
}

type Appointment struct {
	ID                 uint         `json:"id"`
	PatientID          uint         `json:"patient_id"`
	Patient            Patient      `json:"patient" gorm:"foreignKey:PatientID"`
	SpecialistID       uint         `json:"specialist_id"`
	Specialist         Specialist   `json:"specialist" gorm:"foreignKey:SpecialistID"`
	ServiceID          uint         `json:"service_id"`
	Service            Service      `json:"service" gorm:"foreignKey:ServiceID"`
	StartTime          BogotaTime   `json:"start_time"`
	EndTime            BogotaTime   `json:"end_time"`
	Status             string       `json:"status"`
	HistoricalPrice    float64      `json:"historical_price"`
	ModificationCount  int          `json:"modification_count"`
	Notes              string       `json:"notes"`
	CancellationReason string       `json:"cancellation_reason"`
	CancellationNotes  string       `json:"cancellation_notes"`
	DataConsent        *DataConsent `json:"data_consent,omitempty" gorm:"foreignKey:AppointmentID"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

func (Appointment) TableName() string {
	return "\"Appointments\""
}

// PublicAppointmentRequest es el payload del endpoint público de la landing page.
// Nunca acepta status, specialist_id ni campos administrativos.
type PublicAppointmentRequest struct {
	// Datos del paciente
	DocumentNumber string `json:"document_number" binding:"required"`
	FirstName      string `json:"first_name"      binding:"required"`
	LastName       string `json:"last_name"       binding:"required"`
	Phone          string `json:"phone"           binding:"required"`
	Email          string `json:"email"`
	BirthDate      string `json:"birth_date"`
	// Datos de la cita
	ServiceID uint       `json:"service_id" binding:"required"`
	StartTime BogotaTime `json:"start_time"  binding:"required"`
	Notes     string     `json:"notes"`
	// Consentimiento Ley 1581 — obligatorio
	DatosAceptados   bool      `json:"datos_aceptados"`
	DatosAceptadosAt time.Time `json:"datos_aceptados_at"`
	// Metadatos HTTP — NO vienen del JSON, los inyecta el handler
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

// AvailableSlots es la respuesta del endpoint público de slots disponibles.
type AvailableSlots struct {
	Date            string   `json:"date"`             // "2026-05-04"
	ServiceID       uint     `json:"service_id"`
	DurationMinutes int      `json:"duration_minutes"`
	Slots           []string `json:"slots"`            // ["08:00","08:30",...]
}

type AdminUpdateRequest struct {
	Status             string      `json:"status"`
	SpecialistID       *uint       `json:"specialist_id"`
	ServiceID          *uint       `json:"service_id"`
	StartTime          *BogotaTime `json:"start_time"`
	EndTime            *BogotaTime `json:"end_time"`
	CancellationReason string      `json:"cancellation_reason"`
	CancellationNotes  string      `json:"cancellation_notes"`
}
