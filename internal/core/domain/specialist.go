package domain

import "time"

type Specialist struct {
	ID            uint      `json:"id"`
	UserID        *uint     `json:"user_id"` // ← *uint permite NULL
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Specialty     string    `json:"specialty"`
	LicenseNumber string    `json:"license_number"`
	Phone         string    `json:"phone"`
	IsActive      bool      `json:"is_active"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName le dice a GORM que busque en la tabla "Specialists" (Plural y comillas)
func (Specialist) TableName() string {
	return "\"Specialists\""
}
