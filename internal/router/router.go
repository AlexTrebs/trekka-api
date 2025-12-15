package router

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
	"trekka-api/internal/handlers"
)

// Setup configures and returns the HTTP router with all application routes.
func Setup(h *handlers.Handler) http.Handler {
	mux := http.NewServeMux()

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Health check
	mux.HandleFunc("/health", h.HandleHealth)

	// Image endpoints
	mux.HandleFunc("/image", h.HandleImage)
	mux.HandleFunc("/images/list", h.HandleImagesList)

	return mux
}
