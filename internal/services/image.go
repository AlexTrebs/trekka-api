package services

import (
	"context"
	"fmt"
	"log"

	"trekka-api/internal/models"
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

// Retrieves an image by generating a signed URL for direct GCS access.
// Returns the signed URL, content type, geolocation, and any error encountered.
// This approach offloads file serving to GCS, reducing serverless function load.
func (s *ImageService) GetImage(ctx context.Context, req models.ImageRequest) (string, string, string, error) {
	// Determine cache key - use Id if available, otherwise fileName
	cacheKey := req.Id
	if cacheKey == "" {
		cacheKey = req.FileName
	}

	// Check cache first for existing signed URL
	if entry, ok := s.cache.Get(cacheKey); ok {
		log.Printf("[Image] Cache hit: %s", cacheKey)
		return entry.SignedURL, entry.ContentType, entry.GeoLocation, nil
	}

	// Get metadata from Firestore - use Id lookup if available, otherwise fileName lookup
	var metadata *models.ImageMetadata
	var err error
	if req.Id != "" {
		metadata, err = s.firestore.GetImageMetadata(ctx, req.Id)
	} else if req.FileName != "" {
		fileType := ""
		if len(req.FileName) > 4 {
			fileType = req.FileName[len(req.FileName)-4:]
		}
		metadata, err = s.firestore.GetImageMetadataByFilename(ctx, req.FileName, fileType)
	} else {
		return "", "", "", fmt.Errorf("either Id or FileName must be provided")
	}
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get metadata: %w", err)
	}

	// Generate signed URL for direct GCS access
	signedURL, err := s.storage.GenerateSignedURL(ctx, metadata.StoragePath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	log.Printf("[Image] Generated signed URL for: %s", metadata.StoragePath)

	// Cache the signed URL using the same key used for lookup
	s.cache.Set(cacheKey, signedURL, metadata.ContentType, metadata.GeoLocation, metadata.FileName)

	return signedURL, metadata.ContentType, metadata.GeoLocation, nil
}

// ListImages retrieves a list of image metadata from Firestore.
func (s *ImageService) ListImages(ctx context.Context, limit int, page int) ([]*models.ImageMetadata, error) {
	return s.firestore.ListImageMetadata(ctx, limit, page)
}
