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

// SaveWithConsent crea la cita y el consentimiento Ley 1581 en una sola transacción.
// Si cualquiera de los dos falla, ninguno se persiste.
func (r *gormAppointmentRepo) SaveWithConsent(app *domain.Appointment, consent *domain.DataConsent) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(`"Appointments"`).Omit("Patient", "Specialist").Create(app).Error; err != nil {
			return err
		}
		consent.AppointmentID = app.ID
		consent.RegistradoAt = time.Now().UTC()
		return tx.Table(`"DataConsents"`).Create(consent).Error
	})
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

// GetAll trae todas las citas CON los datos del paciente y consentimiento (auditoría)
func (r *gormAppointmentRepo) GetAll() ([]domain.Appointment, error) {
	var apps []domain.Appointment
	err := r.db.Table("\"Appointments\"").
		Preload("Patient").
		Preload("DataConsent").
		Find(&apps).Error
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
		Preload("DataConsent").
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

// AutoCancelExpired cancela citas 'pending' cuyo límite de confirmación ya venció.
// confirmHours: horas antes de la cita que se exige confirmación (ej: 1 → 1h antes).
// Lógica: si start_time - confirmHours <= ahora → el plazo de confirmación venció.
func (r *gormAppointmentRepo) AutoCancelExpired(confirmHours int) (int64, error) {
	// deadline = start_time - confirmHours
	// Si deadline <= now → cancelar
	// Equivalente SQL: start_time <= now + confirmHours (traemos las que ya superaron el límite)
	deadline := time.Now().Add(time.Duration(confirmHours) * time.Hour)

	result := r.db.Table("\"Appointments\"").
		Where("status = ? AND start_time <= ?", "pending", deadline).
		Updates(map[string]interface{}{
			"status":              "cancelled",
			"cancellation_reason": "auto_expired",
			"cancellation_notes":  "Cancelada automáticamente: no se confirmó antes del límite establecido.",
		})
	return result.RowsAffected, result.Error
}
