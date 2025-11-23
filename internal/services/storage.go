package services

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

type StorageService struct {
	client     *storage.Client
	bucketName string
}

func NewStorageService(client *storage.Client, bucketName string) *StorageService {
	return &StorageService{
		client:     client,
		bucketName: bucketName,
	}
}

// Retrieves a file from Google Cloud Storage by its path.
// Returns the file contents as bytes or an error if the file cannot be retrieved.
// Implements a maximum file size limit to prevent memory exhaustion.
func (s *StorageService) FetchFile(ctx context.Context, filePath string) ([]byte, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(filePath)

	// Get object attributes to check size
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get file attributes: %w", err)
	}

	// Limit file size to 50MB to prevent memory exhaustion
	const maxFileSize = 50 * 1024 * 1024 // 50MB
	if attrs.Size > maxFileSize {
		return nil, fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", attrs.Size, maxFileSize)
	}

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create file reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	return data, nil
}
