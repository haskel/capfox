package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimit_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		Enabled: false,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow unlimited requests when disabled
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_AllowsBurst(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		Burst:             5,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 5 requests (burst) should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("burst request %d: expected status 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_RejectsExcessRequests(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             2,
	}

	handler := RateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use all burst tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

func TestPerIPRateLimit_Disabled(t *testing.T) {
	config := &PerIPRateLimitConfig{
		Enabled: false,
	}

	handler := PerIPRateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow unlimited requests when disabled
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, w.Code)
		}
	}
}

func TestPerIPRateLimit_SeparateLimitersPerIP(t *testing.T) {
	config := &PerIPRateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             2,
	}

	handler := PerIPRateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP 1: exhaust burst
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// IP 1: should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 third request: expected status 429, got %d", w.Code)
	}

	// IP 2: should still be allowed (separate limiter)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("IP2 first request: expected status 200, got %d", w.Code)
	}
}

func TestPerIPRateLimit_XForwardedFor(t *testing.T) {
	config := &PerIPRateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
	}

	handler := PerIPRateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request with X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d", w.Code)
	}

	// Second request with same X-Forwarded-For should be limited
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected status 429, got %d", w.Code)
	}
}

func TestPerIPRateLimit_Concurrent(t *testing.T) {
	config := &PerIPRateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1000,
		Burst:             100,
	}

	handler := PerIPRateLimit(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	successCount := make(chan int, 50)

	// Concurrent requests from same IP
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				successCount <- 1
			}
		}()
	}

	wg.Wait()
	close(successCount)

	// Count successful requests
	total := 0
	for range successCount {
		total++
	}

	// With burst of 100, all 50 should succeed
	if total != 50 {
		t.Errorf("expected 50 successful requests, got %d", total)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For takes priority",
			xff:        "10.0.0.1",
			xri:        "10.0.0.2",
			remoteAddr: "10.0.0.3:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP second priority",
			xff:        "",
			xri:        "10.0.0.2",
			remoteAddr: "10.0.0.3:12345",
			expected:   "10.0.0.2",
		},
		{
			name:       "RemoteAddr fallback",
			xff:        "",
			xri:        "",
			remoteAddr: "10.0.0.3:12345",
			expected:   "10.0.0.3:12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, ip)
			}
		})
	}
}

func TestPerIPLimiter_Cleanup(t *testing.T) {
	limiter := newPerIPLimiter(100, 10)

	// Add some limiters
	for i := 0; i < 5; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.getLimiter(ip)
	}

	// Verify limiters were added
	limiter.mu.Lock()
	if len(limiter.limiters) != 5 {
		t.Errorf("expected 5 limiters, got %d", len(limiter.limiters))
	}

	// Set old last access times
	oldTime := time.Now().Add(-perIPMaxAge - time.Minute)
	for _, entry := range limiter.limiters {
		entry.lastAccess = oldTime
	}
	limiter.mu.Unlock()

	// Run cleanup
	limiter.cleanup()

	// All limiters should be removed
	limiter.mu.Lock()
	if len(limiter.limiters) != 0 {
		t.Errorf("expected 0 limiters after cleanup, got %d", len(limiter.limiters))
	}
	limiter.mu.Unlock()

	// Stop cleanup goroutine
	close(limiter.done)
}

func TestPerIPLimiter_EvictOldest(t *testing.T) {
	limiter := newPerIPLimiter(100, 10)

	// Add limiters with different access times
	now := time.Now()
	limiter.mu.Lock()
	for i := 0; i < 3; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i)
		limiter.limiters[ip] = &ipLimiterEntry{
			limiter:    nil,
			lastAccess: now.Add(time.Duration(i) * time.Second),
		}
	}

	// Evict oldest
	limiter.evictOldest()

	// Should have 2 limiters
	if len(limiter.limiters) != 2 {
		t.Errorf("expected 2 limiters, got %d", len(limiter.limiters))
	}

	// The oldest (192.168.1.0) should be gone
	if _, exists := limiter.limiters["192.168.1.0"]; exists {
		t.Error("oldest limiter should have been evicted")
	}
	limiter.mu.Unlock()

	close(limiter.done)
}
