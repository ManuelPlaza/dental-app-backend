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

// GetToday trae todas las citas cuya fecha de inicio es hoy
func (r *gormAppointmentRepo) GetToday() ([]domain.Appointment, error) {
	var apps []domain.Appointment
	today := time.Now().Format("2006-01-02")
	err := r.db.Table("\"Appointments\"").
		Preload("Patient").
		Preload("Specialist").
		Where("DATE(start_time) = ?", today).
		Where("status NOT IN ?", []string{"cancelled"}).
		Order("start_time ASC").
		Find(&apps).Error
	return apps, err
}

// GetMonthlyCancellations trae cancelaciones agrupadas por mes (últimos 6 meses)
func (r *gormAppointmentRepo) GetMonthlyCancellations() ([]map[string]interface{}, error) {
	var results []struct {
		Month  string
		Count  int64
		Reason string
	}

	err := r.db.Table("\"Appointments\"").
		Select("TO_CHAR(DATE_TRUNC('month', start_time), 'YYYY-MM') as month, cancellation_reason as reason, COUNT(*) as count").
		Where("status = ?", "cancelled").
		Where("start_time >= NOW() - INTERVAL '6 months'").
		Group("month, cancellation_reason").
		Order("month ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(results))
	for i, r := range results {
		data[i] = map[string]interface{}{
			"month":  r.Month,
			"count":  r.Count,
			"reason": r.Reason,
		}
	}
	return data, nil
}

// GetTopPatients trae los pacientes con más citas
func (r *gormAppointmentRepo) GetTopPatients(limit int) ([]map[string]interface{}, error) {
	var results []struct {
		PatientID  uint
		FirstName  string
		LastName   string
		Document   string
		TotalCitas int64
	}

	err := r.db.Table("\"Appointments\"").
		Select(`"Patient".id as patient_id, "Patient".first_name, "Patient".last_name, "Patient".document_number as document, COUNT("Appointments".id) as total_citas`).
		Joins(`JOIN "Patient" ON "Patient".id = "Appointments".patient_id`).
		Where(`"Appointments".status NOT IN ?`, []string{"cancelled"}).
		Group(`"Patient".id, "Patient".first_name, "Patient".last_name, "Patient".document_number`).
		Order("total_citas DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, len(results))
	for i, r := range results {
		data[i] = map[string]interface{}{
			"patient_id":  r.PatientID,
			"first_name":  r.FirstName,
			"last_name":   r.LastName,
			"document":    r.Document,
			"total_citas": r.TotalCitas,
		}
	}
	return data, nil
}
