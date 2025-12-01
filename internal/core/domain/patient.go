package domain
import "time"
type Patient struct {
	ID             uint      `json:"id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	DocumentNumber string    `json:"document_number"`
	CreatedAt      time.Time `json:"created_at"`
}
// ... (debajo del struct Patient)

// TableName le dice a GORM el nombre exacto de la tabla en PostgreSQL
func (Patient) TableName() string {
	return "\"Patient\""
}