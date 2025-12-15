package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	apperrors "trekka-api/internal/errors"
	"trekka-api/internal/models"
)

// HandleImage retrieves and serves images from Firebase Storage with caching.
//
//	@Summary		Get an image
//	@Description	Retrieve an image from Firebase Storage by filename
//	@Tags			images
//	@Accept			json
//	@Produce		json
//	@Param			fileName	query		string	true	"Image filename"
//	@Success		302			{string}	string	"Redirect to signed URL"
//	@Failure		400			{string}	string	"Bad Request"
//	@Failure		404			{string}	string	"Not Found"
//	@Failure		500			{string}	string	"Internal Server Error"
//	@Security		ApiKeyAuth
//	@Router			/image [get]
func (h *Handler) HandleImage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	fileName := strings.TrimSpace(query.Get("fileName"))

	// Validate fileName parameter
	if fileName == "" {
		http.Error(w, "Missing fileName parameter", http.StatusBadRequest)
		return
	}

	// Security: Prevent path traversal attacks
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		log.Printf("[Image] Security: Rejected suspicious fileName: %s", fileName)
		http.Error(w, "Invalid fileName", http.StatusBadRequest)
		return
	}

	// Validate fileName length
	if len(fileName) > 255 {
		http.Error(w, "fileName too long", http.StatusBadRequest)
		return
	}

	req := models.ImageRequest{
		FileName: fileName,
	}

	signedURL, contentType, geoLocation, err := h.imageService.GetImage(r.Context(), req)
	if err != nil {
		log.Printf("[Image] Failed to get image %s: %v", fileName, err)
		// Check if it's a "not found" error vs infrastructure error
		if errors.Is(err, apperrors.ErrNotFound) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("[Image] Redirecting to signed URL for %s (%s at %s) in %v", fileName, contentType, geoLocation, time.Since(start))

	// Set metadata headers before redirect
	w.Header().Set("Cache-Control", "public, max-age=900, s-maxage=900") // 15 min
	w.Header().Set("CDN-Cache-Control", "public, max-age=86400")         // Vercel edge: 24hr
	w.Header().Set("X-Geo-Location", geoLocation)
	w.Header().Set("X-Content-Type", contentType)

	// Redirect to GCS signed URL for direct download
	http.Redirect(w, r, signedURL, http.StatusFound)
}

// HandleImagesList retrieves a paginated list of images with metadata.
//
//	@Summary		List images
//	@Description	Get a paginated list of images with metadata from Firestore
//	@Tags			images
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int								false	"Number of items to return (max 1000, default 1000)"	default(1000)
//	@Param			page	query		int								false	"Page number (0-indexed, default 0)"				default(0)
//	@Success		200		{array}		models.ImageMetadata			"List of images"
//	@Failure		400		{string}	string							"Bad Request"
//	@Failure		500		{string}	string							"Internal Server Error"
//	@Security		ApiKeyAuth
//	@Router			/images/list [get]
func (h *Handler) HandleImagesList(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	// Parse and validate limit parameter
	limit := 1000
	if limitStr := query.Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 0 {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	// Parse and validate page parameter
	page := 0 // default
	if pageStr := query.Get("page"); pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err != nil || parsedPage < 0 {
			http.Error(w, "Invalid page parameter", http.StatusBadRequest)
			return
		}
		page = parsedPage
	}

	images, err := h.imageService.ListImages(r.Context(), limit, page)
	if err != nil {
		log.Printf("[Images] Failed to list images: %v", err)
		http.Error(w, "Failed to retrieve images", http.StatusInternalServerError)
		return
	}

	log.Printf("[Images] Served %d images (limit=%d, page=%d) in %v", len(images), limit, page, time.Since(start))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300") // 1 min client, 5 min edge

	if err := json.NewEncoder(w).Encode(images); err != nil {
		log.Printf("[Images] Failed to encode response: %v", err)
	}
}
