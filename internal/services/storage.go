package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

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
func (s *StorageService) FetchFile(ctx context.Context, storagePath string) ([]byte, error) {
	if storagePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(storagePath)

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

// Creates a temporary signed URL for direct access to a GCS object.
// The URL expires after 15 minutes, allowing clients to fetch files directly from GCS
// without proxying through the application server.
func (s *StorageService) GenerateSignedURL(ctx context.Context, storagePath string) (string, error) {
	if storagePath == "" {
		return "", fmt.Errorf("storage path cannot be empty")
	}

	opts := &storage.SignedURLOptions{
		Expires: time.Now().Add(15 * time.Minute),
		Method:  "GET",
		Scheme:  storage.SigningSchemeV4,
	}

	url, err := s.client.Bucket(s.bucketName).SignedURL(storagePath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// Uploads a file to Google Cloud Storage.
// Returns an error if the upload fails.
func (s *StorageService) UploadFile(ctx context.Context, filePath string, data []byte, contentType string) (err error) {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(filePath)

	writer := obj.NewWriter(ctx)
	defer func() {
		if closeErr := writer.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close writer: %w", closeErr)
		}
	}()

	writer.ContentType = contentType
	writer.Metadata = map[string]string{
		"uploaded-by": "trekka-drive-sync",
	}

	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	return nil
}
