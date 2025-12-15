package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"trekka-api/internal/models"
	"trekka-api/internal/utils"
)

// Extracts metadata from file bytes (EXIF for images, MP4 for videos).
// Returns a metadata struct with coordinates, timestamp, resolution, and location (if geocoding succeeds).
func ExtractMetadataFromBytes(ctx context.Context, fileName, contentType string, fileData []byte) (*models.ImageMetadata, error) {
	var coords models.Coordinates
	var timestamp string
	var resolution []float64
	var extractErr error

	isVideo := strings.HasPrefix(contentType, "video/")
	if isVideo {
		coords, timestamp, resolution, extractErr = utils.ExtractMP4Data(fileData)
	} else {
		coords, timestamp, resolution, extractErr = utils.ExtractData(fileData)
	}

	if extractErr != nil {
		log.Printf("Warning: failed to extract metadata from %s: %v", fileName, extractErr)
	}

	// Build metadata struct with extracted data
	metadata := &models.ImageMetadata{
		FileName:    fileName,
		ContentType: contentType,
		StoragePath: fileName,
	}

	// Populate extracted data
	if coords.Lat != "" && coords.Lng != "" {
		metadata.Coordinates = coords

		// Geocode coordinates to location name
		geocoder := NewGeocodingService()
		location, err := geocoder.ReverseGeocode(ctx, coords)
		if err == nil && location != "" {
			metadata.GeoLocation = location
		}
	}

	if timestamp != "" {
		metadata.TakenAt = utils.ParseTimeString(timestamp)
		if formattedDate, err := utils.FormatTimeDisplay(timestamp); err == nil {
			metadata.FormattedDate = formattedDate
		}
	}

	if len(resolution) == 2 {
		metadata.Resolution = resolution
	}

	return metadata, nil
}

// Extracts metadata from file bytes and saves to Firestore.
// For new files (existing == nil), it creates a new record.
// For existing files, it updates only the extracted fields.
func ExtractAndPersistMetadata(
	ctx context.Context,
	firestoreService *FirestoreService,
	fileName, contentType string,
	fileData []byte,
	existing *models.ImageMetadata,
	geoGeocodingService *GeocodingService,
) (*models.ImageMetadata, error) {
	// Extract metadata from file
	extracted, err := ExtractMetadataFromBytes(ctx, fileName, contentType, fileData)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Merge with existing record or create new
	var metadata *models.ImageMetadata
	if existing != nil {
		metadata = existing
		// Update with extracted data
		if extracted.Coordinates.Lat != "" && extracted.Coordinates.Lng != "" {
			metadata.Coordinates = extracted.Coordinates
			metadata.GeoLocation = extracted.GeoLocation
		}
		if !extracted.TakenAt.IsZero() {
			metadata.TakenAt = extracted.TakenAt
			metadata.FormattedDate = extracted.FormattedDate
		}
		if len(extracted.Resolution) == 2 {
			metadata.Resolution = extracted.Resolution
		}
		metadata.UpdatedAt = now
	} else {
		metadata = extracted
		metadata.CreatedAt = now
		metadata.UpdatedAt = now
	}

	// If TakenAt still not set, fall back to CreatedAt
	if metadata.TakenAt.IsZero() && !metadata.CreatedAt.IsZero() {
		metadata.TakenAt = metadata.CreatedAt
	}

	// Persist to Firestore (create or update)
	if existing == nil {
		firestoreID, err := firestoreService.CreateImageMetadata(ctx, metadata)
		if err != nil {
			return nil, fmt.Errorf("create metadata failed: %w", err)
		}
		metadata.Id = firestoreID
	} else {
		if err := firestoreService.UpdateImageMetadata(ctx, metadata.Id, metadata); err != nil {
			return nil, fmt.Errorf("update metadata failed: %w", err)
		}
	}

	return metadata, nil
}
