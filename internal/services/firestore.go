package services

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"trekka-api/internal/models"
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
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	var metadata models.ImageMetadata
	if err := doc.DataTo(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata.ID = doc.Ref.ID
	return &metadata, nil
}

// Retrieves image metadata by storage path.
func (fs *FirestoreService) GetImageMetadataByFileName(ctx context.Context, fileName string) (*models.ImageMetadata, error) {
	iter := fs.client.Collection(fs.collection).
		Where("fileName", "==", fileName).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("no metadata found for storage path: %s", fileName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query metadata: %w", err)
	}

	var metadata models.ImageMetadata
	if err := doc.DataTo(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata.ID = doc.Ref.ID
	return &metadata, nil
}

// Retrieves all image metadata from the collection.
func (fs *FirestoreService) ListImageMetadata(ctx context.Context, limit int, page int) ([]*models.ImageMetadata, error) {
	query := fs.client.Collection(fs.collection).OrderBy("createdAt", firestore.Desc)

	if limit > 0 {
		query = query.Limit(limit)
		if page > 0 {
			query = query.Offset((page - 1) * limit)
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
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		metadata.ID = doc.Ref.ID
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
