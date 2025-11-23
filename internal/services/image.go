package services

import (
	"context"
	"fmt"
	"log"

	"trekka-api/internal/models"
	"trekka-api/internal/utils"
)

type ImageService struct {
	storage   *StorageService
	cache     *CacheService
	firestore *FirestoreService
}

func NewImageService(storage *StorageService, cache *CacheService, firestore *FirestoreService) *ImageService {
	return &ImageService{
		storage:   storage,
		cache:     cache,
		firestore: firestore,
	}
}

// Retrieves an image from cache or storage, converting HEIC/HEIF to JPEG if needed.
// Returns the image data, content type, geolocation, and any error encountered.
func (s *ImageService) GetImage(ctx context.Context, req models.ImageRequest) ([]byte, string, string, error) {
	// Check cache first
	if entry, ok := s.cache.Get(req.FileName); ok {
		log.Printf("[Image] Cache hit: %s", req.FileName)
		return entry.Data, entry.ContentType, entry.GeoLocation, nil
	}

	// Get metadata from Firestore
	metadata, err := s.firestore.GetImageMetadataByFileName(ctx, req.FileName)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get metadata: %w", err)
	}

	// Fetch from storage
	data, err := s.storage.FetchFile(ctx, req.FileName)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to fetch file: %w", err)
	}

	log.Printf("[Image] Fetched %d bytes from storage", len(data))

	// Process image
	output := data
	finalType := metadata.ContentType

	// Convert HEIC/HEIF to JPEG
	if utils.IsHeifLike(metadata.ContentType) {
		log.Printf("[Image] Converting HEIF → JPEG")
		jpegData, err := utils.ConvertHeicToJpeg(output)
		if err != nil {
			log.Printf("[Image] HEIC conversion failed for %s: %v", req.FileName, err)
		} else {
			output = jpegData
			finalType = "image/jpeg"
		}
	}

	// Cache the result
	s.cache.Set(req.FileName, output, finalType, metadata.GeoLocation, req.FileName)

	return output, finalType, metadata.GeoLocation, nil
}

// ListImages retrieves a list of image metadata from Firestore.
func (s *ImageService) ListImages(ctx context.Context, limit int, page int) ([]*models.ImageMetadata, error) {
	return s.firestore.ListImageMetadata(ctx, limit, page)
}
