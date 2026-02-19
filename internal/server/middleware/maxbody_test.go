package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBody_LimitsPostBody(t *testing.T) {
	const limit = 100

	handler := MaxBody(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		method         string
		bodySize       int
		expectedStatus int
	}{
		{
			name:           "POST within limit",
			method:         http.MethodPost,
			bodySize:       50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST at limit",
			method:         http.MethodPost,
			bodySize:       100,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST exceeds limit",
			method:         http.MethodPost,
			bodySize:       200,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "PUT within limit",
			method:         http.MethodPut,
			bodySize:       50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT exceeds limit",
			method:         http.MethodPut,
			bodySize:       200,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "GET no body limit",
			method:         http.MethodGet,
			bodySize:       0,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.Repeat("x", tt.bodySize)
			req := httptest.NewRequest(tt.method, "/test", bytes.NewBufferString(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestMaxBody_DefaultLimit(t *testing.T) {
	// When passed 0, should use default MaxBodySize (1MB)
	handler := MaxBody(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with a small body - should work
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("small body"))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
