package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormDataConsentRepo struct {
	db *gorm.DB
}

func NewGormDataConsentRepo(db *gorm.DB) ports.DataConsentRepository {
	return &gormDataConsentRepo{db: db}
}

// FindByDocument retorna el consentimiento más reciente del titular por número de documento.
// Retorna nil, nil si no existe registro previo.
func (r *gormDataConsentRepo) FindByDocument(doc string) (*domain.DataConsent, error) {
	var consent domain.DataConsent
	err := r.db.Table(`"DataConsents"`).
		Where("documento_titular = ?", doc).
		Order("created_at DESC").
		First(&consent).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &consent, nil
}
