package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebugAuth_BearerToken(t *testing.T) {
	config := &DebugAuthConfig{
		Token: "secret-debug-token",
	}

	handler := DebugAuth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token",
			authHeader:     "Bearer secret-debug-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "wrong auth type",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "bearer without token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDebugAuth_FallbackToBasicAuth(t *testing.T) {
	config := &DebugAuthConfig{
		Token: "", // No token, fall back to basic auth
		FallbackAuthConfig: &AuthConfig{
			Enabled:  true,
			User:     "admin",
			Password: "secret",
		},
	}

	handler := DebugAuth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		setBasicAuth   bool
		user           string
		password       string
		expectedStatus int
	}{
		{
			name:           "valid basic auth",
			setBasicAuth:   true,
			user:           "admin",
			password:       "secret",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid basic auth",
			setBasicAuth:   true,
			user:           "admin",
			password:       "wrong",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing basic auth",
			setBasicAuth:   false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
			if tt.setBasicAuth {
				req.SetBasicAuth(tt.user, tt.password)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestDebugAuth_NoAuthConfigured(t *testing.T) {
	config := &DebugAuthConfig{
		Token:              "",
		FallbackAuthConfig: nil,
	}

	handler := DebugAuth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestDebugAuth_FallbackAuthDisabled(t *testing.T) {
	config := &DebugAuthConfig{
		Token: "",
		FallbackAuthConfig: &AuthConfig{
			Enabled:  false, // Auth disabled
			User:     "admin",
			Password: "secret",
		},
	}

	handler := DebugAuth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should be forbidden since no auth is configured
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}
