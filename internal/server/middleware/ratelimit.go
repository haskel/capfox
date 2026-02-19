package middleware

import (
	"net/http"
	"sync"
	"time"

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

const (
	// perIPCleanupInterval is how often to clean up stale IP limiters.
	perIPCleanupInterval = 5 * time.Minute
	// perIPMaxAge is the max age for an unused IP limiter before cleanup.
	perIPMaxAge = 10 * time.Minute
	// perIPMaxEntries is the maximum number of IP limiters to keep.
	perIPMaxEntries = 10000
)

// ipLimiterEntry holds a rate limiter and its last access time.
type ipLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// perIPLimiter manages rate limiters per client IP.
type perIPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiterEntry
	rps      rate.Limit
	burst    int
	done     chan struct{}
}

func newPerIPLimiter(rps float64, burst int) *perIPLimiter {
	l := &perIPLimiter{
		limiters: make(map[string]*ipLimiterEntry),
		rps:      rate.Limit(rps),
		burst:    burst,
		done:     make(chan struct{}),
	}
	go l.cleanupLoop()
	return l
}

func (l *perIPLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, exists := l.limiters[ip]
	if !exists {
		entry = &ipLimiterEntry{
			limiter:    rate.NewLimiter(l.rps, l.burst),
			lastAccess: now,
		}
		l.limiters[ip] = entry

		// Evict oldest if over limit
		if len(l.limiters) > perIPMaxEntries {
			l.evictOldest()
		}
	} else {
		entry.lastAccess = now
	}
	return entry.limiter
}

// cleanupLoop periodically removes stale limiters.
func (l *perIPLimiter) cleanupLoop() {
	ticker := time.NewTicker(perIPCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.done:
			return
		}
	}
}

// cleanup removes limiters that haven't been used recently.
func (l *perIPLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-perIPMaxAge)
	for ip, entry := range l.limiters {
		if entry.lastAccess.Before(cutoff) {
			delete(l.limiters, ip)
		}
	}
}

// evictOldest removes the oldest limiter entry.
func (l *perIPLimiter) evictOldest() {
	var oldestIP string
	var oldestTime time.Time

	for ip, entry := range l.limiters {
		if oldestIP == "" || entry.lastAccess.Before(oldestTime) {
			oldestIP = ip
			oldestTime = entry.lastAccess
		}
	}

	if oldestIP != "" {
		delete(l.limiters, oldestIP)
	}
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
