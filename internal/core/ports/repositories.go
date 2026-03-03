package ports

import (
	"dental-app/internal/core/domain"
	"time"
)

// PatientRepository define qué se puede hacer con la base de datos
type PatientRepository interface {
	Save(patient *domain.Patient) error
	GetAll() ([]domain.Patient, error)
	FindByDocumentNumber(doc string) (*domain.Patient, error)
	Update(id uint, patient *domain.Patient) error // <--- NUEVO
}

// ... (al final del archivo)
type AppointmentRepository interface {
	Save(appointment *domain.Appointment) error
	GetByID(id uint) (*domain.Appointment, error)
	Update(appointment *domain.Appointment) error
	GetAll() ([]domain.Appointment, error)
	GetPaginated(page, limit int, status string) ([]domain.Appointment, int64, error) // <--- status agregado
	GetSummary() (map[string]int64, error)
	HasSpecialistConflict(specialistID uint, start, end time.Time, excludeID uint) (bool, error)
	GetToday() ([]domain.Appointment, error)                    // <--- NUEVO
	GetMonthlyCancellations() ([]map[string]interface{}, error) // <--- NUEVO
	GetTopPatients(limit int) ([]map[string]interface{}, error) // <--- NUEVO
}

// ... (interfaces anteriores)

type PaymentRepository interface {
	Save(payment *domain.Payment) error
	GetAll() ([]domain.Payment, error) // <--- NUEVO
	GetByAppointmentID(appID uint) ([]domain.Payment, error)
	GetByID(id uint) (*domain.Payment, error)            // <--- NUEVO
	Update(payment *domain.Payment) error                // <--- NUEVO
	GetMonthlyIncome() ([]map[string]interface{}, error) // <--- NUEVO
	GetRecent(limit int) ([]domain.Payment, error)
}

type MedicalHistoryRepository interface {
	Save(history *domain.MedicalHistory) error
	GetByPatientID(patientID uint) ([]domain.MedicalHistory, error)
	GetAll() ([]domain.MedicalHistory, error) // <--- NUEVO
}

type SpecialistRepository interface {
	Save(s *domain.Specialist) error
	GetAll() ([]domain.Specialist, error)
	Inactivate(id uint) error
	Activate(id uint) error // <--- NUEVO
	ExistsByID(id uint) (bool, error)
	GetWithoutUser() ([]domain.Specialist, error) // <--- NUEVO
}

type ServiceRepository interface {
	GetAll() ([]domain.Service, error)
	GetAllIncludingInactive() ([]domain.Service, error) // <--- NUEVO (para el admin)
	GetByID(id uint) (*domain.Service, error)
	Save(service *domain.Service) error        // <--- NUEVO
	Update(service *domain.Service) error      // <--- NUEVO
	ToggleActive(id uint, isActive bool) error // <--- NUEVO
}
type UserRepository interface {
	FindByEmail(email string) (*domain.User, error)
	FindByID(id uint) (*domain.User, error)
}
