package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"trekka-api/internal/models"

	"golang.org/x/time/rate"
)

// Performs reverse geocoding using the OpenStreetMap Nominatim
// API with caching and rate limiting.
type GeocodingService struct {
	cache       map[string]string
	cacheMutex  sync.RWMutex
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// Models the subset of Nominatim’s response that we care about
// (city/town/village + country).
type NominatimResponse struct {
	Address struct {
		City    string `json:"city"`
		Town    string `json:"town"`
		Village string `json:"village"`
		Country string `json:"country"`
	} `json:"address"`
}

// Returns a fully configured geocoder.
// It includes:
//   - in-memory cache
//   - shared HTTP client
//   - Nominatim-compliant rate limiting (1 request/sec)
func NewGeocodingService() *GeocodingService {
	return &GeocodingService{
		cache:      make(map[string]string),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		rateLimiter: rate.NewLimiter(
			rate.Limit(1), // 1 request/sec
			1,             // burst size
		),
	}
}

// Performs a coordinate→location lookup.
// The function:
//  1. normalizes coordinates
//  2. checks the in-memory cache
//  3. applies rate limiting (required by Nominatim)
//  4. calls the Nominatim API
//  5. extracts city/town/village + country
//  6. caches & returns the formatted result
func (g *GeocodingService) ReverseGeocode(ctx context.Context, coordinates models.Coordinates) (string, error) {
	log.Printf("Reverse GeoCoding...")

	lat, lng, key, err := g.normalizeCoordinates(coordinates)
	if err != nil {
		return "", err
	}

	// First check: read lock
	g.cacheMutex.RLock()
	if cached := g.cache[key]; cached != "" {
		g.cacheMutex.RUnlock()
		return cached, nil
	}
	g.cacheMutex.RUnlock()

	// Rate limit before making API call
	if err := g.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}

	// Fetch from API
	result, err := g.fetchLocation(ctx, lat, lng)
	if err != nil {
		return "", err
	}

	// Double-check cache before writing (another goroutine might have set it)
	g.cacheMutex.Lock()
	if cached := g.cache[key]; cached != "" {
		g.cacheMutex.Unlock()
		return cached, nil
	}
	g.cache[key] = result
	g.cacheMutex.Unlock()

	return result, nil
}

// Parses and normalizes latitude/longitude values,
// and returns a rounded cache key.
func (g *GeocodingService) normalizeCoordinates(c models.Coordinates) (lat, lng float64, key string, err error) {
	lat, err = strconv.ParseFloat(strings.TrimSpace(c.Lat), 64)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid latitude: %w", err)
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(c.Lng), 64)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid longitude: %w", err)
	}

	// Key rounded to avoid cache fragmentation
	key = fmt.Sprintf("%.4f,%.4f", lat, lng)
	return lat, lng, key, nil
}

// Performs the actual HTTP request and parses the response.
func (g *GeocodingService) fetchLocation(ctx context.Context, lat, lng float64) (string, error) {
	url := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f&zoom=18&addressdetails=1",
		lat, lng,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Trekka")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Referer", "https://trekka.co.uk")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data NominatimResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return g.extractLocation(data), nil
}

// Chooses the most specific available location from the response.
func (g *GeocodingService) extractLocation(n NominatimResponse) string {
	city := firstNonEmpty(
		n.Address.City,
		n.Address.Town,
		n.Address.Village,
	)
	country := n.Address.Country

	switch {
	case city != "" && country != "":
		return city + ", " + country
	case city != "":
		return city
	default:
		return country
	}
}

// Returns the first non-empty string in the list.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
