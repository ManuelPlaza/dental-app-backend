package worker

import (
	"dental-app/internal/core/ports"
	"log"
	"time"
)

type NotificationWorker struct {
	service ports.NotificationService
}

func NewNotificationWorker(service ports.NotificationService) *NotificationWorker {
	return &NotificationWorker{service: service}
}

func (w *NotificationWorker) Start() {
	log.Println("🔔 Worker de notificaciones iniciado — cada minuto")
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		w.run()
		for range ticker.C {
			w.run()
		}
	}()
}

func (w *NotificationWorker) run() {
	if err := w.service.ProcessPending(); err != nil {
		log.Printf("❌ Worker error: %s", err.Error())
	}
}
