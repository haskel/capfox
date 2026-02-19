package middleware

import "net/http"

// SecurityHeaders adds security-related HTTP headers to responses.
// Provides protection against common web vulnerabilities.
func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent caching of sensitive data
			w.Header().Set("Cache-Control", "no-store")

			// Enable XSS filter (legacy browsers)
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			next.ServeHTTP(w, r)
		})
	}
}
