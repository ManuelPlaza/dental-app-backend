package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"time"

	"gorm.io/gorm"
)

type gormAppointmentRepo struct {
	db *gorm.DB
}

func NewGormAppointmentRepo(db *gorm.DB) ports.AppointmentRepository {
	return &gormAppointmentRepo{db: db}
}

func (r *gormAppointmentRepo) Save(app *domain.Appointment) error {
	return r.db.Table("\"Appointments\"").Omit("Patient", "Specialist").Create(app).Error
}

// GetByID busca una cita por su llave primaria
func (r *gormAppointmentRepo) GetByID(id uint) (*domain.Appointment, error) {
	var app domain.Appointment
	err := r.db.Table("\"Appointments\"").First(&app, id).Error
	return &app, err
}

// Update guarda cualquier cambio (estado, fecha, contador)
func (r *gormAppointmentRepo) Update(app *domain.Appointment) error {
	return r.db.Table("\"Appointments\"").Save(app).Error
}

// GetAll trae todas las citas CON los datos del paciente
func (r *gormAppointmentRepo) GetAll() ([]domain.Appointment, error) {
	var apps []domain.Appointment
	err := r.db.Table("\"Appointments\"").Preload("Patient").Find(&apps).Error
	return apps, err
}

// GetPaginated ahora soporta filtro opcional por status
func (r *gormAppointmentRepo) GetPaginated(page, limit int, status string) ([]domain.Appointment, int64, error) {
	var apps []domain.Appointment
	var total int64

	offset := (page - 1) * limit
	query := r.db.Table("\"Appointments\"")

	// Aplicar filtro por status si viene
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Contar total con el filtro aplicado
	query.Count(&total)

	// Traer solo los de esta página
	err := query.
		Preload("Patient").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&apps).Error

	return apps, total, err
}

// GetSummary retorna conteos por estado para las cards
func (r *gormAppointmentRepo) GetSummary() (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}

	err := r.db.Table("\"Appointments\"").
		Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	summary := map[string]int64{
		"total":     0,
		"pending":   0,
		"scheduled": 0,
		"completed": 0,
		"cancelled": 0,
	}

	for _, r := range results {
		summary[r.Status] = r.Count
		summary["total"] += r.Count
	}

	return summary, nil
}

// HasSpecialistConflict verifica si el especialista ya tiene cita en ese horario
func (r *gormAppointmentRepo) HasSpecialistConflict(specialistID uint, start, end time.Time, excludeID uint) (bool, error) {
	var count int64
	err := r.db.Table("\"Appointments\"").
		Where("specialist_id = ?", specialistID).
		Where("id != ?", excludeID). // Excluir la cita actual al editar
		Where("status IN ?", []string{"pending", "scheduled"}).
		Where("start_time < ? AND end_time > ?", end, start). // Solapamiento
		Count(&count).Error

	return count > 0, err
}
