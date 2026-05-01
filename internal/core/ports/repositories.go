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
	FindByID(id uint) (*domain.Patient, error)
}

// ... (al final del archivo)
type AppointmentRepository interface {
	Save(appointment *domain.Appointment) error
	SaveWithConsent(app *domain.Appointment, consent *domain.DataConsent) error
	GetByID(id uint) (*domain.Appointment, error)
	Update(appointment *domain.Appointment) error
	GetAll() ([]domain.Appointment, error)
	GetPaginated(page, limit int, status string) ([]domain.Appointment, int64, error)
	GetSummary() (map[string]int64, error)
	HasSpecialistConflict(specialistID uint, start, end time.Time, excludeID uint) (bool, error)
	GetToday() ([]domain.Appointment, error)
	GetMonthlyCancellations() ([]map[string]interface{}, error)
	GetTopPatients(limit int) ([]map[string]interface{}, error)
	AutoCancelExpired(confirmHours int) (int64, error) // cancela pending cuyo límite de confirmación venció
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
	Activate(id uint) error
	ExistsByID(id uint) (bool, error)
	GetWithoutUser() ([]domain.Specialist, error)
	GetDefault() (*domain.Specialist, error)
	SetDefault(id uint) error
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
type ServiceCategoryRepository interface {
	GetAll() ([]domain.ServiceCategory, error)
}
type NotificationRepository interface {
	Save(n *domain.NotificationQueue) error
	GetPending() ([]domain.NotificationQueue, error)
	Update(n *domain.NotificationQueue) error
	FindByToken(token string) (*domain.NotificationQueue, error)
	FindByAppointmentAndType(appointmentID uint, nType domain.NotificationType) (*domain.NotificationQueue, error)
	SaveLog(log *domain.NotificationLog) error
}

type BannerRepository interface {
	Save(banner *domain.Banner) error
	GetByID(id uint) (*domain.Banner, error)
	GetAll() ([]domain.Banner, error)
	GetActive() ([]domain.Banner, error)
	Update(banner *domain.Banner) error
	SoftDelete(id uint) error
}

type DataConsentRepository interface {
	// FindByDocument retorna el consentimiento más reciente del titular.
	// Retorna nil, nil si no existe (sin error).
	FindByDocument(doc string) (*domain.DataConsent, error)
}

type ChatConfigRepository interface {
	// Get retorna la configuración del chatbot. Retorna nil, nil si no existe aún.
	Get() (*domain.ChatConfig, error)
	// Save crea o actualiza (upsert por id=1).
	Save(config *domain.ChatConfig) error
}

type PaymentLinkRepository interface {
	Save(link *domain.PaymentLink) error
	GetByToken(token string) (*domain.PaymentLink, error)
	GetByNequiTxnID(txnID string) (*domain.PaymentLink, error)
	GetByAppointmentID(appointmentID uint) ([]domain.PaymentLink, error)
	UpdateStatus(id uint, status domain.PaymentLinkStatus, paidAt *time.Time, webhookPayload string) error
	UpdateNequiTxnID(id uint, txnID string) error
	CancelPendingByAppointment(appointmentID uint) error
	ExpireStale() error // marca como expired los links con expires_at < now y status=pending
}
