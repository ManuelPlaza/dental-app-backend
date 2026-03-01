package services

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
)

type dashboardService struct {
	appointRepo ports.AppointmentRepository
	payRepo     ports.PaymentRepository
	patientRepo ports.PatientRepository
}

func NewDashboardService(
	appointRepo ports.AppointmentRepository,
	payRepo ports.PaymentRepository,
	patientRepo ports.PatientRepository,
) ports.DashboardService {
	return &dashboardService{
		appointRepo: appointRepo,
		payRepo:     payRepo,
		patientRepo: patientRepo,
	}
}

func (s *dashboardService) GetStats() (map[string]interface{}, error) {
	// 1. Resumen de citas
	summary, err := s.appointRepo.GetSummary()
	if err != nil {
		return nil, err
	}

	// 2. Citas de hoy
	todayApps, err := s.appointRepo.GetToday()
	if err != nil {
		return nil, err
	}

	// 3. Cancelaciones mensuales
	cancellations, err := s.appointRepo.GetMonthlyCancellations()
	if err != nil {
		return nil, err
	}

	// 4. Top 5 pacientes
	topPatients, err := s.appointRepo.GetTopPatients(5)
	if err != nil {
		return nil, err
	}

	// 5. Ingresos mensuales
	monthlyIncome, err := s.payRepo.GetMonthlyIncome()
	if err != nil {
		return nil, err
	}

	// 6. Últimos 5 pagos
	recentPayments, err := s.payRepo.GetRecent(5)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"summary":         summary,
		"today":           todayApps,
		"cancellations":   cancellations,
		"top_patients":    topPatients,
		"monthly_income":  monthlyIncome,
		"recent_payments": recentPayments,
	}, nil
}

// Necesario para compilar
var _ domain.Appointment