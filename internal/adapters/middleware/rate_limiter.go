package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
	// Limpiar entradas expiradas cada minuto
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for key, attempts := range rl.attempts {
		valid := attempts[:0]
		for _, t := range attempts {
			if now.Sub(t) < rl.window {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.attempts, key)
		} else {
			rl.attempts[key] = valid
		}
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Filtrar intentos dentro de la ventana
	valid := rl.attempts[key][:0]
	for _, t := range rl.attempts[key] {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		rl.attempts[key] = valid
		return false
	}

	rl.attempts[key] = append(valid, now)
	return true
}

// RateLimitMiddleware limita peticiones por IP.
// max: numero maximo de peticiones permitidas en la ventana.
// window: duracion de la ventana de tiempo.
func RateLimitMiddleware(max int, window time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(max, window)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "demasiados intentos, intente de nuevo más tarde",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
