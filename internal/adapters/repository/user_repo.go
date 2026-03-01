package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormUserRepo struct {
	db *gorm.DB
}

func NewGormUserRepo(db *gorm.DB) ports.UserRepository {
	return &gormUserRepo{db: db}
}

func (r *gormUserRepo) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Table("\"Users\"").Where("email = ? AND state = ?", email, "active").First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepo) FindByID(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.Table("\"Users\"").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
