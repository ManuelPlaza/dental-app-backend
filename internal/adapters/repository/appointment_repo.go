package repository

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormAppointmentRepo struct {
	db *gorm.DB
}

func NewGormAppointmentRepo(db *gorm.DB) ports.AppointmentRepository {
	return &gormAppointmentRepo{db: db}
}

func (r *gormAppointmentRepo) Save(app *domain.Appointment) error {
	// OJO AQUÍ: Usamos comillas escapadas porque DrawSQL creó la tabla como "Appointments"
	return r.db.Table("\"Appointments\"").Create(app).Error
}

// ... (código existente)

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

// ... (imports y código anterior) ...

// GetAll trae todas las citas CON los datos del paciente
func (r *gormAppointmentRepo) GetAll() ([]domain.Appointment, error) {
	var apps []domain.Appointment
	// .Preload("Patient") llena el struct Patient dentro de Appointment
	err := r.db.Table("\"Appointments\"").Preload("Patient").Find(&apps).Error
	return apps, err
}
