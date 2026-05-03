package repository

import (
	"strings"
	"time"

	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"gorm.io/gorm"
)

type gormPatientRepo struct {
	db *gorm.DB
}

func NewGormPatientRepo(db *gorm.DB) ports.PatientRepository {
	return &gormPatientRepo{db: db}
}

// Save maneja duplicados de document_number
func (r *gormPatientRepo) Save(patient *domain.Patient) error {
	err := r.db.Table("\"Patient\"").Create(patient).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return domain.ErrPatientAlreadyExists
		}
		return err
	}
	return nil
}

// GetAll obtiene todos los pacientes
func (r *gormPatientRepo) GetAll() ([]domain.Patient, error) {
	var patients []domain.Patient
	err := r.db.Table("\"Patient\"").Find(&patients).Error
	return patients, err
}

// FindByDocumentNumber busca un paciente por número de documento
func (r *gormPatientRepo) FindByDocumentNumber(doc string) (*domain.Patient, error) {
	var patient domain.Patient
	err := r.db.Table("\"Patient\"").Where("document_number = ?", doc).First(&patient).Error
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

// Update actualiza solo los campos permitidos del paciente
func (r *gormPatientRepo) Update(id uint, patient *domain.Patient) error {
	return r.db.Table("\"Patient\"").Where("id = ?", id).Updates(map[string]interface{}{
		"phone":                          patient.Phone,
		"email":                          patient.Email,
		"emergency_contact_name":         patient.EmergencyContactName,
		"emergency_contact_relationship": patient.EmergencyContactRelationship,
		"emergency_contact_phone":        patient.EmergencyContactPhone,
	}).Error
}
func (r *gormPatientRepo) FindByID(id uint) (*domain.Patient, error) {
	var patient domain.Patient
	err := r.db.Table("\"Patient\"").First(&patient, id).Error
	if err != nil {
		return nil, err
	}
	return &patient, nil
}

// HasActiveAppointments retorna true si el paciente tiene citas pending o scheduled.
func (r *gormPatientRepo) HasActiveAppointments(id uint) (bool, error) {
	var count int64
	err := r.db.Table(`"Appointments"`).
		Where(`patient_id = ? AND status IN ?`, id, []string{"pending", "scheduled"}).
		Count(&count).Error
	return count > 0, err
}

// GetLastContactDate retorna la fecha del último end_time de citas no canceladas del paciente.
// Retorna nil si no tiene citas activas registradas.
func (r *gormPatientRepo) GetLastContactDate(id uint) (*time.Time, error) {
	var result struct{ MaxEnd *time.Time }
	err := r.db.Table(`"Appointments"`).
		Select(`MAX(end_time) as max_end`).
		Where(`patient_id = ? AND status != 'cancelled'`, id).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result.MaxEnd, nil
}

// GetExpiringPatients retorna pacientes no anonimizados cuyo último contacto es anterior al cutoff.
// Usa created_at como fallback cuando el paciente no tiene citas no canceladas.
func (r *gormPatientRepo) GetExpiringPatients(cutoff time.Time) ([]domain.Patient, error) {
	var patients []domain.Patient
	err := r.db.Table(`"Patient" p`).
		Select(`p.*`).
		Joins(`LEFT JOIN (
			SELECT patient_id, MAX(end_time) AS last_appt
			FROM "Appointments"
			WHERE status != 'cancelled'
			GROUP BY patient_id
		) a ON a.patient_id = p.id`).
		Where(`p.anonymized_at IS NULL`).
		Where(`COALESCE(a.last_appt, p.created_at) <= ?`, cutoff).
		Find(&patients).Error
	return patients, err
}

// AnonymizePatient reemplaza los campos PII del paciente con valores neutros.
func (r *gormPatientRepo) AnonymizePatient(id uint, anonDoc string) error {
	return r.db.Table(`"Patient"`).Where("id = ?", id).Updates(map[string]interface{}{
		"first_name":      "[eliminado]",
		"last_name":       "[eliminado]",
		"phone":           "[eliminado]",
		"email":           "[eliminado]",
		"document_number": anonDoc,
		"anonymized_at":   time.Now().UTC(),
	}).Error
}
