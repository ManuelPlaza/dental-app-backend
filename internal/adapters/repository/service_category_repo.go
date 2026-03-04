package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormServiceCategoryRepo struct {
	db *gorm.DB
}

func NewGormServiceCategoryRepo(db *gorm.DB) ports.ServiceCategoryRepository {
	return &gormServiceCategoryRepo{db: db}
}

func (r *gormServiceCategoryRepo) GetAll() ([]domain.ServiceCategory, error) {
	var categories []domain.ServiceCategory
	err := r.db.Table("\"ServiceCategories\"").Find(&categories).Error
	return categories, err
}
