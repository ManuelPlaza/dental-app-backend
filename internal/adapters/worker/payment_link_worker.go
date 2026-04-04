package worker

import (
	"dental-app/internal/core/ports"
	"log"
	"time"
)

// PaymentLinkWorker expira links vencidos cada minuto.
type PaymentLinkWorker struct {
	service ports.PaymentLinkService
}

func NewPaymentLinkWorker(service ports.PaymentLinkService) *PaymentLinkWorker {
	return &PaymentLinkWorker{service: service}
}

func (w *PaymentLinkWorker) Start() {
	log.Println("💳 Worker de links de pago iniciado — cada minuto")
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		w.run()
		for range ticker.C {
			w.run()
		}
	}()
}

func (w *PaymentLinkWorker) run() {
	if err := w.service.ExpireStale(); err != nil {
		log.Printf("❌ [PaymentLinkWorker] Error expirando links: %v", err)
	}
}
