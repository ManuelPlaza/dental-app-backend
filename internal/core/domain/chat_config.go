package domain

import "time"

// ChatConfig almacena el conocimiento del chatbot editable desde el admin.
// Una sola fila en la tabla (upsert por id=1).
type ChatConfig struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Whatsapp      string    `json:"whatsapp"`       // ej: "+57 310 123 4567"
	Address       string    `json:"address"`        // dirección física
	BusinessHours string    `json:"business_hours"` // ej: "Lunes a Viernes 8am–6pm"
	PolicyURL     string    `json:"policy_url"`     // URL política de tratamiento de datos
	ContactEmail  string    `json:"contact_email"`
	ExtraInfo     string    `json:"extra_info"`     // texto libre: FAQs, instrucciones, etc.
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ChatConfig) TableName() string { return `"ChatConfig"` }
