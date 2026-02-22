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

// GetByID busca un servicio por su ID
func (r *gormServiceRepo) GetByID(id uint) (*domain.Service, error) {
	var service domain.Service
	err := r.db.Table("\"Services\"").First(&service, id).Error
	if err != nil {
		return nil, err
	}
	return &service, nil
}
