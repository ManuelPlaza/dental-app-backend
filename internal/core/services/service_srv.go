package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
)

type serviceService struct {
	repo ports.ServiceRepository
}

func NewServiceService(repo ports.ServiceRepository) ports.ServiceService {
	return &serviceService{repo: repo}
}

// List retorna solo servicios activos (landing page)
func (s *serviceService) List() ([]domain.Service, error) {
	return s.repo.GetAll()
}

// ListAll retorna todos los servicios incluyendo inactivos (admin)
func (s *serviceService) ListAll() ([]domain.Service, error) {
	return s.repo.GetAllIncludingInactive()
}

// Create crea un nuevo servicio
func (s *serviceService) Create(service *domain.Service) error {
	if service.Name == "" {
		return errors.New("el nombre del servicio es obligatorio")
	}
	if service.Price <= 0 {
		return errors.New("el precio debe ser mayor a cero")
	}
	if service.DurationMinutes <= 0 {
		service.DurationMinutes = 60 // Default 60 minutos
	}
	service.IsActive = true // Activo por defecto
	return s.repo.Save(service)
}

// Update actualiza un servicio existente
func (s *serviceService) Update(id uint, service *domain.Service) error {
	if service.Name == "" {
		return errors.New("el nombre del servicio es obligatorio")
	}
	if service.Price <= 0 {
		return errors.New("el precio debe ser mayor a cero")
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("servicio no encontrado")
	}

	existing.Name = service.Name
	existing.Description = service.Description
	existing.Price = service.Price
	existing.DurationMinutes = service.DurationMinutes
	existing.CategoryID = service.CategoryID

	return s.repo.Update(existing)
}

// Activate activa un servicio
func (s *serviceService) Activate(id uint) error {
	return s.repo.ToggleActive(id, true)
}

// Inactivate inactiva un servicio
func (s *serviceService) Inactivate(id uint) error {
	return s.repo.ToggleActive(id, false)
}
