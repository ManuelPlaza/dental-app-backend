package domain

type CancellationReason struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// GetCancellationReasons retorna el catálogo de motivos
func GetCancellationReasons() []CancellationReason {
	return []CancellationReason{
		{
			Code:        "no_show",
			Label:       "No se presentó",
			Description: "El paciente no llegó a la cita y no avisó",
		},
		{
			Code:        "patient_request",
			Label:       "Solicitud del paciente",
			Description: "El paciente canceló por decisión personal",
		},
		{
			Code:        "auto_expired",
			Label:       "Expiró sin confirmar",
			Description: "La cita no fue confirmada con al menos 1 hora de anticipación",
		},
		{
			Code:        "emergency",
			Label:       "Emergencia del paciente",
			Description: "El paciente tuvo una emergencia y no pudo asistir",
		},
		{
			Code:        "scheduling_conflict",
			Label:       "Conflicto de horario",
			Description: "El paciente tuvo un imprevisto en su agenda",
		},
		{
			Code:        "specialist_unavailable",
			Label:       "Especialista no disponible",
			Description: "El especialista asignado no pudo atender",
		},
		{
			Code:        "clinic_decision",
			Label:       "Decisión administrativa",
			Description: "La clínica debió cancelar por razones internas",
		},
		{
			Code:        "other",
			Label:       "Otro motivo",
			Description: "Motivo no listado, ver notas adicionales",
		},
	}
}
