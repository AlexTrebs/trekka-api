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
	resolver    *MetadataResolver
	storage     *StorageService
	firestore   *FirestoreService
	folderID    string
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
	resolver := NewMetadataResolver(storage, driveClient, geocoder, folderID)
	return &DriveService{
		driveClient: driveClient,
		resolver:    resolver,
		storage:     storage,
		firestore:   firestore,
		folderID:    folderID,
		logger:      logger,
	}
}

// Synchronizes a single Drive file. It downloads the file, converts HEIC to JPEG
// when needed, uploads to Storage, then resolves and persists metadata in Firestore.
func (ds *DriveService) SyncFile(ctx context.Context, file *drive.File) error {
	if !strings.HasPrefix(file.MimeType, "image/") {
		ds.logger.Printf("Skipping non-image: %s", file.Name)
		return nil
	}

	ds.logger.Printf("Processing %s (%s)", file.Name, file.Id)

	// Check existing Firestore metadata (by filename, not Drive ID)
	// Try both original filename and converted filename (HEIC -> JPG)
	ds.logger.Printf("Checking Firestore for existing metadata: %s", file.Name)
	existing, err := ds.firestore.GetImageMetadataByFilename(ctx, file.Name, file.FileExtension)

	// If not found with original name and it's HEIC, try with .jpg extension
	if err != nil && utils.IsHeifLike(file.MimeType) {
		convertedName := strings.TrimSuffix(file.Name, filepath.Ext(file.Name)) + ".jpg"
		ds.logger.Printf("HEIC file not found, trying converted name: %s", convertedName)
		existing, err = ds.firestore.GetImageMetadataByFilename(ctx, convertedName, "jpg")
	}

	if existing != nil && !utils.HasEmptyFields(existing, false) {
		ds.logger.Printf("Already has complete metadata, skipping: %s", file.Name)
		return nil
	} else if existing != nil {
		ds.logger.Printf("File found in storage but missing metadata: %s", file.Name)
	} else if err != nil {
		ds.logger.Printf("File not found in Firestore, will create new entry: %s", file.Name)
	}

	// Download bytes from Drive with timeout
	ds.logger.Printf("Starting download from Drive: %s (%s)", file.Name, file.Id)
	downloadCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	raw, err := ds.driveClient.DownloadBytes(downloadCtx, file.Id)
	if err != nil {
		return fmt.Errorf("download from drive failed: %w", err)
	}
	ds.logger.Printf("Downloaded %d bytes from drive: %s", len(raw), file.Id)

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

	// Create minimal Firestore metadata if not present so resolver can use storage fetch
	if existing == nil {
		ds.logger.Printf("Uploading to firebase storage: %s", file.Id)

		// Upload to Storage
		if err := ds.storage.UploadFile(ctx, finalName, finalData, finalMime); err != nil {
			return fmt.Errorf("upload to storage failed: %w", err)
		}

		now := time.Now()
		existing = &models.ImageMetadata{
			FileName:    finalName,
			ContentType: finalMime,
			StoragePath: finalName,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		coords, timestamp, err := ds.resolver.extractDataFromDriveMetadata(file)
		formatTs, err := utils.FormatTimestamp(timestamp)

		if coords != nil {
			existing.Coordinates = *coords
		}

		if formatTs != "" {
			existing.FormattedDate = formatTs
		}

		ds.logger.Printf("Uploading to firebaseDB: %s", file.Id)
		firestoreID, err := ds.firestore.CreateImageMetadata(ctx, existing)
		if err != nil {
			return fmt.Errorf("create metadata failed for %s: %w", finalName, err)
		}

		// Set the ID to the Firestore-generated document ID
		existing.Id = firestoreID
	}
	ds.logger.Printf("Resolving metadata: %s", file.Id)

	// Resolve metadata (will read from storage + fallback to drive)
	updated, err := ds.resolver.ResolveMetadata(ctx, existing)
	if err != nil {
		return fmt.Errorf("resolve metadata failed: %w", err)
	}
	ds.logger.Printf("Metadata resolved: %s (coords: %s, location: %s, date: %s)",
		file.Id, updated.Coordinates, updated.GeoLocation, updated.FormattedDate)

	// Persist metadata (update)
	if err := ds.firestore.UpdateImageMetadata(ctx, updated.Id, updated); err != nil {
		return fmt.Errorf("update firestore failed: %w", err)
	}

	if updated.GeoLocation != "" {
		ds.logger.Printf("Successfully synced %s with location: %s", updated.FileName, updated.GeoLocation)
	} else {
		ds.logger.Printf("Successfully synced %s (no GPS data)", updated.FileName)
	}
	return nil
}

// BackfillFromDrive iterates all files in the Drive folder and syncs them.
// It uses SyncFile for each file.
func (ds *DriveService) BackfillFromDrive(ctx context.Context) error {
	ds.logger.Printf("Starting backfill for folder %s", ds.folderID)

	files, err := ds.driveClient.ListFilesInFolder(ctx, ds.folderID)
	if err != nil {
		return err
	}

	var (
		newCount, errCount, consecutiveErrors int
	)

	for _, f := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// attempt sync
		if err := ds.SyncFile(ctx, f); err != nil {
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

	ds.logger.Printf("Backfill complete: %d processed, %d errors", newCount, errCount)
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
					if err := ds.SyncFile(ctx, file); err != nil {
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
