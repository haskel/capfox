package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	// RequestsPerSecond is the rate limit (requests per second).
	RequestsPerSecond float64
	// Burst is the maximum burst size.
	Burst int
	// Enabled controls whether rate limiting is active.
	Enabled bool
}

// RateLimit creates a middleware that limits request rate.
// Uses token bucket algorithm: allows bursts up to Burst size,
// refills at RequestsPerSecond rate.
func RateLimit(config *RateLimitConfig) Middleware {
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PerIPRateLimitConfig holds per-IP rate limiting configuration.
type PerIPRateLimitConfig struct {
	// RequestsPerSecond is the rate limit per IP.
	RequestsPerSecond float64
	// Burst is the maximum burst size per IP.
	Burst int
	// Enabled controls whether rate limiting is active.
	Enabled bool
}

// perIPLimiter manages rate limiters per client IP.
type perIPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newPerIPLimiter(rps float64, burst int) *perIPLimiter {
	return &perIPLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (l *perIPLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = limiter
	}
	return limiter
}

// PerIPRateLimit creates a middleware that limits request rate per client IP.
func PerIPRateLimit(config *PerIPRateLimitConfig) Middleware {
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	ipLimiter := newPerIPLimiter(config.RequestsPerSecond, config.Burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			limiter := ipLimiter.getLimiter(ip)

			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
