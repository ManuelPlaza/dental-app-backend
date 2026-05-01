package ports

import (
	"dental-app/internal/core/domain"
	"time"
)

// PatientService define la lógica de negocio disponible para el mundo exterior.
type PatientService interface {
	Create(patient *domain.Patient) error
	List() ([]domain.Patient, error)
	FindByDocument(documentNumber string) (*domain.Patient, error)
	Update(id uint, patient *domain.Patient) error // <--- NUEVO
}

type AppointmentService interface {
	Schedule(appointment *domain.Appointment) error
	ScheduleFromWeb(req *domain.PublicAppointmentRequest) (*domain.Appointment, error)
	Modify(id uint, newStart, newEnd time.Time) error
	Cancel(id uint) error
	List() ([]domain.Appointment, error)
	AdminUpdateStatus(id uint, status string) error
	ListPaginated(page, limit int, status string) ([]domain.Appointment, int64, error)
	GetSummary() (map[string]int64, error)
	AdminUpdate(id uint, req domain.AdminUpdateRequest) error
	AutoCancelExpired() (int64, error) // lee APPOINTMENT_CONFIRM_HOURS del env
}

// ... (interfaces anteriores)

type PaymentService interface {
	Process(payment *domain.Payment) error
	List() ([]domain.Payment, error) // <--- NUEVO
	GetBalance(appID uint) (float64, float64, float64, error)
	UpdatePayment(id uint, payment *domain.Payment) error // <--- NUEVO
	GetByAppointment(appID uint) ([]domain.Payment, error)
}
type MedicalHistoryService interface {
	Create(history *domain.MedicalHistory) error
	GetHistoryByPatient(patientID uint) ([]domain.MedicalHistory, error)
	GetAll() ([]domain.MedicalHistory, error)
}

type SpecialistService interface {
	Create(s *domain.Specialist) error
	List() ([]domain.Specialist, error)
	Inactivate(id uint) error
	Activate(id uint) error
	GetWithoutUser() ([]domain.Specialist, error)
	SetDefault(id uint) error
}

type DashboardService interface {
	GetStats() (map[string]interface{}, error)
}

type AuthService interface {
	Login(email, password string) (*domain.AuthResponse, error)
	RefreshToken(refreshToken string) (*domain.AuthResponse, error)
	ValidateToken(token string) (uint, error)
}
type ServiceService interface {
	List() ([]domain.Service, error)
	ListAll() ([]domain.Service, error)            // <--- NUEVO (admin ve todos)
	Create(service *domain.Service) error          // <--- NUEVO
	Update(id uint, service *domain.Service) error // <--- NUEVO
	Activate(id uint) error                        // <--- NUEVO
	Inactivate(id uint) error                      // <--- NUEVO
}

type ServiceCategoryService interface {
	List() ([]domain.ServiceCategory, error)
}
type NotificationService interface {
	ScheduleConfirmation(appointment *domain.Appointment) error
	ScheduleReminders(appointment *domain.Appointment) error
	ProcessPending() error
	ConfirmAppointment(token string) error
}

type BannerService interface {
	Create(banner *domain.Banner) error
	Update(id uint, banner *domain.Banner) error
	Delete(id uint) error
	GetByID(id uint) (*domain.Banner, error)
	GetAll() ([]domain.Banner, error)
	GetActive() ([]domain.Banner, error)
}

type PaymentLinkService interface {
	// Generate valida saldo, crea el link y dispara la notificación Nequi.
	// amount puede ser parcial (> 0 y <= saldo pendiente).
	Generate(appointmentID uint, amount float64, phone string, createdBy uint) (*domain.PaymentLink, error)
	// GetByToken consulta el estado de un link (endpoint público).
	GetByToken(token string) (*domain.PaymentLink, error)
	// GetByAppointment lista todos los links de una cita (admin).
	GetByAppointment(appointmentID uint) ([]domain.PaymentLink, error)
	// Cancel anula un link pendiente desde el admin.
	Cancel(token string) error
	// ProcessWebhook procesa el callback de Nequi y registra el pago automáticamente.
	ProcessWebhook(payload *NequiWebhookPayload, rawBody string) error
	// ExpireStale lo llama el worker periódicamente.
	ExpireStale() error
}

type ChatService interface {
	Chat(messages []domain.ChatMessage) (string, error)
	InvalidateCache()
}

// NequiWebhookPayload estructura del callback de Nequi.
type NequiWebhookPayload struct {
	TransactionID string  `json:"transactionId"`
	Reference     string  `json:"reference"` // token UUID del PaymentLink
	Status        string  `json:"status"`    // "SUCCESS" | "REJECTED" | "EXPIRED"
	Amount        float64 `json:"amount"`
	PhoneNumber   string  `json:"phoneNumber"`
	Timestamp     string  `json:"timestamp"`
}
