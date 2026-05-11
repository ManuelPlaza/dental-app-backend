package services

import (
	"dental-app/internal/adapters/groq"
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"encoding/json"
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

// opcionesRe extrae el bloque %%OPCIONES%%[...] que el LLM añade al final de la respuesta.
var opcionesRe = regexp.MustCompile(`(?m)\s*%%OPCIONES%%(\[.*?\])\s*$`)

// parseQuickReplies separa el texto limpio del bloque %%OPCIONES%%[...].
// Si el bloque no existe o el JSON es inválido, retorna el texto intacto y replies vacío.
func parseQuickReplies(raw string) (text string, replies []string) {
	m := opcionesRe.FindStringSubmatch(raw)
	if m == nil {
		return strings.TrimSpace(raw), []string{}
	}
	if err := json.Unmarshal([]byte(m[1]), &replies); err != nil {
		return strings.TrimSpace(raw), []string{}
	}
	return strings.TrimSpace(opcionesRe.ReplaceAllString(raw, "")), replies
}

// ── Definición de herramientas ────────────────────────────────────────────────

var appointmentTools = []groq.Tool{
	{
		Type: "function",
		Function: groq.ToolFunction{
			Name:        "buscar_citas",
			Description: "Busca las citas activas (pendientes o agendadas) de un paciente. REQUIERE verificar identidad: proporciona el documento y el nombre completo o teléfono que el paciente dijo en el chat.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"document_number": map[string]interface{}{
						"type":        "string",
						"description": "Número de cédula del paciente",
					},
					"verificacion": map[string]interface{}{
						"type":        "string",
						"description": "Nombre completo o teléfono que el paciente proporcionó para verificar identidad",
					},
				},
				"required": []string{"document_number", "verificacion"},
			},
		},
	},
	{
		Type: "function",
		Function: groq.ToolFunction{
			Name:        "confirmar_cita",
			Description: "Confirma una cita que está en estado pendiente. Solo usar después de que buscar_citas haya verificado la identidad del paciente.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cita_id": map[string]interface{}{
						"type":        "integer",
						"description": "ID numérico de la cita a confirmar",
					},
					"document_number": map[string]interface{}{
						"type":        "string",
						"description": "Cédula del paciente (para verificar propiedad de la cita)",
					},
				},
				"required": []string{"cita_id", "document_number"},
			},
		},
	},
	{
		Type: "function",
		Function: groq.ToolFunction{
			Name:        "cancelar_cita",
			Description: "Cancela una cita pendiente o agendada. Solo usar después de que buscar_citas haya verificado la identidad del paciente y el paciente haya confirmado explícitamente que desea cancelar.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cita_id": map[string]interface{}{
						"type":        "integer",
						"description": "ID numérico de la cita a cancelar",
					},
					"document_number": map[string]interface{}{
						"type":        "string",
						"description": "Cédula del paciente (para verificar propiedad de la cita)",
					},
				},
				"required": []string{"cita_id", "document_number"},
			},
		},
	},
}

// ── Servicio ──────────────────────────────────────────────────────────────────

type chatService struct {
	serviceRepo     ports.ServiceRepository
	chatConfigRepo  ports.ChatConfigRepository
	groqClient      *groq.Client
	referralRepo    ports.ReferralGraphRepository
	patientRepo     ports.PatientRepository
	appointmentRepo ports.AppointmentRepository
	cacheMu         sync.RWMutex
	cachedPrompt    string
	cacheExpiry     time.Time
}

func NewChatService(
	serviceRepo ports.ServiceRepository,
	chatConfigRepo ports.ChatConfigRepository,
	groqClient *groq.Client,
	referralRepo ports.ReferralGraphRepository,
	patientRepo ports.PatientRepository,
	appointmentRepo ports.AppointmentRepository,
) ports.ChatService {
	return &chatService{
		serviceRepo:     serviceRepo,
		chatConfigRepo:  chatConfigRepo,
		groqClient:      groqClient,
		referralRepo:    referralRepo,
		patientRepo:     patientRepo,
		appointmentRepo: appointmentRepo,
	}
}

// appointmentKeywords son términos que indican que el usuario quiere gestionar citas existentes.
// Si ninguno aparece en las últimas 4 interacciones, no se envían las herramientas a Groq.
var appointmentKeywords = []string{
	"cita", "citas", "consultar", "confirmar", "cancelar", "gestionar",
	"cedula", "cédula", "celular", "teléfono", "telefono", "nombre",
	"pendiente", "agendada", "verificar", cedulaRe.String(),
}

// needsTools retorna true si el historial reciente sugiere que el usuario
// quiere gestionar citas existentes (no solo preguntar por servicios/horarios).
func needsTools(messages []domain.ChatMessage) bool {
	// Revisar las últimas 4 interacciones
	start := len(messages) - 4
	if start < 0 {
		start = 0
	}
	for _, m := range messages[start:] {
		lower := strings.ToLower(m.Content)
		for _, kw := range appointmentKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		// Detectar cédula (número 5-10 dígitos)
		if cedulaRe.MatchString(lower) {
			return true
		}
	}
	return false
}

func (s *chatService) Chat(messages []domain.ChatMessage) (domain.ChatResponse, error) {
	systemPrompt, err := s.buildSystemPrompt()
	if err != nil {
		return domain.ChatResponse{}, err
	}

	// Contexto de referidos si se detecta cédula
	if ctx := s.buildReferralContext(messages); ctx != "" {
		systemPrompt = ctx + "\n\n" + systemPrompt
	}

	if len(messages) > 6 {
		messages = messages[len(messages)-6:]
	}

	groqMessages := make([]groq.Message, len(messages))
	for i, m := range messages {
		groqMessages[i] = groq.Message{Role: m.Role, Content: m.Content}
	}

	var raw string
	// Solo enviar tools (y pagar el costo de múltiples rondas) si el contexto lo requiere
	if needsTools(messages) {
		raw, err = s.groqClient.ChatWithTools(systemPrompt, groqMessages, appointmentTools, s.execTool)
	} else {
		raw, err = s.groqClient.Chat(systemPrompt, groqMessages)
	}
	if err != nil {
		log.Printf("[chatbot] error Groq: %v", err)
		msg := "Lo siento, en este momento tengo dificultades para responder. Por favor intenta de nuevo en un momento."
		if strings.HasPrefix(err.Error(), "rate_limit:") {
			msg = "El asistente está recibiendo muchas consultas en este momento. Espera unos segundos e intenta de nuevo. 🙏"
		}
		return domain.ChatResponse{Response: msg, QuickReplies: []string{}}, nil
	}

	text, qr := parseQuickReplies(raw)
	return domain.ChatResponse{Response: text, QuickReplies: qr}, nil
}

func (s *chatService) InvalidateCache() {
	s.cacheMu.Lock()
	s.cacheExpiry = time.Time{}
	s.cacheMu.Unlock()
}

// ── Ejecución de herramientas ─────────────────────────────────────────────────

func (s *chatService) execTool(name, argsJSON string) string {
	switch name {
	case "buscar_citas":
		var args struct {
			DocumentNumber string `json:"document_number"`
			Verificacion   string `json:"verificacion"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return errJSON("JSON inválido")
		}
		return s.toolBuscarCitas(args.DocumentNumber, args.Verificacion)

	case "confirmar_cita":
		var args struct {
			CitaID         int    `json:"cita_id"`
			DocumentNumber string `json:"document_number"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return errJSON("JSON inválido")
		}
		return s.toolConfirmarCita(uint(args.CitaID), args.DocumentNumber)

	case "cancelar_cita":
		var args struct {
			CitaID         int    `json:"cita_id"`
			DocumentNumber string `json:"document_number"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return errJSON("JSON inválido")
		}
		return s.toolCancelarCita(uint(args.CitaID), args.DocumentNumber)
	}
	return errJSON("herramienta desconocida: " + name)
}

func (s *chatService) toolBuscarCitas(doc, verificacion string) string {
	if doc == "" || verificacion == "" {
		return errJSON("Necesito tanto la cédula como el dato de verificación (nombre completo o teléfono)")
	}

	// Rechazar si el LLM pasó la cédula misma como verificación
	verTrimmed := strings.TrimSpace(verificacion)
	docTrimmed := strings.TrimSpace(doc)
	if verTrimmed == docTrimmed {
		return errJSON("El dato de verificación no puede ser la misma cédula. Solicita al paciente su nombre completo o número de celular registrado.")
	}
	// Detectar si el LLM confundió celular con cédula (10 dígitos que empieza por 3)
	if len(docTrimmed) == 10 && docTrimmed[0] == '3' {
		return errJSON("Ese número parece ser un celular, no una cédula. Por favor solicita al paciente su número de cédula de ciudadanía.")
	}

	// 1. Buscar paciente en PostgreSQL
	patient, err := s.patientRepo.FindByDocumentNumber(doc)
	if err != nil {
		log.Printf("[chatbot] buscar_citas: paciente doc=%s no encontrado: %v", doc, err)
		return errJSON("No encontré ningún paciente registrado con esa cédula. Verifica que el número sea correcto.")
	}

	// 2. Verificar identidad
	if !verifyIdentity(verificacion, patient) {
		log.Printf("[chatbot] buscar_citas: verificación fallida doc=%s verificacion=%q fn=%q ln=%q phone=%q",
			doc, verificacion, patient.FirstName, patient.LastName, patient.Phone)
		return errJSON("Los datos de verificación no coinciden con los registrados. Por seguridad no puedo mostrar la información.")
	}

	// 3. Consultar citas activas
	appointments, err := s.appointmentRepo.GetActiveByPatientDocument(doc)
	if err != nil {
		log.Printf("[chatbot] buscar_citas: error consultando citas doc=%s: %v", doc, err)
		return errJSON("Error consultando las citas. Intenta de nuevo en un momento.")
	}

	if len(appointments) == 0 {
		return okJSON(map[string]interface{}{
			"verificado":    true,
			"primer_nombre": patient.FirstName,
			"citas":         []interface{}{},
			"mensaje":       "No tienes citas pendientes ni agendadas en este momento.",
		})
	}

	// 4. Formatear citas — solo datos de la cita, sin PII del paciente
	bogota, _ := time.LoadLocation("America/Bogota")
	citas := make([]map[string]interface{}, 0, len(appointments))
	for _, a := range appointments {
		start := a.StartTime.Time.In(bogota)
		c := map[string]interface{}{
			"id":           a.ID,
			"servicio":     a.Service.Name,
			"fecha":        start.Format("02/01/2006"),
			"hora":         start.Format("03:04 PM"),
			"estado":       statusLabel(a.Status),
			"estado_code":  a.Status,
			"especialista": a.Specialist.FirstName + " " + a.Specialist.LastName,
		}
		citas = append(citas, c)
	}

	return okJSON(map[string]interface{}{
		"verificado":    true,
		"primer_nombre": patient.FirstName,
		"citas":         citas,
	})
}

func (s *chatService) toolConfirmarCita(citaID uint, doc string) string {
	appt, err := s.appointmentRepo.GetByID(citaID)
	if err != nil {
		return errJSON("No encontré la cita #" + fmt.Sprint(citaID))
	}

	// Verificar propiedad
	if appt.Patient.DocumentNumber != doc {
		return errJSON("Esa cita no corresponde a tu número de cédula.")
	}
	if appt.Status != "pending" {
		bogota, _ := time.LoadLocation("America/Bogota")
		start := appt.StartTime.Time.In(bogota)
		return errJSON(fmt.Sprintf(
			"La cita del %s a las %s ya está en estado '%s' y no puede confirmarse.",
			start.Format("02/01/2006"), start.Format("03:04 PM"), statusLabel(appt.Status),
		))
	}

	// Confirmar: pending → scheduled
	appt.Status = "scheduled"
	if err := s.appointmentRepo.Update(appt); err != nil {
		return errJSON("No se pudo confirmar la cita. Intenta más tarde.")
	}

	bogota, _ := time.LoadLocation("America/Bogota")
	start := appt.StartTime.Time.In(bogota)
	log.Printf("[chatbot:audit] CONFIRMAR cita id=%d doc=%s fecha=%s", appt.ID, doc, start.Format("2006-01-02 15:04"))
	return okJSON(map[string]interface{}{
		"success": true,
		"mensaje": fmt.Sprintf("✅ Cita confirmada para el %s a las %s. ¡Te esperamos!",
			start.Format("02 de January de 2006"), start.Format("03:04 PM")),
	})
}

func (s *chatService) toolCancelarCita(citaID uint, doc string) string {
	appt, err := s.appointmentRepo.GetByID(citaID)
	if err != nil {
		return errJSON("No encontré la cita #" + fmt.Sprint(citaID))
	}

	// Verificar propiedad
	if appt.Patient.DocumentNumber != doc {
		return errJSON("Esa cita no corresponde a tu número de cédula.")
	}
	if appt.Status != "pending" && appt.Status != "scheduled" {
		return errJSON("La cita ya está en estado '" + statusLabel(appt.Status) + "' y no puede cancelarse.")
	}

	bogota, _ := time.LoadLocation("America/Bogota")
	start := appt.StartTime.Time.In(bogota)

	appt.Status = "cancelled"
	appt.CancellationReason = "patient_request"
	appt.CancellationNotes = "Cancelada por el paciente vía chatbot"
	if err := s.appointmentRepo.Update(appt); err != nil {
		return errJSON("No se pudo cancelar la cita. Intenta más tarde.")
	}

	log.Printf("[chatbot:audit] CANCELAR cita id=%d doc=%s fecha=%s", appt.ID, doc, start.Format("2006-01-02 15:04"))
	return okJSON(map[string]interface{}{
		"success": true,
		"mensaje": fmt.Sprintf("❌ Cita del %s a las %s cancelada correctamente. Si necesitas reagendar, puedes hacerlo desde nuestro sitio web.",
			start.Format("02/01/2006"), start.Format("03:04 PM")),
	})
}

// ── Contexto de referidos (feature anterior) ──────────────────────────────────

func (s *chatService) buildReferralContext(messages []domain.ChatMessage) string {
	if s.referralRepo == nil {
		return ""
	}
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
	sb.WriteString(fmt.Sprintf("Cédula detectada: %s\n", cedula))
	if status.ReferralCount > 0 {
		sb.WriteString(fmt.Sprintf("→ Ha referido a %d %s al laboratorio.\n",
			status.ReferralCount,
			map[bool]string{true: "persona", false: "personas"}[status.ReferralCount == 1]))
	}
	if status.ReferredByName != "" {
		sb.WriteString(fmt.Sprintf("→ Fue referido por: %s.\n", status.ReferredByName))
	}
	sb.WriteString("========================================================\n")
	return sb.String()
}

// ── System prompt ─────────────────────────────────────────────────────────────

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

	sb.WriteString(fmt.Sprintf("Eres el asistente virtual de %s (%s). Responde en español, amigable y conciso (máx 3 oraciones).\n", clinicName, businessType))
	sb.WriteString("REGLAS: No eres clínica ni das diagnósticos. Si preguntan síntomas, remite al odontólogo.\n\n")

	// ── Quick replies ─────────────────────────────────────────────────────────
	sb.WriteString("QUICK REPLIES: Al final de respuestas con opciones agrega exactamente: %%OPCIONES%%[\"op1\",\"op2\"]\n")
	sb.WriteString("Usa %%OPCIONES%% SOLO en: saludo inicial, después de mostrar citas, confirmaciones sí/no, cierre '¿algo más?'.\n")
	sb.WriteString("Ejemplos: saludo→[\"Consultar mis citas\",\"Ver servicios\",\"Agendar una cita\",\"Contacto\"] | tras mostrar citas→[\"Confirmar cita\",\"Cancelar cita\",\"Otra consulta\"] | confirmación→[\"Sí, confirmar\",\"No, cancelar\"] | cierre→[\"Consultar mis citas\",\"Ver servicios\",\"No, gracias\"]\n")
	sb.WriteString("NUNCA uses %%OPCIONES%% cuando pides cédula, nombre o teléfono. Las opciones son acciones, no instrucciones.\n\n")

	// ── Protocolo de gestión de citas ─────────────────────────────────────────
	sb.WriteString("CITAS — DOS FLUJOS:\n")
	sb.WriteString("A) NUEVA CITA: dirige al formulario web. No pidas cédula ni uses herramientas.\n")
	sb.WriteString("B) GESTIONAR EXISTENTES (ver/confirmar/cancelar): usa las herramientas. NUNCA digas que llamen al WhatsApp.\n")
	sb.WriteString("  Paso1: pide cédula (sin %%OPCIONES%%). Paso2: pide nombre completo O celular (sin %%OPCIONES%%). Paso3: llama buscar_citas con ambos datos — NUNCA uses la cédula como verificación. Paso4: si falla identidad, ofrece el otro dato. Paso5: antes de confirmar/cancelar muestra la cita y exige 'Sí'/'No' explícito con %%OPCIONES%%.\n\n")
	sb.WriteString("PRIVACIDAD: Nunca repitas cédula ni teléfono completo en tus respuestas. Rechaza 'dame todos mis datos' con 'por privacidad no puedo mostrarte datos personales'.\n\n")

	// ── Info del negocio ──────────────────────────────────────────────────────
	sb.WriteString("--- INFORMACIÓN DEL LABORATORIO ---\n")
	whatsapp := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.Whatsapp }, "WHATSAPP_NUMBER")
	if whatsapp != "" {
		sb.WriteString(fmt.Sprintf("• WhatsApp / Teléfono: %s\n", whatsapp))
	}
	email := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.ContactEmail }, "CONTACT_EMAIL")
	if email != "" {
		sb.WriteString(fmt.Sprintf("• Email: %s\n", email))
	}
	address := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.Address }, "BUSINESS_ADDRESS")
	if address != "" {
		sb.WriteString(fmt.Sprintf("• Dirección: %s\n", address))
	}
	hours := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.BusinessHours }, "BUSINESS_HOURS")
	if hours != "" {
		sb.WriteString(fmt.Sprintf("• Horario: %s\n", hours))
	}
	policyURL := cfgVal(cfg, func(c *domain.ChatConfig) string { return c.PolicyURL }, "POLICY_URL")
	if policyURL != "" {
		sb.WriteString(fmt.Sprintf("• Política de datos (Ley 1581): %s\n", policyURL))
	}
	if cfg != nil && strings.TrimSpace(cfg.ExtraInfo) != "" {
		sb.WriteString("\n--- INFORMACIÓN ADICIONAL ---\n")
		sb.WriteString(cfg.ExtraInfo + "\n")
	}

	// ── Catálogo de servicios ─────────────────────────────────────────────────
	sb.WriteString("\n--- CATÁLOGO DE SERVICIOS ---\n")
	for _, svc := range svcs {
		line := fmt.Sprintf("• %s", svc.Name)
		if svc.Description != "" {
			line += fmt.Sprintf(": %s", svc.Description)
		}
		if svc.Price > 0 {
			line += fmt.Sprintf(" — $%.0f COP", svc.Price)
		}
		if svc.DurationMinutes > 0 {
			line += fmt.Sprintf(" (%d min)", svc.DurationMinutes)
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString("\nPara agendar una cita nueva, el paciente puede hacerlo desde el formulario en la página web.\n")

	s.cachedPrompt = sb.String()
	s.cacheExpiry = time.Now().Add(15 * time.Minute)
	return s.cachedPrompt, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// verifyIdentity comprueba si el texto proporcionado por el paciente
// contiene su nombre (primer nombre, apellido) o su teléfono registrado.
func verifyIdentity(provided string, patient *domain.Patient) bool {
	p := strings.ToLower(strings.TrimSpace(provided))
	fn := strings.ToLower(strings.TrimSpace(patient.FirstName))
	ln := strings.ToLower(strings.TrimSpace(patient.LastName))

	nameMatch := (fn != "" && strings.Contains(p, fn)) &&
		(ln != "" && strings.Contains(p, ln))
	phoneMatch := patient.Phone != "" && strings.Contains(p, strings.TrimSpace(patient.Phone))

	return nameMatch || phoneMatch
}

func statusLabel(s string) string {
	switch s {
	case "pending":
		return "pendiente de confirmación"
	case "scheduled":
		return "confirmada"
	case "completed":
		return "completada"
	case "cancelled":
		return "cancelada"
	default:
		return s
	}
}

func errJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

func okJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func cfgVal(cfg *domain.ChatConfig, field func(*domain.ChatConfig) string, envKey string) string {
	if cfg != nil {
		if v := strings.TrimSpace(field(cfg)); v != "" {
			return v
		}
	}
	return os.Getenv(envKey)
}
