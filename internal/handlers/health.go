package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// Responds to health check requests with a simple status message.
// GET /health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	}); err != nil {
		log.Printf("[Health] Failed to encode response: %v", err)
	}
}
