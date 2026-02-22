package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type gormSpecialistRepo struct {
	db *gorm.DB
}

func NewGormSpecialistRepo(db *gorm.DB) ports.SpecialistRepository {
	return &gormSpecialistRepo{db: db}
}

func (r *gormSpecialistRepo) Save(s *domain.Specialist) error {
	// Usa el nombre real según tu schema.sql: "Specialists"
	err := r.db.Table(`"Specialists"`).Create(s).Error
	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 = unique_violation (ej: license_number unique)
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrSpecialistAlreadyExists
		}
		return err
	}
	return nil
}

func (r *gormSpecialistRepo) GetAll() ([]domain.Specialist, error) {
	var list []domain.Specialist
	err := r.db.Table(`"Specialists"`).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *gormSpecialistRepo) ExistsByID(id uint) (bool, error) {
	var count int64
	err := r.db.Table(`"Specialists"`).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *gormSpecialistRepo) Inactivate(id uint) error {
	// Inactivación lógica
	res := r.db.Table(`"Specialists"`).
		Where("id = ?", id).
		Update("is_active", false)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrSpecialistNotFound
	}
	return nil
}

// GetWithoutUser retorna especialistas que no tienen usuario asociado
func (r *gormSpecialistRepo) GetWithoutUser() ([]domain.Specialist, error) {
	var specialists []domain.Specialist
	err := r.db.Table("\"Specialists\"").
		Where("user_id IS NULL AND is_active = ?", true).
		Find(&specialists).Error
	return specialists, err
}
func (r *gormSpecialistRepo) Activate(id uint) error {
	return r.db.Table("\"Specialists\"").Where("id = ?", id).Update("is_active", true).Error
}
