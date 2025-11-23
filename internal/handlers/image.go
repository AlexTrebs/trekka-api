package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"trekka-api/internal/models"
)

// Retrieves and serves images from Firebase Storage with caching.
// GET /image?fileName=<name>
// Query parameters:
//   - fileName (required): Name of the file for Content-Disposition header
func (h *Handler) HandleImage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	query := r.URL.Query()
	fileName := query.Get("fileName")

	if fileName == "" {
		http.Error(w, "Missing fileName", http.StatusBadRequest)
		return
	}

	req := models.ImageRequest{
		FileName: fileName,
	}

	data, contentType, geoLocation, err := h.imageService.GetImage(r.Context(), req)
	if err != nil {
		log.Printf("[Image] Failed to get image %s: %v", fileName, err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	log.Printf("[Image] Served %s (%s at %s) in %v", fileName, contentType, geoLocation, time.Since(start))

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`inline; filename="%s"`, url.QueryEscape(fileName)))
	w.Header().Set("X-Geo-Location", geoLocation)

	if _, err := w.Write(data); err != nil {
		log.Printf("[Image] Failed to write response: %v", err)
	}
}

// Retrieves and serves images from Firebase Storage with caching.
// GET /images/list
// Query parameters:
//   - limit (optional): number of items to return
//   - page  (optional): what page of items to return
func (h *Handler) HandleImagesList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit")) // defaults to 0 if not provided or invalid
	page, _ := strconv.Atoi(query.Get("page"))   // defaults to 0 if not provided or invalid

	images, err := h.imageService.ListImages(r.Context(), limit, page)
	if err != nil {
		log.Printf("[Images] Failed to list images: %v", err)
		http.Error(w, "Failed to retrieve images", http.StatusInternalServerError)
		return
	}

	log.Printf("[Images] Served %d images in %v", len(images), time.Since(start))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if err := json.NewEncoder(w).Encode(images); err != nil {
		log.Printf("[Images] Failed to encode response: %v", err)
	}
}
