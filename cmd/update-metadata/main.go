package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
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
	firestoreService *services.FirestoreService,
	resolver *services.MetadataResolver,
	images []*models.ImageMetadata, // can wrap Firestore models
	onlyEmpty, dryRun bool,
	stats *struct {
		updated, skipped, noGPS, errors int
	},
) {
	for _, img := range images {
		// Rate limiting: add delay between Drive API calls in backfill mode
		if onlyEmpty && !utils.HasEmptyFields(img, false) {
			logger.Printf("⏭️  Skipping %s (already has complete data)", img.FileName)
			stats.skipped++
			continue
		}

		logger.Printf("🔄 Resolving metadata for %s", img.FileName)

		var resolved *models.ImageMetadata
		var err error

		resolved, err = resolver.ResolveMetadata(ctx, img)

		if err != nil {
			logger.Printf("❌ Failed to resolve %s: %v", img.FileName, err)
			// Check if this is a "no GPS data" error or an actual failure
			if strings.Contains(err.Error(), "no GPS") || strings.Contains(err.Error(), "coordinates not found") {
				stats.noGPS++
			} else {
				stats.errors++
			}
			continue
		}

		if dryRun {
			logger.Printf("🔍 [DRY] Would update %s -> %s", resolved.FileName, resolved.GeoLocation)
			stats.updated++
			continue
		}

		if err := firestoreService.UpdateImageMetadata(ctx, resolved.Id, resolved); err != nil {
			logger.Printf("❌ Save failed for %s: %v", resolved.FileName, err)
			stats.errors++
			continue
		}

		logger.Printf("✅ Updated %s with location: %s", resolved.FileName, resolved.GeoLocation)
		stats.updated++

		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	logger := log.New(os.Stdout, "[MetadataUpdate] ", log.LstdFlags)

	onlyEmpty := flag.Bool("only-empty", false, "Only update entries with empty GPS/location fields")
	dryRun := flag.Bool("dry-run", false, "Preview changes without updating Firestore")
	backfill := flag.Bool("backfill", false, "Force download from Google Drive (slower but more reliable)")
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
	geocoder := services.NewGeocodingService()
	driveFileService := services.NewDriveClient(driveSvc)
	resolver := services.NewMetadataResolver(storageService, driveFileService, geocoder, cfg.GoogleDriveFolderID)

	// Drive sync service (for backfill mode)
	var driveService *services.DriveService
	if driveSvc != nil && cfg.GoogleDriveFolderID != "" {
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
		if err := driveService.BackfillFromDrive(ctx); err != nil {
			logger.Fatalf("Backfill failed: %v", err)
		}

		logger.Println("Backfill complete!")
		return
	} else {
		allImages, err := firestoreService.ListImageMetadata(ctx, 0, 0)
		if err != nil {
			logger.Fatalf("list images: %v", err)
		}
		processImages(ctx, logger, firestoreService, resolver, allImages, *onlyEmpty, *dryRun, &stats)

		logger.Printf("Done: updated=%d skipped=%d noGPS=%d errors=%d",
			stats.updated, stats.skipped, stats.noGPS, stats.errors)
	}
}
