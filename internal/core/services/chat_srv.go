package services

import (
	"dental-app/internal/adapters/groq"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// cedulaRe detecta cédulas colombianas (5-10 dígitos) en texto libre.
// El límite de 10 excluye teléfonos (11 dígitos) y el mínimo de 5 excluye años (4 dígitos).
var cedulaRe = regexp.MustCompile(`\b(\d{5,10})\b`)

type chatService struct {
	serviceRepo    ports.ServiceRepository
	chatConfigRepo ports.ChatConfigRepository
	groqClient     *groq.Client
	referralRepo   ports.ReferralGraphRepository
	cacheMu        sync.RWMutex
	cachedPrompt   string
	cacheExpiry    time.Time
}

func NewChatService(
	serviceRepo ports.ServiceRepository,
	chatConfigRepo ports.ChatConfigRepository,
	groqClient *groq.Client,
	referralRepo ports.ReferralGraphRepository,
) ports.ChatService {
	return &chatService{
		serviceRepo:    serviceRepo,
		chatConfigRepo: chatConfigRepo,
		groqClient:     groqClient,
		referralRepo:   referralRepo,
	}
}

func (s *chatService) Chat(messages []domain.ChatMessage) (string, error) {
	systemPrompt, err := s.buildSystemPrompt()
	if err != nil {
		return "", err
	}

	// Inyectar contexto de referidos si detectamos una cédula en los últimos mensajes
	if ctx := s.buildReferralContext(messages); ctx != "" {
		systemPrompt = ctx + "\n\n" + systemPrompt
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

// buildReferralContext busca una cédula en los últimos 3 mensajes del usuario
// y consulta ArangoDB para obtener su perfil en el grafo de referidos.
// Retorna un bloque de contexto listo para inyectar al system prompt, o "" si no aplica.
func (s *chatService) buildReferralContext(messages []domain.ChatMessage) string {
	if s.referralRepo == nil {
		return ""
	}

	// Buscar en los últimos 3 mensajes del usuario (más recientes primero)
	cedula := ""
	count := 0
	for i := len(messages) - 1; i >= 0 && count < 3; i-- {
		if messages[i].Role != "user" {
			continue
		}
		count++
		if m := cedulaRe.FindString(messages[i].Content); m != "" {
			cedula = m
			break
		}
	}
	if cedula == "" {
		return ""
	}

	status, err := s.referralRepo.GetPatientReferralStatus(cedula)
	if err != nil {
		log.Printf("⚠️ chatbot referral lookup %s: %v", cedula, err)
		return ""
	}
	if status == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== CONTEXTO DEL PACIENTE IDENTIFICADO EN ESTE CHAT ===\n")
	sb.WriteString(fmt.Sprintf("Cédula detectada en la conversación: %s\n", cedula))

	if status.ReferralCount > 0 {
		sb.WriteString(fmt.Sprintf(
			"→ Este paciente ha referido a %d %s al laboratorio. Es un referidor activo — puedes agradecerle su confianza y mencionarlo si es pertinente.\n",
			status.ReferralCount,
			map[bool]string{true: "persona", false: "personas"}[status.ReferralCount == 1],
		))
	} else {
		sb.WriteString("→ Este paciente no ha referido a nadie aún.\n")
	}

	if status.ReferredByName != "" {
		sb.WriteString(fmt.Sprintf(
			"→ Llegó al laboratorio referido por: %s. Puedes mencionarlo para personalizar la bienvenida.\n",
			status.ReferredByName,
		))
	} else {
		sb.WriteString("→ No fue referido por otro paciente (vino por cuenta propia).\n")
	}

	sb.WriteString("Usa este contexto para personalizar tu respuesta si es relevante, pero sin revelar datos sensibles.\n")
	sb.WriteString("========================================================\n")

	return sb.String()
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
