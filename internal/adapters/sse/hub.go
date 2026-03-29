package sse

import "sync"

// BannerHub gestiona los clientes SSE conectados a la landing page.
type BannerHub struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

func NewBannerHub() *BannerHub {
	return &BannerHub{
		clients: make(map[chan string]struct{}),
	}
}

func (h *BannerHub) Subscribe() chan string {
	ch := make(chan string, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *BannerHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *BannerHub) Broadcast(event string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
		}
	}
}
