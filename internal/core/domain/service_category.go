package domain

type ServiceCategory struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (ServiceCategory) TableName() string {
	return "\"ServiceCategories\""
}
