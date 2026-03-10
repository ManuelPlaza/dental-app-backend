package services

import (
	"crypto/rand"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	resend "github.com/resend/resend-go/v3"
)

type notificationService struct {
	repo        ports.NotificationRepository
	appointRepo ports.AppointmentRepository
	patientRepo ports.PatientRepository
}

func NewNotificationService(
	repo ports.NotificationRepository,
	appointRepo ports.AppointmentRepository,
	patientRepo ports.PatientRepository,
) ports.NotificationService {
	return &notificationService{
		repo:        repo,
		appointRepo: appointRepo,
		patientRepo: patientRepo, // ← estaba faltando esto
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *notificationService) ScheduleConfirmation(app *domain.Appointment) error {
	if app.Status != "pending" && app.Status != "scheduled" {
		return nil
	}

	existing, _ := s.repo.FindByAppointmentAndType(app.ID, domain.NotificationConfirmation)
	if existing != nil {
		return nil
	}

	token, err := generateToken()
	if err != nil {
		return errors.New("error generando token")
	}

	n := &domain.NotificationQueue{
		AppointmentID: app.ID,
		PatientID:     app.PatientID,
		Type:          domain.NotificationConfirmation,
		Status:        domain.NotificationPending,
		ScheduledAt:   time.Now(),
		MaxAttempts:   3,
		Token:         token,
	}

	if err := s.repo.Save(n); err != nil {
		return err
	}

	s.saveLog(n.ID, "scheduled", "Confirmación programada")
	return nil
}

func (s *notificationService) ScheduleReminders(app *domain.Appointment) error {
	if app.Status != "pending" && app.Status != "scheduled" {
		return nil
	}

	startTime := app.StartTime.Time

	existing24h, _ := s.repo.FindByAppointmentAndType(app.ID, domain.NotificationReminder24H)
	if existing24h == nil {
		scheduled24h := startTime.Add(-24 * time.Hour)
		if scheduled24h.After(time.Now()) {
			r24 := &domain.NotificationQueue{
				AppointmentID: app.ID,
				PatientID:     app.PatientID,
				Type:          domain.NotificationReminder24H,
				Status:        domain.NotificationPending,
				ScheduledAt:   scheduled24h,
				MaxAttempts:   3,
			}
			if err := s.repo.Save(r24); err != nil {
				return err
			}
			s.saveLog(r24.ID, "scheduled", "Recordatorio 24h programado")
		}
	}

	existingFinal, _ := s.repo.FindByAppointmentAndType(app.ID, domain.NotificationReminderFinal)
	if existingFinal == nil {
		scheduledFinal := startTime.Add(-2 * time.Hour)
		if scheduledFinal.After(time.Now()) {
			rFinal := &domain.NotificationQueue{
				AppointmentID: app.ID,
				PatientID:     app.PatientID,
				Type:          domain.NotificationReminderFinal,
				Status:        domain.NotificationPending,
				ScheduledAt:   scheduledFinal,
				MaxAttempts:   3,
			}
			if err := s.repo.Save(rFinal); err != nil {
				return err
			}
			s.saveLog(rFinal.ID, "scheduled", "Recordatorio final 2h programado")
		}
	}

	return nil
}

func (s *notificationService) ProcessPending() error {
	notifications, err := s.repo.GetPending()
	if err != nil {
		return err
	}

	for _, n := range notifications {
		app, err := s.appointRepo.GetByID(n.AppointmentID)
		if err != nil {
			log.Printf("⚠️ Cita %d no encontrada para notificación %d", n.AppointmentID, n.ID)
			n.Status = domain.NotificationSkipped
			s.repo.Update(&n)
			continue
		}

		if app.Status == "completed" || app.Status == "cancelled" {
			n.Status = domain.NotificationSkipped
			s.repo.Update(&n)
			s.saveLog(n.ID, "skipped", fmt.Sprintf("Cita en estado %s", app.Status))
			log.Printf("⏭️ Notificación %d omitida — cita %s", n.ID, app.Status)
			continue
		}

		n.Appointment = *app

		if err := s.sendNotification(&n); err != nil {
			n.Attempts++
			n.ErrorLog = err.Error()
			if n.Attempts >= n.MaxAttempts {
				n.Status = domain.NotificationFailed
				s.saveLog(n.ID, "failed", err.Error())
			}
			log.Printf("❌ Notificación %d error (intento %d): %s", n.ID, n.Attempts, err.Error())
		} else {
			now := time.Now()
			n.Status = domain.NotificationSent
			n.SentAt = &now
			n.Attempts++
			s.saveLog(n.ID, "sent", "Enviado correctamente")
			log.Printf("✅ Notificación %d enviada correctamente", n.ID)
		}

		s.repo.Update(&n)
	}
	return nil
}

// sendNotification recibe el paciente ya cargado desde ProcessPending
func (s *notificationService) sendNotification(n *domain.NotificationQueue) error {
	app := n.Appointment

	// Cargar paciente directamente por ID — no confiar en el Preload
	patient, err := s.patientRepo.FindByID(n.PatientID)
	if err != nil {
		return fmt.Errorf("paciente %d no encontrado: %w", n.PatientID, err)
	}

	log.Printf("🔍 DEBUG — PatientID: %d | Email: '%s' | Nombre: '%s'",
		n.PatientID, patient.Email, patient.FirstName)

	if patient.Email == "" {
		n.Status = domain.NotificationSkipped
		s.saveLog(n.ID, "skipped", "Paciente sin email")
		return nil
	}

	patientName := fmt.Sprintf("%s %s", patient.FirstName, patient.LastName)

	specialistName := "tu especialista"
	if app.Specialist.FirstName != "" {
		specialistName = fmt.Sprintf("%s %s", app.Specialist.FirstName, app.Specialist.LastName)
	}

	dateStr := app.StartTime.Time.In(bogotaLocEmail).Format("Monday 02 de January de 2006 a las 03:04 PM")

	var subject string
	var htmlBody string

	switch n.Type {
	case domain.NotificationConfirmation:
		baseURL := os.Getenv("FRONTEND_URL")
		if baseURL == "" {
			baseURL = "http://localhost:3001"
		}
		confirmURL := fmt.Sprintf("%s/confirmar-cita?token=%s", baseURL, n.Token)
		subject = "Confirma tu cita - Dental JC"
		htmlBody, err = renderConfirmation(confirmationData{
			PatientName:    patientName,
			SpecialistName: specialistName,
			Date:           dateStr,
			ConfirmURL:     confirmURL,
		})

	case domain.NotificationReminder24H:
		subject = "Recordatorio: Tu cita es mañana - Dental JC"
		htmlBody, err = renderReminder24h(reminderData{
			PatientName:    patientName,
			SpecialistName: specialistName,
			Date:           dateStr,
		})

	case domain.NotificationReminderFinal:
		subject = "¡Tu cita es hoy! - Dental JC"
		htmlBody, err = renderReminderFinal(reminderData{
			PatientName:    patientName,
			SpecialistName: specialistName,
			Date:           dateStr,
		})

	default:
		return errors.New("tipo de notificación desconocido")
	}

	if err != nil {
		return fmt.Errorf("error en template: %w", err)
	}

	return sendEmailResend(patient.Email, subject, htmlBody)
}

func sendEmailResend(toEmail, subject, htmlBody string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return errors.New("RESEND_API_KEY no configurada")
	}

	from := os.Getenv("RESEND_FROM_EMAIL")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	log.Printf("📧 Intentando enviar a: %s | from: %s", toEmail, from)

	client := resend.NewClient(apiKey)
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{toEmail},
		Subject: subject,
		Html:    htmlBody,
	}

	resp, err := client.Emails.Send(params)
	if err != nil {
		log.Printf("❌ Resend error: %v", err)
		return fmt.Errorf("error Resend: %w", err)
	}

	log.Printf("📧 Resend Email ID: %s → enviado a: %s", resp.Id, toEmail)
	return nil
}

func (s *notificationService) ConfirmAppointment(token string) error {
	n, err := s.repo.FindByToken(token)
	if err != nil {
		return errors.New("token inválido o expirado")
	}

	app, err := s.appointRepo.GetByID(n.AppointmentID)
	if err != nil {
		return errors.New("cita no encontrada")
	}

	if app.Status == "cancelled" {
		return errors.New("esta cita fue cancelada")
	}
	if app.Status == "completed" {
		return errors.New("esta cita ya fue completada")
	}
	if app.Status == "scheduled" {
		return nil
	}

	app.Status = "scheduled"
	if err := s.appointRepo.Update(app); err != nil {
		return errors.New("error confirmando la cita")
	}

	n.Status = domain.NotificationSkipped
	s.repo.Update(n)
	s.saveLog(n.ID, "confirmed", "Paciente confirmó asistencia")
	return nil
}

func (s *notificationService) saveLog(notificationID uint, event, detail string) {
	l := &domain.NotificationLog{
		NotificationID: notificationID,
		Event:          event,
		Detail:         detail,
	}
	if err := s.repo.SaveLog(l); err != nil {
		log.Printf("⚠️ Error guardando log %d: %s", notificationID, err.Error())
	}
}
