package router

import (
	"net/http"

	"trekka-api/internal/handlers"
)

// Setup configures and returns the HTTP router with all application routes.
func Setup(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", h.HandleHealth)

	// Image endpoints
	mux.HandleFunc("/image", h.HandleImage)
	mux.HandleFunc("/images/list", h.HandleAll)

	return mux
}
