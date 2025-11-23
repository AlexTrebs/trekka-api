package middleware

import (
	"net/http"
	"slices"
)

// Adds Cross-Origin Resource Sharing headers to HTTP responses.
// It allows all origins (*) and common HTTP methods.
// Handles preflight OPTIONS requests automatically.
func CORS(next http.Handler, allowedOrigins []string) http.Handler {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && slices.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
