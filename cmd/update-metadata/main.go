package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"trekka-api/internal/config"
	"trekka-api/internal/models"
	"trekka-api/internal/services"
	"trekka-api/internal/utils"
)

// Handles a list of images, resolves metadata, updates Firestore, and tracks stats
func processImages(
	ctx context.Context,
	logger *log.Logger,
	storageService *services.StorageService,
	firestoreService *services.FirestoreService,
	images []*models.ImageMetadata,
	onlyEmpty, dryRun bool,
	stats *struct {
		updated, skipped, noGPS, errors int
	},
) {
	for _, img := range images {
		if onlyEmpty && !utils.HasEmptyFields(img) {
			logger.Printf("⏭️  Skipping %s (already has complete data)", img.FileName)
			stats.skipped++
			continue
		}

		logger.Printf("🔄 Processing %s", img.FileName)

		// Fetch file from Storage
		fileData, err := storageService.FetchFile(ctx, img.StoragePath)
		if err != nil {
			logger.Printf("❌ Failed to fetch %s from storage: %v", img.FileName, err)
			stats.errors++
			continue
		}

		if dryRun {
			// Extract metadata but don't persist
			extracted, err := services.ExtractMetadataFromBytes(ctx, img.FileName, img.ContentType, fileData)
			if err != nil {
				logger.Printf("❌ Failed to extract metadata from %s: %v", img.FileName, err)
				stats.errors++
				continue
			}
			logger.Printf("🔍 [DRY] Would update %s -> %s", img.FileName, extracted.GeoLocation)
			stats.updated++
			continue
		}

		// Extract and persist metadata using shared function
		updated, err := services.ExtractAndPersistMetadata(ctx, firestoreService, img.FileName, img.ContentType, fileData, img, services.NewGeocodingService())
		if err != nil {
			logger.Printf("❌ Failed to process %s: %v", img.FileName, err)
			stats.errors++
			continue
		}

		logger.Printf("✅ Updated %s with location: %s", updated.FileName, updated.GeoLocation)
		stats.updated++

		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	logger := log.New(os.Stdout, "[MetadataUpdate] ", log.LstdFlags)

	onlyEmpty := flag.Bool("only-empty", false, "Only update entries with empty GPS/location fields")
	dryRun := flag.Bool("dry-run", false, "Preview changes without updating Firestore")
	backfill := flag.Bool("backfill", false, "Force download from Google Drive (slower but more reliable)")
	skipExisting := flag.Bool("skip-existing", true, "Skip files that already exist in Firestore during backfill")
	flag.Parse()

	if *dryRun {
		logger.Println("DRY RUN - no Firestore writes")
	}
	if *backfill {
		logger.Println("BACKFILL MODE - will download from Drive")
		logger.Println("Rate limiting: 3 seconds between Drive API calls with exponential backoff retry")
	}
	if *onlyEmpty {
		logger.Println("Only updating entries with empty GPS/location fields")
	}

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

	// Optional: Drive client
	var driveSvc *drive.Service
	driveSvc, _ = drive.NewService(ctx, option.WithAPIKey(cfg.GoogleAPIKey))

	// Services
	storageService := services.NewStorageService(storageClient, cfg.FirebaseBucketName)
	firestoreService := services.NewFirestoreService(firestoreClient, cfg.FirestoreCollection)

	// Drive sync service (for backfill mode)
	var driveService *services.DriveService
	if driveSvc != nil && cfg.GoogleDriveFolderID != "" {
		driveFileService := services.NewDriveClient(driveSvc)
		geocoder := services.NewGeocodingService()
		driveService = services.NewDriveService(driveFileService, storageService, firestoreService, geocoder, cfg.GoogleDriveFolderID)
	}

	stats := struct {
		updated, skipped, noGPS, errors int
	}{}

	if *backfill {
		if driveService == nil {
			logger.Fatalf("Backfill mode requires GOOGLE_DRIVE_FOLDER_ID and Drive credentials")
		}

		logger.Println("Starting Drive backfill...")
		// When running from update-metadata in backfill mode, respect the skipExisting flag
		// By default it's true (skip existing), but can be disabled with --skip-existing=false
		if err := driveService.BackfillFromDrive(ctx, *skipExisting); err != nil {
			logger.Fatalf("Backfill failed: %v", err)
		}

		logger.Println("Backfill complete!")
		return
	} else {
		allImages, err := firestoreService.ListImageMetadata(ctx, 0, 0)
		if err != nil {
			logger.Fatalf("list images: %v", err)
		}
		processImages(ctx, logger, storageService, firestoreService, allImages, *onlyEmpty, *dryRun, &stats)

		logger.Printf("Done: updated=%d skipped=%d noGPS=%d errors=%d",
			stats.updated, stats.skipped, stats.noGPS, stats.errors)
	}
}
