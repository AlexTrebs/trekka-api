package middleware

import (
	"crypto/subtle"
	"net/http"
)

// APIKeyAuth creates middleware that validates API key authentication.
// It checks the X-API-Key header against a list of valid API keys using
// constant-time comparison to prevent timing attacks.
// Requests to /health are exempted from authentication.
func APIKeyAuth(apiKeys []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt health check endpoint from authentication
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Get API key from header
			key := r.Header.Get("X-API-Key")
			if key == "" {
				http.Error(w, "Unauthorized: missing API key", http.StatusUnauthorized)
				return
			}

			// Validate API key using constant-time comparison
			valid := false
			for _, validKey := range apiKeys {
				if subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1 {
					valid = true
					break
				}
			}

			if !valid {
				http.Error(w, "Unauthorized: invalid API key", http.StatusUnauthorized)
				return
			}

			// API key is valid, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}
