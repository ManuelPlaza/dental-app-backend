package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormServiceRepo struct {
	db *gorm.DB
}

func NewGormServiceRepo(db *gorm.DB) ports.ServiceRepository {
	return &gormServiceRepo{db: db}
}

// GetAll retorna solo los servicios activos
func (r *gormServiceRepo) GetAll() ([]domain.Service, error) {
	var services []domain.Service
	err := r.db.Table("\"Services\"").Where("is_active = ?", true).Find(&services).Error
	return services, err
}
