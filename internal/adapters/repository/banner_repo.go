package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"time"

	"gorm.io/gorm"
)

type gormBannerRepo struct {
	db *gorm.DB
}

func NewGormBannerRepo(db *gorm.DB) ports.BannerRepository {
	return &gormBannerRepo{db: db}
}

func (r *gormBannerRepo) Save(banner *domain.Banner) error {
	now := time.Now()
	banner.CreatedAt = now
	banner.UpdatedAt = now
	return r.db.Table("\"Banners\"").Create(banner).Error
}

func (r *gormBannerRepo) GetByID(id uint) (*domain.Banner, error) {
	var banner domain.Banner
	err := r.db.Table("\"Banners\"").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&banner).Error
	if err != nil {
		return nil, err
	}
	return &banner, nil
}

// GetAll retorna todos los banners no eliminados (vista admin)
func (r *gormBannerRepo) GetAll() ([]domain.Banner, error) {
	var banners []domain.Banner
	err := r.db.Table("\"Banners\"").
		Where("deleted_at IS NULL").
		Order("priority DESC, start_time ASC").
		Find(&banners).Error
	return banners, err
}

// GetActive retorna solo banners activos y vigentes (vista pública)
func (r *gormBannerRepo) GetActive() ([]domain.Banner, error) {
	var banners []domain.Banner
	now := time.Now().UTC()
	err := r.db.Table("\"Banners\"").
		Where("is_active = ? AND start_time <= ? AND end_time >= ? AND deleted_at IS NULL", true, now, now).
		Order("priority DESC, start_time ASC").
		Find(&banners).Error
	return banners, err
}

func (r *gormBannerRepo) Update(banner *domain.Banner) error {
	banner.UpdatedAt = time.Now()
	return r.db.Table("\"Banners\"").Save(banner).Error
}

func (r *gormBannerRepo) SoftDelete(id uint) error {
	now := time.Now()
	return r.db.Table("\"Banners\"").
		Where("id = ?", id).
		Updates(map[string]interface{}{"deleted_at": now, "updated_at": now}).Error
}
