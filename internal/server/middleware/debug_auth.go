package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// DebugAuthConfig holds debug endpoint authentication configuration.
type DebugAuthConfig struct {
	// Token for Bearer authentication on debug endpoints.
	Token string
	// FallbackAuthConfig is used when Token is empty.
	FallbackAuthConfig *AuthConfig
}

// DebugAuth creates a middleware that protects debug endpoints.
// If token is set, requires Bearer <token> header.
// If token is empty but fallback auth is enabled, uses Basic Auth.
// If both are empty/disabled, blocks all requests.
func DebugAuth(config *DebugAuthConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If debug token is set, use Bearer auth
			if config.Token != "" {
				if checkBearerToken(r, config.Token) {
					next.ServeHTTP(w, r)
					return
				}
				forbiddenDebug(w)
				return
			}

			// Fall back to Basic Auth if enabled
			if config.FallbackAuthConfig != nil {
				// Use thread-safe getter to avoid data race during config hot reload
				enabled, configUser, configPass := config.FallbackAuthConfig.get()
				if enabled {
					user, pass, ok := r.BasicAuth()
					if !ok {
						unauthorizedDebug(w)
						return
					}

					userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(configUser)) == 1
					passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(configPass)) == 1

					if !userMatch || !passMatch {
						unauthorizedDebug(w)
						return
					}

					next.ServeHTTP(w, r)
					return
				}
			}

			// No authentication configured - block access
			forbiddenDebug(w)
		})
	}
}

// checkBearerToken validates the Authorization: Bearer <token> header.
func checkBearerToken(r *http.Request, expectedToken string) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Check for "Bearer <token>" format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1
}

func unauthorizedDebug(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="capfox-debug"`)
	http.Error(w, "Unauthorized - Debug endpoint requires authentication", http.StatusUnauthorized)
}

func forbiddenDebug(w http.ResponseWriter) {
	http.Error(w, "Forbidden - Debug authentication required", http.StatusForbidden)
}
