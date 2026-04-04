package worker

import (
	"dental-app/internal/core/ports"
	"log"
	"time"
)

// AppointmentWorker cancela automáticamente citas pendientes cuyo horario ya pasó.
type AppointmentWorker struct {
	service ports.AppointmentService
}

func NewAppointmentWorker(service ports.AppointmentService) *AppointmentWorker {
	return &AppointmentWorker{service: service}
}

func (w *AppointmentWorker) Start() {
	log.Println("📅 Worker de citas iniciado — auto-cancelación cada 5 minutos")
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		w.run()
		for range ticker.C {
			w.run()
		}
	}()
}

func (w *AppointmentWorker) run() {
	n, err := w.service.AutoCancelExpired()
	if err != nil {
		log.Printf("❌ [AppointmentWorker] Error auto-cancelando citas: %v", err)
		return
	}
	if n > 0 {
		log.Printf("📅 [AppointmentWorker] %d cita(s) canceladas automáticamente por no confirmación", n)
	}
}
