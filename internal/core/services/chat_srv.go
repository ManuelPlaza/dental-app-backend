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
	serviceRepo    ports.ServiceRepository
	chatConfigRepo ports.ChatConfigRepository
	groqClient     *groq.Client
	cacheMu        sync.RWMutex
	cachedPrompt   string
	cacheExpiry    time.Time
}

func NewChatService(
	serviceRepo ports.ServiceRepository,
	chatConfigRepo ports.ChatConfigRepository,
	groqClient *groq.Client,
) ports.ChatService {
	return &chatService{
		serviceRepo:    serviceRepo,
		chatConfigRepo: chatConfigRepo,
		groqClient:     groqClient,
	}
}

func (s *chatService) Chat(messages []domain.ChatMessage) (string, error) {
	systemPrompt, err := s.buildSystemPrompt()
	if err != nil {
		return "", err
	}

	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
	}

	groqMessages := make([]groq.Message, len(messages))
	for i, m := range messages {
		groqMessages[i] = groq.Message{Role: m.Role, Content: m.Content}
	}

	return s.groqClient.Chat(systemPrompt, groqMessages)
}

// InvalidateCache fuerza la reconstrucción del prompt en la próxima llamada.
// Se llama cuando el admin actualiza la ChatConfig.
func (s *chatService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cacheExpiry = time.Time{}
	s.cacheMu.Unlock()
}

// buildSystemPrompt construye el system prompt con el catálogo de servicios y la
// configuración del negocio cargada desde la BD. Resultado cacheado 15 min.
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

	if time.Now().Before(s.cacheExpiry) {
		return s.cachedPrompt, nil
	}

	svcs, err := s.serviceRepo.GetAll()
	if err != nil {
		return "", fmt.Errorf("error cargando catálogo de servicios: %w", err)
	}

	cfg, err := s.chatConfigRepo.Get()
	if err != nil {
		return "", fmt.Errorf("error cargando configuración del chatbot: %w", err)
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

	// ── Identidad y restricciones legales ────────────────────────────────────
	sb.WriteString(fmt.Sprintf("Eres el asistente virtual de %s, un %s.\n", clinicName, businessType))
	sb.WriteString("RESTRICCIONES LEGALES — NUNCA las ignores:\n")
	sb.WriteString("1. No te describas como clínica, consultorio, centro médico ni institución de salud.\n")
	sb.WriteString("2. No des diagnósticos, recomendaciones clínicas ni consejos sobre tratamientos odontológicos.\n")
	sb.WriteString("3. Si preguntan sobre síntomas o dolor dental, indica que consulten a su odontólogo tratante.\n")
	sb.WriteString("4. Solo informa sobre los servicios técnicos del laboratorio, contacto y agendamiento.\n\n")

	sb.WriteString("Responde siempre en español, de forma amigable, clara y profesional.\n")
	sb.WriteString("Sé conciso: máximo 3-4 oraciones por respuesta.\n")
	sb.WriteString("Si no tienes la información solicitada, sugiere contactar directamente al laboratorio.\n\n")

	// ── Información del negocio (desde BD o env fallback) ────────────────────
	sb.WriteString("--- INFORMACIÓN DEL LABORATORIO ---\n")

	whatsapp := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.Whatsapp }, "WHATSAPP_NUMBER")
	if whatsapp != "" {
		sb.WriteString(fmt.Sprintf("• WhatsApp / Teléfono: %s\n", whatsapp))
	}

	email := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.ContactEmail }, "CONTACT_EMAIL")
	if email != "" {
		sb.WriteString(fmt.Sprintf("• Email de contacto: %s\n", email))
	}

	address := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.Address }, "BUSINESS_ADDRESS")
	if address != "" {
		sb.WriteString(fmt.Sprintf("• Dirección: %s\n", address))
	}

	hours := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.BusinessHours }, "BUSINESS_HOURS")
	if hours != "" {
		sb.WriteString(fmt.Sprintf("• Horario de atención: %s\n", hours))
	}

	policyURL := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.PolicyURL }, "POLICY_URL")
	if policyURL != "" {
		sb.WriteString(fmt.Sprintf("• Política de tratamiento de datos (Ley 1581): %s\n", policyURL))
		sb.WriteString("  Si preguntan por la política de datos, indica esa URL.\n")
	}

	if cfg != nil && strings.TrimSpace(cfg.ExtraInfo) != "" {
		sb.WriteString("\n--- INFORMACIÓN ADICIONAL ---\n")
		sb.WriteString(cfg.ExtraInfo + "\n")
	}

	// ── Catálogo de servicios ────────────────────────────────────────────────
	sb.WriteString("\n--- CATÁLOGO DE SERVICIOS DEL LABORATORIO ---\n")
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

	sb.WriteString("\nPara agendar una cita, el usuario puede hacerlo desde el formulario en la página web.\n")

	s.cachedPrompt = sb.String()
	s.cacheExpiry = time.Now().Add(15 * time.Minute)

	return s.cachedPrompt, nil
}

// cfgVal retorna el valor del campo de la config si existe, o el env var como fallback.
func cfgVal(cfg *domain.ChatConfig, field func(*domain.ChatConfig) string, envKey string) string {
	if cfg != nil {
		if v := strings.TrimSpace(field(cfg)); v != "" {
			return v
		}
	}
	return os.Getenv(envKey)
}
