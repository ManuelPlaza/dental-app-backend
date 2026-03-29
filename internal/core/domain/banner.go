package domain

import "time"

type Banner struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	ImageURLDesktop string `json:"image_url_desktop"`
	ImageURLMobile  string `json:"image_url_mobile"`
	RedirectURL string     `json:"redirect_url,omitempty"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	IsActive    bool       `json:"is_active"`
	Priority    int        `json:"priority"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (Banner) TableName() string {
	return "\"Banners\""
}
