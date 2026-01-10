package middleware

import (
	"log"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// Registry stores a rate limiter for each Host
type LimitRegistry struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

// Global Singleton
var GlobalLimitRegistry = &LimitRegistry{
	limiters: make(map[string]*rate.Limiter),
}

// UpdateConfig updates or creates a limiter for a specific host
func (r *LimitRegistry) UpdateConfig(host string, rps float64, burst int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// rps = 0 means infinite/disabled in some contexts, but here we assume strict config.
	// If burst is 0, we default to 1 so at least one request can pass.
	if burst == 0 {
		burst = 1
	}

	log.Printf("🔒 Rate Limit set for %s: %.2f RPS (Burst %d)", host, rps, burst)
	r.limiters[host] = rate.NewLimiter(rate.Limit(rps), burst)
}

// RemoveConfig removes a limiter (used during hot reload deletion)
func (r *LimitRegistry) RemoveConfig(host string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.limiters, host)
}

// Middleware enforces the limit
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host // e.g. "api_dynamic.localhost" or "api_dynamic.local"

		// 1. Check if a limiter exists for this host
		GlobalLimitRegistry.mu.RLock()
		limiter, exists := GlobalLimitRegistry.limiters[host]
		GlobalLimitRegistry.mu.RUnlock()

		// 2. If no limiter is configured for this API, allow traffic (Fail Open)
		if !exists {
			next.ServeHTTP(w, r)
			return
		}

		// 3. Try to take 1 token. Allow() returns false if empty.
		if !limiter.Allow() {
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			// Optional: Record a metric for dropped requests here
			return
		}

		next.ServeHTTP(w, r)
	})
}
