package services

import (
	"context"
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
func (s *StorageService) FetchFile(ctx context.Context, filePath string) ([]byte, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(filePath)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
