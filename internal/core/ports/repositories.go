package ports

import "dental-app/internal/core/domain"

// PatientRepository define qué se puede hacer con la base de datos
type PatientRepository interface {
	Save(patient *domain.Patient) error
	GetAll() ([]domain.Patient, error)
}

// ... (al final del archivo)
type AppointmentRepository interface {
	Save(appointment *domain.Appointment) error
	GetByID(id uint) (*domain.Appointment, error) // <--- NUEVO
	Update(appointment *domain.Appointment) error // <--- NUEVO
	GetAll() ([]domain.Appointment, error)        // <--- ¡ESTA FALTABA!
}

// ... (interfaces anteriores)

type PaymentRepository interface {
	Save(payment *domain.Payment) error
	GetAll() ([]domain.Payment, error) // <--- NUEVO
	GetByAppointmentID(appID uint) ([]domain.Payment, error)
}
type MedicalHistoryRepository interface {
	Save(history *domain.MedicalHistory) error
	GetByPatientID(patientID uint) ([]domain.MedicalHistory, error)
}