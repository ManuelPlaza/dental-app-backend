package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
)

type referralService struct {
	repo ports.ReferralGraphRepository
}

func NewReferralService(repo ports.ReferralGraphRepository) ports.ReferralService {
	return &referralService{repo: repo}
}

func (s *referralService) TopReferrers(limit int) ([]domain.TopReferrer, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.TopReferrers(limit)
}
