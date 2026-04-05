package domain

import "time"

// DataConsent registra la aceptación explícita de tratamiento de datos personales
// según la Ley 1581 de 2012 (Colombia). Es una entidad legal independiente e inmutable.
type DataConsent struct {
	ID               uint      `json:"id"`
	AppointmentID    uint      `json:"appointment_id"    gorm:"uniqueIndex;not null"`
	DocumentoTitular string    `json:"documento_titular" gorm:"type:varchar(20);not null"`
	Aceptado         bool      `json:"aceptado"          gorm:"not null;default:false"`
	AceptadoAt       time.Time `json:"aceptado_at"       gorm:"type:timestamptz;not null"`
	RegistradoAt     time.Time `json:"registrado_at"     gorm:"type:timestamptz;not null"`
	IPAddress        string    `json:"ip_address"        gorm:"type:varchar(45);not null"`
	VersionPolitica  string    `json:"version_politica"  gorm:"type:varchar(20);not null;default:'1.0'"`
	UserAgent        string    `json:"user_agent"        gorm:"type:text"`
	CreatedAt        time.Time `json:"created_at"`
}

func (DataConsent) TableName() string {
	return "\"DataConsents\""
}
