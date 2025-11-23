package utils

import (
	"bytes"
	"fmt"
	"time"

	"trekka-api/internal/models"

	"github.com/rwcarlsen/goexif/exif"
)

// Extracts GPS coordinates and timestamp from image EXIF data
func ExtractData(imageData []byte) (models.Coordinates, string, error) {
	reader := bytes.NewReader(imageData)

	x, err := exif.Decode(reader)
	if err != nil {
		return models.Coordinates{}, "", fmt.Errorf("failed to decode EXIF: %w", err)
	}

	// Try to get GPS latitude
	lat, lon, err := x.LatLong()
	if err != nil {
		return models.Coordinates{}, "", fmt.Errorf("no GPS data found: %w", err)
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
		return models.Coordinates{}, "", fmt.Errorf("failed to parse timestamp: %w", lastErr)
	}

	// Format coordinates
	coords := models.Coordinates{
		Lat: fmt.Sprintf("%.6f", lat),
		Lng: fmt.Sprintf("%.6f", lon),
	}

	return coords, timestamp, nil
}

// Checks if an image already has GPS/location data
func HasEmptyFields(metadata *models.ImageMetadata, ignoreGeoLoc bool) bool {
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

	return false
}

// FormatTimestamp converts various timestamp formats to a human-readable format
// Supports both ISO 8601 (2006-01-02T15:04:05Z) and EXIF format (2006:01:02 15:04:05)
// Returns format: "Wednesday, 15 January 2025, 14:30"
func FormatTimestamp(timestamp string) (string, error) {
	var t time.Time
	var err error

	// Try multiple timestamp formats in order of likelihood
	formats := []string{
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00" (with timezone)
		"2006:01:02 15:04:05", // EXIF format
		"2006-01-02T15:04:05", // ISO 8601 without timezone
	}

	for _, format := range formats {
		t, err = time.Parse(format, timestamp)
		if err == nil {
			break
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp %q: %w", timestamp, err)
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
