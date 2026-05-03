package worker

import (
	"dental-app/internal/core/ports"
	"log"
	"time"
)

// RetentionWorker anonimiza pacientes cuyo período de retención de datos venció
// según la política de privacidad (2 años desde último contacto).
type RetentionWorker struct {
	service ports.PatientService
}

func NewRetentionWorker(service ports.PatientService) *RetentionWorker {
	return &RetentionWorker{service: service}
}

func (w *RetentionWorker) Start() {
	log.Println("🗑️  Worker de retención iniciado — revisión diaria de datos expirados")
	ticker := time.NewTicker(24 * time.Hour)

	go func() {
		w.run()
		for range ticker.C {
			w.run()
		}
	}()
}

func (w *RetentionWorker) run() {
	n, err := w.service.AnonymizeExpired()
	if err != nil {
		log.Printf("❌ [RetentionWorker] Error anonimizando pacientes: %v", err)
		return
	}
	if n > 0 {
		log.Printf("🗑️  [RetentionWorker] %d paciente(s) anonimizados por vencimiento del período de retención (2 años)", n)
	}
}
