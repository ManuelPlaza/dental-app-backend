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

func (s *chatService) Chat(messages []domain.ChatMessage) (domain.ChatResponse, error) {
	systemPrompt, err := s.buildSystemPrompt()
	if err != nil {
		return domain.ChatResponse{}, err
	}

	// Contexto de referidos si se detecta cédula
	if ctx := s.buildReferralContext(messages); ctx != "" {
		systemPrompt = ctx + "\n\n" + systemPrompt
	}

	if len(messages) > 12 {
		messages = messages[len(messages)-12:]
	}

	groqMessages := make([]groq.Message, len(messages))
	for i, m := range messages {
		groqMessages[i] = groq.Message{Role: m.Role, Content: m.Content}
	}

	raw, err := s.groqClient.ChatWithTools(systemPrompt, groqMessages, appointmentTools, s.execTool)
	if err != nil {
		return domain.ChatResponse{}, err
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
	if verTrimmed == strings.TrimSpace(doc) {
		return errJSON("El dato de verificación no puede ser la misma cédula. Solicita al paciente su nombre completo o número de celular registrado.")
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

	sb.WriteString(fmt.Sprintf("Eres el asistente virtual de %s, un %s.\n", clinicName, businessType))
	sb.WriteString("RESTRICCIONES LEGALES — NUNCA las ignores:\n")
	sb.WriteString("1. No te describas como clínica, consultorio, centro médico ni institución de salud.\n")
	sb.WriteString("2. No des diagnósticos, recomendaciones clínicas ni consejos sobre tratamientos odontológicos.\n")
	sb.WriteString("3. Si preguntan sobre síntomas o dolor dental, indica que consulten a su odontólogo tratante.\n")
	sb.WriteString("4. Solo informa sobre servicios técnicos del laboratorio, contacto y agendamiento.\n\n")
	sb.WriteString("Responde siempre en español, de forma amigable, clara y profesional.\n")
	sb.WriteString("Sé conciso: máximo 3-4 oraciones por respuesta.\n\n")

	// ── Quick replies ─────────────────────────────────────────────────────────
	sb.WriteString("=== OPCIONES DE RESPUESTA RÁPIDA ===\n")
	sb.WriteString("Cuando presentes opciones al usuario, DEBES terminar tu respuesta con una nueva línea que contenga EXACTAMENTE este bloque (sin espacios extra, sin texto adicional después):\n")
	sb.WriteString("%%OPCIONES%%[\"Opción 1\",\"Opción 2\",\"Opción 3\"]\n\n")
	sb.WriteString("CUÁNDO incluir el bloque %%OPCIONES%%:\n")
	sb.WriteString("• Saludo o inicio de chat → %%OPCIONES%%[\"Consultar mis citas\",\"Ver servicios\",\"Información de contacto\",\"Agendar una cita\"]\n")
	sb.WriteString("• Cuando el paciente ya está verificado y pregunta por sus citas → %%OPCIONES%%[\"Confirmar cita\",\"Cancelar cita\",\"Ver mis citas\"]\n")
	sb.WriteString("• Cuando preguntas '¿Confirmas que deseas [acción]...?' → %%OPCIONES%%[\"Sí, confirmar\",\"No, cancelar\"]\n")
	sb.WriteString("• Cuando preguntas '¿Necesitas algo más?' → %%OPCIONES%%[\"Ver mis citas\",\"Hablar con un asesor\",\"No, gracias\"]\n")
	sb.WriteString("CUÁNDO NO incluir el bloque (el usuario DEBE escribir):\n")
	sb.WriteString("• Cuando pides la cédula al paciente\n")
	sb.WriteString("• Cuando pides nombre completo o teléfono para verificar identidad\n")
	sb.WriteString("REGLA: El bloque %%OPCIONES%%[...] va siempre en la última línea. El usuario nunca lo ve — el sistema lo procesa automáticamente.\n\n")

	// ── Protocolo de gestión de citas ─────────────────────────────────────────
	sb.WriteString("=== GESTIÓN DE CITAS — FUNCIONALIDAD PRINCIPAL DEL CHATBOT ===\n")
	sb.WriteString("TIENES ACCESO DIRECTO a las citas de los pacientes mediante herramientas (buscar_citas, confirmar_cita, cancelar_cita).\n")
	sb.WriteString("⚠️ PROHIBIDO: Jamás digas al paciente que vaya al WhatsApp, llame o escriba un correo para consultar sus citas.\n")
	sb.WriteString("⚠️ PROHIBIDO: Nunca respondas preguntas sobre 'mis citas', 'consultar cita', 'confirmar cita' o 'cancelar cita' sin usar las herramientas.\n")
	sb.WriteString("✅ CORRECTO: Cuando el paciente pregunta por sus citas, SIEMPRE inicia el protocolo de verificación de identidad.\n\n")
	sb.WriteString("PROTOCOLO OBLIGATORIO:\n")
	sb.WriteString("PASO 1: Si el paciente quiere gestionar citas y no ha dado su cédula, responde SOLO: '¡Con gusto! Para proteger tu información, necesito verificar tu identidad. ¿Cuál es tu número de cédula?'\n")
	sb.WriteString("PASO 2: Con la cédula, pregunta: 'Gracias. Ahora confírmame tu nombre completo o tu número de celular registrado en el sistema.'\n")
	sb.WriteString("PASO 3: Con cédula + verificación, llama a buscar_citas. NO llames esta herramienta sin ambos datos.\n")
	sb.WriteString("PASO 4: Si la verificación falla (error de identidad), informa que los datos no coinciden y ofrece intentar con otro dato.\n")
	sb.WriteString("PASO 5: Para confirmar o cancelar, muestra la cita y espera respuesta afirmativa EXPLÍCITA antes de llamar la herramienta. Di: '¿Confirmas que deseas [acción] tu cita del [fecha] a las [hora]? Responde SÍ o NO.' Si el paciente no confirma explícitamente, NO llames la herramienta.\n")
	sb.WriteString("PASO 6: Nunca reveles datos de otros pacientes. Nunca operes sobre citas de una cédula diferente.\n\n")
	sb.WriteString("=== PRIVACIDAD ESTRICTA — OBLIGATORIO ===\n")
	sb.WriteString("• NUNCA repitas la cédula del paciente en tus respuestas. Si necesitas referirte al paciente, usa su nombre.\n")
	sb.WriteString("• NUNCA muestres el número de teléfono completo. Si debes referirte a él, usa solo los últimos 4 dígitos (****XXXX).\n")
	sb.WriteString("• NUNCA respondas preguntas como 'dame toda mi información', 'qué datos tienes de mí', 'cuál es mi cédula'. Responde: 'Por tu privacidad, no puedo mostrar tus datos personales. Si necesitas actualizar tu información, comunícate con el laboratorio.'\n")
	sb.WriteString("• Solo muestra información de citas (fecha, hora, servicio, especialista, estado). Nada más.\n\n")

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
