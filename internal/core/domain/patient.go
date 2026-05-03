package domain

import "time"

type Patient struct {
	ID                           uint       `json:"id"`
	FirstName                    string     `json:"first_name"`
	LastName                     string     `json:"last_name"`
	DocumentNumber               string     `json:"document_number"`
	Email                        string     `json:"email" gorm:"column:email"`
	Phone                        string     `json:"phone"`
	EmergencyContactName         string     `json:"emergency_contact_name"`
	EmergencyContactRelationship string     `json:"emergency_contact_relationship"`
	EmergencyContactPhone        string     `json:"emergency_contact_phone"`
	ReferredByID                 *uint      `json:"referred_by_id,omitempty"`
	ReferredBy                   *Patient   `json:"referred_by,omitempty"`
	UserID                       *uint      `json:"user_id,omitempty"`
	AnonymizedAt                 *time.Time `json:"anonymized_at,omitempty" gorm:"column:anonymized_at"`
	CreatedAt                    time.Time  `json:"created_at"`
}

// ConsentStatus resume el estado de consentimiento y retención de datos de un paciente.
type ConsentStatus struct {
	HasConsent      bool       `json:"has_consent"`
	ConsentDate     *time.Time `json:"consent_date,omitempty"`
	LastContactDate *time.Time `json:"last_contact_date,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	DaysUntilExpiry *int       `json:"days_until_expiry,omitempty"`
	IsAnonymized    bool       `json:"is_anonymized"`
}

// ... (debajo del struct Patient)

// TableName le dice a GORM el nombre exacto de la tabla en PostgreSQL
func (Patient) TableName() string {
	return "\"Patient\""
}
