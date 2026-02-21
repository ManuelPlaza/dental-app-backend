package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
)

type serviceService struct {
	repo ports.ServiceRepository
}

func NewServiceService(repo ports.ServiceRepository) ports.ServiceService {
	return &serviceService{repo: repo}
}

func (s *serviceService) List() ([]domain.Service, error) {
	return s.repo.GetAll()
}
