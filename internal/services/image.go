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

// Retrieves an image from cache or storage, converting HEIC/HEIF to JPEG if needed.
// Returns the image data, content type, geolocation, and any error encountered.
func (s *ImageService) GetImage(ctx context.Context, req models.ImageRequest) ([]byte, string, string, error) {
	// Check cache first
	if entry, ok := s.cache.Get(req.Id); ok {
		log.Printf("[Image] Cache hit: %s", req.Id)
		return entry.Data, entry.ContentType, entry.GeoLocation, nil
	}

	// Get metadata from Firestore
	metadata, err := s.firestore.GetImageMetadata(ctx, req.Id)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get metadata: %w", err)
	}

	// Fetch from storage
	data, err := s.storage.FetchFile(ctx, metadata.StoragePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to fetch file: %w", err)
	}

	log.Printf("[Image] Fetched %d bytes from storage", len(data))

	// Cache the result using the same key used for lookup (req.Id)
	s.cache.Set(req.Id, data, metadata.ContentType, metadata.GeoLocation, metadata.FileName)

	return data, metadata.ContentType, metadata.GeoLocation, nil
}

// ListImages retrieves a list of image metadata from Firestore.
func (s *ImageService) ListImages(ctx context.Context, limit int, page int) ([]*models.ImageMetadata, error) {
	return s.firestore.ListImageMetadata(ctx, limit, page)
}
