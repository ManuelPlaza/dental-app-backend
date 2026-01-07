package domain

import "time"

type Patient struct {
	ID                           uint      `json:"id"`
	FirstName                    string    `json:"first_name"`
	LastName                     string    `json:"last_name"`
	DocumentNumber               string    `json:"document_number"`
	Phone                        string    `json:"phone"`
	EmergencyContactName         string    `json:"emergency_contact_name"`
	EmergencyContactRelationship string    `json:"emergency_contact_relationship"`
	EmergencyContactPhone        string    `json:"emergency_contact_phone"`
	ReferredByID                 *uint     `json:"referred_by_id,omitempty"`
	ReferredBy                   *Patient  `json:"referred_by,omitempty"`
	UserID                       *uint     `json:"user_id,omitempty"`
	CreatedAt                    time.Time `json:"created_at"`
}

// ... (debajo del struct Patient)

// TableName le dice a GORM el nombre exacto de la tabla en PostgreSQL
func (Patient) TableName() string {
	return "\"Patient\""
}
