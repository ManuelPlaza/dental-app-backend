package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"strings"
)

type specialistService struct {
	repo ports.SpecialistRepository
}

func NewSpecialistService(repo ports.SpecialistRepository) ports.SpecialistService {
	return &specialistService{repo: repo}
}

func (s *specialistService) Create(sp *domain.Specialist) error {
	// Validaciones mínimas (puedes endurecerlas luego)
	sp.LicenseNumber = strings.TrimSpace(sp.LicenseNumber)
	if sp.LicenseNumber == "" {
		return domain.ErrLicenseRequired
	}
	sp.FirstName = strings.TrimSpace(sp.FirstName)
	sp.LastName = strings.TrimSpace(sp.LastName)
	sp.Specialty = strings.TrimSpace(sp.Specialty)
	sp.Phone = strings.TrimSpace(sp.Phone)

	// Por defecto activo al crear
	sp.IsActive = true

	return s.repo.Save(sp)
}

func (s *specialistService) List() ([]domain.Specialist, error) {
	return s.repo.GetAll()
}

func (s *specialistService) Inactivate(id uint) error {
	if id == 0 {
		return domain.ErrInvalidPayload
	}

	exists, err := s.repo.ExistsByID(id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrSpecialistNotFound
	}

	return s.repo.Inactivate(id)
}
func (s *specialistService) GetWithoutUser() ([]domain.Specialist, error) {
	return s.repo.GetWithoutUser()
}
func (s *specialistService) Activate(id uint) error {
	return s.repo.Activate(id)
}
