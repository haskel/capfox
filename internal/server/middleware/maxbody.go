package middleware

import (
	"net/http"
)

// MaxBodySize is the default maximum request body size (1 MB).
const MaxBodySize = 1 << 20 // 1 MB

// MaxBody creates a middleware that limits the request body size.
// If maxSize is 0, uses MaxBodySize constant (1 MB).
func MaxBody(maxSize int64) Middleware {
	if maxSize <= 0 {
		maxSize = MaxBodySize
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only limit for methods that typically have a body
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}
			next.ServeHTTP(w, r)
		})
	}
}
