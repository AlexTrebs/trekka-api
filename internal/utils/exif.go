package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"time"

	"trekka-api/internal/models"

	"github.com/rwcarlsen/goexif/exif"
)

// Extracts GPS coordinates, timestamp, and resolution from image EXIF data
func ExtractData(imageData []byte) (models.Coordinates, string, []float64, error) {
	reader := bytes.NewReader(imageData)

	x, err := exif.Decode(reader)
	if err != nil {
		return models.Coordinates{}, "", nil, fmt.Errorf("failed to decode EXIF: %w", err)
	}

	// Try to get GPS latitude
	lat, lon, err := x.LatLong()
	if err != nil {
		return models.Coordinates{}, "", nil, fmt.Errorf("no GPS data found: %w", err)
	}

	// Get dateTime timestamp
	var timestamp string

	dt, err := x.DateTime()
	if err == nil {
		timestamp = dt.Format("2006-01-02T15:04:05Z07:00")
	}

	var lastErr error = err
	if timestamp == "" {
		// Try DateTimeOriginal
		dateTag, getErr := x.Get(exif.DateTimeOriginal)
		if getErr != nil {
			lastErr = getErr
		} else {
			dateStr, strErr := dateTag.StringVal()
			if strErr != nil {
				lastErr = strErr
			} else {
				// EXIF DateTimeOriginal is typically "2006:01:02 15:04:05"
				// Parse it and format consistently
				t, parseErr := time.Parse("2006:01:02 15:04:05", dateStr)
				if parseErr != nil {
					lastErr = parseErr
				} else {
					timestamp = t.Format("2006-01-02T15:04:05Z07:00")
				}
			}
		}
	}

	if timestamp == "" {
		return models.Coordinates{}, "", nil, fmt.Errorf("failed to parse timestamp: %w", lastErr)
	}

	// Format coordinates
	coords := models.Coordinates{
		Lat: fmt.Sprintf("%.6f", lat),
		Lng: fmt.Sprintf("%.6f", lon),
	}

	// Extract resolution from image data
	var resolution []float64
	reader.Seek(0, 0) // Reset reader to start
	config, _, err := image.DecodeConfig(reader)
	if err == nil && config.Width > 0 && config.Height > 0 {
		resolution = []float64{float64(config.Width), float64(config.Height)}
	}

	return coords, timestamp, resolution, nil
}

// Checks if an image already has GPS/location data
func hasEmptyFields(metadata *models.ImageMetadata, ignoreGeoLoc bool) bool {
	// Check if GeoLocation string is present
	if metadata.GeoLocation == "" && !ignoreGeoLoc {
		return true
	}
	// Check if coordinates are present (return true if either is missing)
	if metadata.Coordinates.Lat == "" || metadata.Coordinates.Lng == "" {
		return true
	}

	if metadata.FormattedDate == "" {
		return true
	}

	if metadata.TakenAt.IsZero() {
		return true
	}

	return false
}

func HasEmptyFields(metadata *models.ImageMetadata) bool {
	return hasEmptyFields(metadata, false)
}

func HasEmptyFieldsSkipGeolocation(metadata *models.ImageMetadata) bool {
	return hasEmptyFields(metadata, true)
}

// ParseTimeString attempts to parse a timestamp string into a time.Time value.
// It tries multiple common formats (RFC3339, EXIF, ISO 8601).
// Returns zero time if parsing fails.
func ParseTimeString(timestamp string) time.Time {
	if timestamp == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00" (with timezone)
		"2006:01:02 15:04:05", // EXIF format
		"2006-01-02T15:04:05", // ISO 8601 without timezone
		"2006-01-02 15:04:05", // ISO 8601 date with space separator (from MP4)
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return t
		}
	}

	return time.Time{}
}

func FormatTimeDisplay(timestamp string) (string, error) {
	t := ParseTimeString(timestamp)

	if t.IsZero() {
		return "", fmt.Errorf("failed to parse timestamp %q", timestamp)
	}

	// Format as: "Wednesday, 15 January 2025, 14:30"
	weekday := t.Weekday().String()
	day := t.Day()
	month := t.Month().String()
	year := t.Year()
	hour := fmt.Sprintf("%02d", t.Hour())
	minute := fmt.Sprintf("%02d", t.Minute())

	return fmt.Sprintf("%s, %d %s %d, %s:%s", weekday, day, month, year, hour, minute), nil
}
