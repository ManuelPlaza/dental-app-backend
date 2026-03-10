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

// GetAll retorna solo los servicios activos y visibles en web (para landing page)
func (r *gormServiceRepo) GetAll() ([]domain.Service, error) {
	var services []domain.Service
	err := r.db.Table("\"Services\"").Where("is_active = ? AND show_on_web = ?", true, true).Find(&services).Error
	return services, err
}

// GetAllIncludingInactive retorna todos los servicios (para admin)
func (r *gormServiceRepo) GetAllIncludingInactive() ([]domain.Service, error) {
	var services []domain.Service
	err := r.db.Table("\"Services\"").Find(&services).Error
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

// Save crea un nuevo servicio
func (r *gormServiceRepo) Save(service *domain.Service) error {
	return r.db.Table("\"Services\"").Create(service).Error
}

// Update actualiza un servicio existente
func (r *gormServiceRepo) Update(service *domain.Service) error {
	return r.db.Table("\"Services\"").Save(service).Error
}

// ToggleActive activa o inactiva un servicio
func (r *gormServiceRepo) ToggleActive(id uint, isActive bool) error {
	return r.db.Table("\"Services\"").Where("id = ?", id).Update("is_active", isActive).Error
}
