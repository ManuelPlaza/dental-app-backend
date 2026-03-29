package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"
)

type bannerService struct {
	repo ports.BannerRepository
}

func NewBannerService(repo ports.BannerRepository) ports.BannerService {
	return &bannerService{repo: repo}
}

func (s *bannerService) Create(banner *domain.Banner) error {
	if banner.Title == "" {
		return errors.New("el título del banner es obligatorio")
	}
	if banner.ImageURLDesktop == "" {
		return errors.New("la URL de la imagen para desktop es obligatoria")
	}
	if banner.StartTime.IsZero() || banner.EndTime.IsZero() {
		return errors.New("las fechas de inicio y fin son obligatorias")
	}
	if !banner.EndTime.After(banner.StartTime) {
		return errors.New("la fecha de fin debe ser posterior a la fecha de inicio")
	}
	banner.IsActive = true
	return s.repo.Save(banner)
}

func (s *bannerService) Update(id uint, banner *domain.Banner) error {
	if banner.Title == "" {
		return errors.New("el título del banner es obligatorio")
	}
	if banner.ImageURLDesktop == "" {
		return errors.New("la URL de la imagen para desktop es obligatoria")
	}
	if banner.StartTime.IsZero() || banner.EndTime.IsZero() {
		return errors.New("las fechas de inicio y fin son obligatorias")
	}
	if !banner.EndTime.After(banner.StartTime) {
		return errors.New("la fecha de fin debe ser posterior a la fecha de inicio")
	}

	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("banner no encontrado")
	}

	existing.Title = banner.Title
	existing.Description = banner.Description
	existing.ImageURLDesktop = banner.ImageURLDesktop
	existing.ImageURLMobile = banner.ImageURLMobile
	existing.RedirectURL = banner.RedirectURL
	existing.StartTime = banner.StartTime
	existing.EndTime = banner.EndTime
	existing.IsActive = banner.IsActive
	existing.Priority = banner.Priority

	return s.repo.Update(existing)
}

func (s *bannerService) Delete(id uint) error {
	_, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("banner no encontrado")
	}
	return s.repo.SoftDelete(id)
}

func (s *bannerService) GetByID(id uint) (*domain.Banner, error) {
	return s.repo.GetByID(id)
}

func (s *bannerService) GetAll() ([]domain.Banner, error) {
	return s.repo.GetAll()
}

// GetActive retorna banners activos y vigentes según la hora actual
func (s *bannerService) GetActive() ([]domain.Banner, error) {
	return s.repo.GetActive()
}
