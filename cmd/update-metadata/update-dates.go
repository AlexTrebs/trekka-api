package main

import (
	"context"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"trekka-api/internal/config"
	"trekka-api/internal/services"
)

func main() {
	logger := log.New(os.Stdout, "[MetadataUpdate] ", log.LstdFlags)

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		logger.Fatalf("load config: %v", cfgErr)
	}

	ctx := context.Background()

	// Configure GCP credentials
	var opts []option.ClientOption
	if cfg.FirebaseCredentialsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON)))
	} else {
		opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCredentialsPath))
	}

	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		logger.Fatalf("storage client: %v", err)
	}
	defer storageClient.Close()

	firestoreClient, err := firestore.NewClient(ctx, cfg.FirebaseProjectID, opts...)
	if err != nil {
		logger.Fatalf("firestore client: %v", err)
	}
	defer firestoreClient.Close()

	// Services
	storageService := services.NewStorageService(storageClient, cfg.FirebaseBucketName)
	firestoreService := services.NewFirestoreService(firestoreClient, cfg.FirestoreCollection)

	allImages, err := firestoreService.ListImageMetadata(ctx, 0, 0)
	if err != nil {
		logger.Fatalf("list images: %v", err)
	}

	imagesLen := len(allImages)
	for i, image := range allImages {
		file, _ := storageService.FetchFile(ctx, image.StoragePath)
		filemeta, _ := services.ExtractMetadataFromBytes(ctx, image.FileName, image.ContentType, file)
		storagemeta, _ := firestoreService.GetImageMetadataByFilename(ctx, image.FileName, image.ContentType)
		updated := false
		if filemeta.TakenAt != storagemeta.TakenAt {
			storagemeta.TakenAt = filemeta.TakenAt
			updated = true
		}

		if filemeta.FormattedDate != storagemeta.FormattedDate {
			storagemeta.FormattedDate = filemeta.FormattedDate
			updated = true
		}

		if updated {
			storagemeta.UpdatedAt = time.Now()
			firestoreService.UpdateImageMetadata(ctx, storagemeta.Id, storagemeta)
			logger.Printf("Updated %d/%d: %s with type %s", i, imagesLen, storagemeta.FileName, storagemeta.ContentType)
		}
	}
}
