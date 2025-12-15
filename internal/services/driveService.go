package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	"trekka-api/internal/models"
	"trekka-api/internal/utils"
)

type DriveService struct {
	driveClient *DriveClient
	storage     *StorageService
	firestore   *FirestoreService
	folderID    string
	geocoder    *GeocodingService
	logger      *log.Logger
}

func NewDriveService(
	driveClient *DriveClient,
	storage *StorageService,
	firestore *FirestoreService,
	geocoder *GeocodingService,
	folderID string,
) *DriveService {
	logger := log.New(os.Stdout, "[DriveSync] ", log.LstdFlags)
	return &DriveService{
		driveClient: driveClient,
		storage:     storage,
		firestore:   firestore,
		folderID:    folderID,
		geocoder:    geocoder,
		logger:      logger,
	}
}

// Synchronizes a single Drive file. It downloads the file, converts HEIC to JPEG
// when needed, uploads to Storage, then resolves and persists metadata in Firestore.
// If skipExisting is true, files that already exist in Firestore will be skipped entirely.
func (ds *DriveService) SyncFile(ctx context.Context, file *drive.File, skipExisting bool) error {
	// Accept both images and videos
	isImage := strings.HasPrefix(file.MimeType, "image/")
	isVideo := strings.HasPrefix(file.MimeType, "video/")

	if !isImage && !isVideo {
		ds.logger.Printf("Skipping non-media file: %s (%s)", file.Name, file.MimeType)
		return nil
	}

	ds.logger.Printf("Processing %s (%s) [%s]", file.Name, file.Id, file.MimeType)

	// Check if file already exists in Firestore
	existing, _ := ds.firestore.GetImageMetadataByFilename(ctx, file.Name, file.FileExtension)

	if skipExisting && existing != nil {
		ds.logger.Printf("File already exists in Firestore, skipping: %s", file.Name)
		return nil
	}

	if existing != nil && !utils.HasEmptyFields(existing) {
		ds.logger.Printf("Already has complete metadata, skipping: %s", file.Name)
		return nil
	}

	// Download and prepare file
	ds.logger.Printf("Downloading from Drive: %s (%s)", file.Name, file.Id)
	downloadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	raw, err := ds.driveClient.DownloadBytes(downloadCtx, file.Id)
	if err != nil {
		return fmt.Errorf("download from drive failed: %w", err)
	}

	finalName := file.Name
	finalMime := file.MimeType
	finalData := raw

	// Convert HEIC → JPEG if needed
	if utils.IsHeifLike(file.MimeType) {
		ds.logger.Printf("Converting HEIC -> JPEG: %s", file.Name)
		jpeg, err := utils.ConvertHeicToJpeg(raw)
		if err != nil {
			ds.logger.Printf("HEIC conversion failed for %s: %v — continuing with original", file.Name, err)
		} else {
			finalData = jpeg
			finalMime = "image/jpeg"
			if ext := filepath.Ext(file.Name); ext != "" {
				finalName = strings.TrimSuffix(file.Name, ext) + ".jpg"
			}
		}
	}

	// Upload to Storage
	ds.logger.Printf("Uploading to storage: %s", finalName)
	if err := ds.storage.UploadFile(ctx, finalName, finalData, finalMime); err != nil {
		return fmt.Errorf("upload to storage failed: %w", err)
	}

	// Resolve and persist metadata in one sweep (using the file bytes we already have)
	return ds.resolveAndPersist(ctx, finalName, finalMime, finalData, existing)
}

// resolveAndPersist handles metadata resolution and Firestore persistence.
// It extracts metadata from file bytes and creates or updates the Firestore record.
func (ds *DriveService) resolveAndPersist(ctx context.Context, fileName, contentType string, fileData []byte, existing *models.ImageMetadata) error {
	ds.logger.Printf("Extracting metadata from file: %s", fileName)

	metadata, err := ExtractAndPersistMetadata(ctx, ds.firestore, fileName, contentType, fileData, existing, ds.geocoder)
	if err != nil {
		return err
	}

	if metadata.GeoLocation != "" {
		ds.logger.Printf("Successfully synced %s with location: %s", fileName, metadata.GeoLocation)
	} else {
		ds.logger.Printf("Successfully synced %s (no GPS data)", fileName)
	}

	return nil
}

// BackfillFromDrive iterates all files in the Drive folder and syncs them.
// It uses SyncFile for each file.
// If skipExisting is true, files that already exist in Firestore will be skipped entirely.
func (ds *DriveService) BackfillFromDrive(ctx context.Context, skipExisting bool) error {
	if skipExisting {
		ds.logger.Printf("Starting backfill for folder %s (skipping existing files)", ds.folderID)
	} else {
		ds.logger.Printf("Starting backfill for folder %s (processing all files)", ds.folderID)
	}

	files, err := ds.driveClient.ListFilesInFolder(ctx, ds.folderID)
	if err != nil {
		return err
	}

	var (
		newCount, errCount, consecutiveErrors, skippedCount int
	)

	for _, f := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Add delay between files to avoid rate limiting (especially for videos)
		time.Sleep(2 * time.Second)

		// attempt sync
		if err := ds.SyncFile(ctx, f, skipExisting); err != nil {
			ds.logger.Printf("Sync error for %s: %v", f.Name, err)
			errCount++
			consecutiveErrors++

			// If we're getting persistent 403 errors, back off significantly
			// Use proper type assertion to detect rate limit errors
			var apiErr *googleapi.Error
			if errors.As(err, &apiErr) && (apiErr.Code == 403 || apiErr.Code == 429) && consecutiveErrors >= 3 {
				backoffDuration := 5 * time.Minute
				ds.logger.Printf("Detected persistent rate limiting (HTTP %d), pausing for %v", apiErr.Code, backoffDuration)
				time.Sleep(backoffDuration)
				consecutiveErrors = 0 // Reset after backing off
			}
			continue
		}

		// Reset consecutive error count on success
		consecutiveErrors = 0
		newCount++
	}

	ds.logger.Printf("Backfill complete: %d processed, %d skipped, %d errors", newCount, skippedCount, errCount)
	if errCount > 0 {
		return fmt.Errorf("backfill completed with %d errors", errCount)
	}
	return nil
}

// Polls the Drive folder at a fixed interval for new files.
// For production, consider using Drive push notifications.
func (ds *DriveService) WatchForChanges(ctx context.Context, interval time.Duration) error {
	ds.logger.Printf("Starting watch for changes (polling every %v)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastCheck := time.Now()

	for {
		select {
		case <-ctx.Done():
			ds.logger.Println("Watch stopped by context")
			return ctx.Err()
		case <-ticker.C:
			ds.logger.Printf("Checking for new files since %v", lastCheck)

			files, err := ds.driveClient.ListFilesInFolder(ctx, ds.folderID)
			if err != nil {
				ds.logger.Printf("Error listing files: %v", err)
				continue
			}

			newFilesCount := 0
			for _, file := range files {
				createdTime, err := time.Parse(time.RFC3339, file.CreatedTime)
				if err != nil {
					ds.logger.Printf("Failed to parse creation time for %s: %v", file.Name, err)
					continue
				}

				if createdTime.After(lastCheck) {
					ds.logger.Printf("Found new file: %s", file.Name)
					// Don't skip existing files when watching for changes
					if err := ds.SyncFile(ctx, file, false); err != nil {
						ds.logger.Printf("Error syncing new file %s: %v", file.Name, err)
						continue
					}
					newFilesCount++
				}
			}

			if newFilesCount > 0 {
				ds.logger.Printf("Synced %d new files", newFilesCount)
			}

			lastCheck = time.Now()
		}
	}
}
