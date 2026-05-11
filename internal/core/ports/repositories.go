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
	Update(id uint, patient *domain.Patient) error
	FindByID(id uint) (*domain.Patient, error)
	// GetLastContactDate retorna la fecha del último contacto (max end_time de citas no canceladas)
	// o nil si no tiene citas.
	GetLastContactDate(id uint) (*time.Time, error)
	// HasActiveAppointments retorna true si el paciente tiene citas pending o scheduled.
	HasActiveAppointments(id uint) (bool, error)
	// GetExpiringPatients retorna pacientes no anonimizados cuyo último contacto es anterior al cutoff.
	GetExpiringPatients(cutoff time.Time) ([]domain.Patient, error)
	// AnonymizePatient reemplaza PII por valores neutrales y registra la fecha de anonimización.
	AnonymizePatient(id uint, anonDoc string) error
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
	// GetOccupiedRanges retorna los rangos [start, end] de citas pending/scheduled
	// del especialista en la fecha indicada (hora Bogotá).
	GetOccupiedRanges(specialistID uint, date time.Time) ([][2]time.Time, error)
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
	// RevokeByDocument elimina todos los registros de consentimiento del titular.
	RevokeByDocument(doc string) error
}

type ChatConfigRepository interface {
	// Get retorna la configuración del chatbot. Retorna nil, nil si no existe aún.
	Get() (*domain.ChatConfig, error)
	// Save crea o actualiza (upsert por id=1).
	Save(config *domain.ChatConfig) error
}

// ReferralGraphRepository gestiona el grafo de referidos en ArangoDB.
// Todas las operaciones son no-fatales: si ArangoDB no está disponible (db == nil), retornan nil.
type ReferralGraphRepository interface {
	// SyncPatient upserta el vértice del paciente en la colección `patients`.
	SyncPatient(id uint, doc, name string) error
	// LinkReferral crea la arista referidor → referido en la colección `referred`.
	LinkReferral(referrerDoc, referredDoc string) error
	// TopReferrers retorna los N pacientes con más referidos, ordenados desc.
	TopReferrers(limit int) ([]domain.TopReferrer, error)
	// GetPatientReferralStatus retorna el estado del paciente en el grafo:
	// cuántos ha referido y quién lo refirió a él (si aplica).
	GetPatientReferralStatus(doc string) (*domain.PatientReferralStatus, error)
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
