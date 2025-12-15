package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "requestID"

// RequestID creates middleware that generates a unique request ID for each request.
// The request ID is added to the request context and included in the response headers
// as X-Request-ID for easier debugging and request tracing.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate unique request ID
		requestID := uuid.New().String()

		// Add request ID to context for use in handlers/services
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		// Continue with the request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
