package domain

type Service struct {
	ID              uint    `json:"id"`
	CategoryID      uint    `json:"category_id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	DurationMinutes int     `json:"duration_minutes"`
	IsActive        bool    `json:"is_active"`
	ShowOnWeb       bool    `json:"show_on_web"`
}

func (Service) TableName() string {
	return "\"Services\""
}
