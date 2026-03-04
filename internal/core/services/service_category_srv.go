package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
)

type serviceCategoryService struct {
	repo ports.ServiceCategoryRepository
}

func NewServiceCategoryService(repo ports.ServiceCategoryRepository) ports.ServiceCategoryService {
	return &serviceCategoryService{repo: repo}
}

func (s *serviceCategoryService) List() ([]domain.ServiceCategory, error) {
	return s.repo.GetAll()
}
