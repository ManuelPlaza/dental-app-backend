package domain

type Specialist struct {
	ID            uint   `json:"id"`
	UserID        uint   `json:"user_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Specialty     string `json:"specialty"`
	LicenseNumber string `json:"license_number"`
	Phone         string `json:"phone"`
	IsActive      bool   `json:"is_active"`
}

// TableName le dice a GORM que busque en la tabla "Specialists" (Plural y comillas)
func (Specialist) TableName() string {
	return "\"Specialists\""
}
