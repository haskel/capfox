package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Enabled  bool
	User     string
	Password string
}

// Auth creates a Basic Auth middleware.
// Paths in excludePaths will be excluded from authentication.
// Paths ending with "*" are treated as prefixes (e.g., "/debug/*" matches "/debug/foo").
func Auth(config *AuthConfig, excludePaths ...string) Middleware {
	exactExcludes := make(map[string]bool)
	var prefixExcludes []string

	for _, path := range excludePaths {
		if strings.HasSuffix(path, "*") {
			prefixExcludes = append(prefixExcludes, strings.TrimSuffix(path, "*"))
		} else {
			exactExcludes[path] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if disabled
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for excluded exact paths
			if exactExcludes[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for excluded prefixes
			for _, prefix := range prefixExcludes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check Basic Auth
			user, pass, ok := r.BasicAuth()
			if !ok {
				unauthorized(w)
				return
			}

			// Constant time comparison to prevent timing attacks
			userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(config.User)) == 1
			passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(config.Password)) == 1

			if !userMatch || !passMatch {
				unauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="capfox"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
