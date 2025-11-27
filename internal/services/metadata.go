package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"trekka-api/internal/models"
	"trekka-api/internal/utils"

	"google.golang.org/api/drive/v3"
)

// Performs the full metadata extraction pipeline:
//  1. try Firebase Storage EXIF
//  2. fallback to Drive metadata
//  3. fallback to Drive EXIF (download)
//  4. reverse geocode
//  5. timestamp formatting
type MetadataResolver struct {
	storage  *StorageService
	drive    *DriveClient
	geocoder *GeocodingService
	folderID string
}

// Creates a resolver used in both DriveSync + Updater.
func NewMetadataResolver(
	storage *StorageService,
	drive *DriveClient,
	geocoder *GeocodingService,
	folderID string,
) *MetadataResolver {
	return &MetadataResolver{
		storage:  storage,
		drive:    drive,
		geocoder: geocoder,
		folderID: folderID,
	}
}

// Populates Coordinates, GeoLocation, and Timestamp for an image.
func (r *MetadataResolver) ResolveMetadata(ctx context.Context, metadata *models.ImageMetadata) (*models.ImageMetadata, error) {
	return r.resolveMetadata(ctx, metadata, false)
}

// Forces download from Drive instead of trying Storage first.
func (r *MetadataResolver) ResolveMetadataWithBackfill(ctx context.Context, metadata *models.ImageMetadata) (*models.ImageMetadata, error) {
	return r.resolveMetadata(ctx, metadata, true)
}

func (r *MetadataResolver) resolveMetadata(ctx context.Context, metadata *models.ImageMetadata, skipStorage bool) (*models.ImageMetadata, error) {
	var gps *models.Coordinates
	var timestamp string

	if !utils.HasEmptyFields(metadata, true) {
		timestamp = metadata.FormattedDate
		gps = &metadata.Coordinates
	}

	// Try Firebase Storage EXIF (unless backfill mode)
	if !skipStorage && gps == nil {
		log.Printf("Fetching from storage: %s", metadata.StoragePath)

		data, err := r.storage.FetchFile(ctx, metadata.StoragePath)
		if err == nil {
			coords, extractedTimestamp, extractionErr := utils.ExtractData(data)

			log.Printf("coords: %s, timestamp: %s", coords, extractedTimestamp)

			if extractionErr == nil {
				timestamp = extractedTimestamp
				gps = &coords
			}
		}
	}

	// Try Drive metadata
	if (gps == nil || timestamp == "") && r.drive != nil && r.folderID != "" {
		log.Printf("Fetching metadata from Drive")

		file, err := r.drive.Find(ctx, r.folderID, metadata.FileName)
		if err == nil {
			extractedGPS, extractedTimestamp, extractErr := r.extractDataFromDriveMetadata(file)
			if extractErr == nil {
				if gps == nil {
					gps = extractedGPS
				}
				if timestamp == "" {
					timestamp = extractedTimestamp
				}
			} else {
				log.Printf("Warning: failed to extract Drive metadata for %s: %v", metadata.FileName, extractErr)
			}
		} else {
			log.Printf("Warning: failed to find file in Drive: %v", err)
		}
	}

	// Populate GPS data if available
	if gps != nil {
		metadata.Coordinates = *gps

		// Always reverse geocode when we have coordinates
		location, err := r.geocoder.ReverseGeocode(ctx, *gps)
		if err == nil && location != "" {
			metadata.GeoLocation = location
		}
	}

	// Populate timestamp if available
	if timestamp != "" {
		formattedTS, err := utils.FormatTimestamp(timestamp)
		if err != nil {
			log.Printf("Warning: failed to format timestamp %q: %v", timestamp, err)
		} else {
			metadata.FormattedDate = formattedTS
		}
	}

	metadata.UpdatedAt = time.Now()

	return metadata, nil
}

// Extracts GPS coordinates from Drive's imageMediaMetadata.
// This is the most reliable method for all image formats including HEIF.
func (r *MetadataResolver) extractDataFromDriveMetadata(driveFile *drive.File) (*models.Coordinates, string, error) {
	if driveFile.ImageMediaMetadata == nil {
		return nil, "", fmt.Errorf("no image metadata in Drive file")
	}

	if driveFile.ImageMediaMetadata.Location == nil {
		return nil, "", fmt.Errorf("no location data in Drive metadata")
	}

	lat := driveFile.ImageMediaMetadata.Location.Latitude
	lon := driveFile.ImageMediaMetadata.Location.Longitude

	if lat == 0 && lon == 0 {
		return nil, "", fmt.Errorf("GPS coordinates are zero")
	}

	timestamp := driveFile.ImageMediaMetadata.Time

	return &models.Coordinates{
			Lat: fmt.Sprintf("%.6f", lat),
			Lng: fmt.Sprintf("%.6f", lon),
		},
		timestamp,
		nil
}
