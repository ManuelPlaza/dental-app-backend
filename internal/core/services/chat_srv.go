package services

import (
	"dental-app/internal/adapters/groq"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type chatService struct {
	serviceRepo  ports.ServiceRepository
	groqClient   *groq.Client
	cacheMu      sync.RWMutex
	cachedPrompt string
	cacheExpiry  time.Time
}

func NewChatService(serviceRepo ports.ServiceRepository, groqClient *groq.Client) ports.ChatService {
	return &chatService{
		serviceRepo: serviceRepo,
		groqClient:  groqClient,
	}
}

func (s *chatService) Chat(messages []domain.ChatMessage) (string, error) {
	systemPrompt, err := s.buildSystemPrompt()
	if err != nil {
		return "", err
	}

	// Limit to last 10 messages to control context size
	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
	}

	groqMessages := make([]groq.Message, len(messages))
	for i, m := range messages {
		groqMessages[i] = groq.Message{Role: m.Role, Content: m.Content}
	}

	return s.groqClient.Chat(systemPrompt, groqMessages)
}

// buildSystemPrompt constructs the system prompt injecting the active service catalog.
// Result is cached 15 minutes to avoid repeated DB reads per request.
func (s *chatService) buildSystemPrompt() (string, error) {
	s.cacheMu.RLock()
	if time.Now().Before(s.cacheExpiry) {
		prompt := s.cachedPrompt
		s.cacheMu.RUnlock()
		return prompt, nil
	}
	s.cacheMu.RUnlock()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(s.cacheExpiry) {
		return s.cachedPrompt, nil
	}

	svcs, err := s.serviceRepo.GetAll()
	if err != nil {
		return "", fmt.Errorf("error cargando catálogo de servicios: %w", err)
	}

	clinicName := os.Getenv("CLINIC_NAME")
	if clinicName == "" {
		clinicName = "Técnica Dental JC"
	}
	businessType := os.Getenv("BUSINESS_TYPE")
	if businessType == "" {
		businessType = "laboratorio dental técnico"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Eres el asistente virtual de %s, un %s.\n", clinicName, businessType))
	sb.WriteString("IMPORTANTE — restricciones legales que NUNCA debes ignorar:\n")
	sb.WriteString("1. NUNCA te describas como clínica, consultorio, centro médico ni institución de salud.\n")
	sb.WriteString("2. NUNCA des diagnósticos, recomendaciones clínicas ni consejos sobre tratamientos odontológicos.\n")
	sb.WriteString("3. Si alguien pregunta sobre síntomas, dolor dental o qué tratamiento necesita, indícale que consulte a su odontólogo tratante.\n")
	sb.WriteString("4. Solo informa sobre los servicios técnicos que ofrece el laboratorio y cómo agendar.\n")
	sb.WriteString("Responde siempre en español, de forma amigable, clara y profesional.\n")
	sb.WriteString("Sé conciso: respuestas de máximo 3-4 oraciones.\n")
	sb.WriteString("Si alguien quiere agendar, indícale que puede hacerlo desde el formulario en la página web.\n")
	sb.WriteString("No inventes información que no esté en el catálogo. Si no sabes algo, sugiere contactar directamente al laboratorio.\n\n")
	sb.WriteString("--- CATÁLOGO DE SERVICIOS DEL LABORATORIO ---\n")

	for _, svc := range svcs {
		line := fmt.Sprintf("• %s", svc.Name)
		if svc.Description != "" {
			line += fmt.Sprintf(": %s", svc.Description)
		}
		if svc.Price > 0 {
			line += fmt.Sprintf(" — Precio: $%.0f COP", svc.Price)
		}
		if svc.DurationMinutes > 0 {
			line += fmt.Sprintf(" (%d min)", svc.DurationMinutes)
		}
		sb.WriteString(line + "\n")
	}

	s.cachedPrompt = sb.String()
	s.cacheExpiry = time.Now().Add(15 * time.Minute)

	return s.cachedPrompt, nil
}
