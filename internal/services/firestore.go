package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"trekka-api/internal/errors"
	"trekka-api/internal/models"
	"trekka-api/internal/utils"
)

type FirestoreService struct {
	client     *firestore.Client
	collection string
}

func NewFirestoreService(client *firestore.Client, collection string) *FirestoreService {
	return &FirestoreService{
		client:     client,
		collection: collection,
	}
}

// Retrieves image metadata by document ID.
func (fs *FirestoreService) GetImageMetadata(ctx context.Context, id string) (*models.ImageMetadata, error) {
	doc, err := fs.client.Collection(fs.collection).Doc(id).Get(ctx)
	if err != nil {
		// Check if document not found
		if status.Code(err) == codes.NotFound {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	var metadata models.ImageMetadata
	if err := doc.DataTo(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// Retrieves all image metadata from the collection with pagination.
func (fs *FirestoreService) ListImageMetadata(ctx context.Context, limit int, page int) ([]*models.ImageMetadata, error) {
	// Validate pagination parameters
	if limit < 0 {
		return nil, fmt.Errorf("limit cannot be negative")
	}
	if page < 0 {
		return nil, fmt.Errorf("page cannot be negative")
	}

	// Order by takenAt if available, fallback to createdAt
	query := fs.client.Collection(fs.collection).OrderBy("takenAt", firestore.Desc)

	if limit > 0 {
		// Cap maximum limit to prevent excessive memory usage
		if limit > 1000 {
			limit = 1000
		}
		query = query.Limit(limit)
		if page > 0 {
			query = query.Offset(page * limit)
		}
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var results []*models.ImageMetadata
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate documents: %w", err)
		}

		var metadata models.ImageMetadata
		if err := doc.DataTo(&metadata); err != nil {
			// Log but don't fail on individual document parse errors
			continue
		}

		results = append(results, &metadata)
	}

	return results, nil
}

// Retrieves all image metadata ordered by createdAt.
// Used for migrations where takenAt field might not exist yet.
func (fs *FirestoreService) ListAllImageMetadata(ctx context.Context, limit int, page int) ([]*models.ImageMetadata, error) {
	// Validate pagination parameters
	if limit < 0 {
		return nil, fmt.Errorf("limit cannot be negative")
	}
	if page < 0 {
		return nil, fmt.Errorf("page cannot be negative")
	}

	// Order by createdAt instead of takenAt for migration compatibility
	query := fs.client.Collection(fs.collection).OrderBy("createdAt", firestore.Desc)

	if limit > 0 {
		// Cap maximum limit to prevent excessive memory usage
		if limit > 1000 {
			limit = 1000
		}
		query = query.Limit(limit)
		if page > 0 {
			query = query.Offset(page * limit)
		}
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var results []*models.ImageMetadata
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate documents: %w", err)
		}

		var metadata models.ImageMetadata
		if err := doc.DataTo(&metadata); err != nil {
			// Log but don't fail on individual document parse errors
			continue
		}

		results = append(results, &metadata)
	}

	return results, nil
}

// Creates a new image metadata document.
func (fs *FirestoreService) CreateImageMetadata(ctx context.Context, metadata *models.ImageMetadata) (string, error) {
	docRef, _, err := fs.client.Collection(fs.collection).Add(ctx, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata: %w", err)
	}

	return docRef.ID, nil
}

// Updates an existing image metadata document.
func (fs *FirestoreService) UpdateImageMetadata(ctx context.Context, id string, metadata *models.ImageMetadata) error {
	_, err := fs.client.Collection(fs.collection).Doc(id).Set(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// Deletes an image metadata document by ID.
func (fs *FirestoreService) DeleteImageMetadata(ctx context.Context, id string) error {
	_, err := fs.client.Collection(fs.collection).Doc(id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

// Gets image metadata by filename.
func (fs *FirestoreService) GetImageMetadataByFilename(ctx context.Context, filename string, fileType string) (*models.ImageMetadata, error) {
	finalFilename := filename
	if utils.IsHeifLike(fileType) {
		ext := filepath.Ext(filename)
		finalFilename = strings.TrimSuffix(filename, ext) + ".jpg"
	}
	iter := fs.client.Collection(fs.collection).Where("fileName", "==", finalFilename).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		// Check if no documents found
		if err == iterator.Done {
			return nil, errors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}

	var metadata models.ImageMetadata
	if err := doc.DataTo(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}
