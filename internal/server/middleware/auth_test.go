package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_Disabled(t *testing.T) {
	config := &AuthConfig{
		Enabled: false,
	}

	handler := Auth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuth_ValidCredentials(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		User:     "admin",
		Password: "secret",
	}

	handler := Auth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("admin", "secret")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuth_InvalidCredentials(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		User:     "admin",
		Password: "secret",
	}

	handler := Auth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("admin", "wrong")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestAuth_NoCredentials(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		User:     "admin",
		Password: "secret",
	}

	handler := Auth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAuth_ExcludedPath(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		User:     "admin",
		Password: "secret",
	}

	handler := Auth(config, "/health")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request to excluded path without auth should succeed
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for excluded path, got %d", w.Code)
	}
}

func TestAuth_ExcludedPath_OtherPathsProtected(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		User:     "admin",
		Password: "secret",
	}

	handler := Auth(config, "/health")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request to non-excluded path without auth should fail
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for protected path, got %d", w.Code)
	}
}

func TestAuth_WrongUser(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		User:     "admin",
		Password: "secret",
	}

	handler := Auth(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("wronguser", "secret")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}
