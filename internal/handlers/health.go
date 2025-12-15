package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// HandleHealth responds to health check requests.
//
//	@Summary		Health check
//	@Description	Check if the API is running
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string	"status: ok"
//	@Router			/health [get]
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	}); err != nil {
		log.Printf("[Health] Failed to encode response: %v", err)
	}
}
