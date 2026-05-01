package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormChatConfigRepo struct {
	db *gorm.DB
}

func NewGormChatConfigRepo(db *gorm.DB) ports.ChatConfigRepository {
	return &gormChatConfigRepo{db: db}
}

func (r *gormChatConfigRepo) Get() (*domain.ChatConfig, error) {
	var cfg domain.ChatConfig
	err := r.db.First(&cfg, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *gormChatConfigRepo) Save(config *domain.ChatConfig) error {
	config.ID = 1 // fila única
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(config).Error
}
