package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"trekka-api/internal/config"
	"trekka-api/internal/handlers"
	"trekka-api/internal/middleware"
	"trekka-api/internal/router"
	"trekka-api/internal/services"
)

// Services holds all initialized services for the application
type Services struct {
	Cache     *services.CacheService
	Storage   *services.StorageService
	Firestore *services.FirestoreService
	Image     *services.ImageService
	Drive     *services.DriveService // May be nil if Drive sync is disabled
}

// InitServices initializes all application services based on configuration.
// Returns the initialized services or an error if initialization fails.
func InitServices(ctx context.Context, cfg *config.Config) (*Services, error) {
	// Configure Firebase credentials
	var opts []option.ClientOption
	if cfg.FirebaseCredentialsJSON != "" {
		// Use JSON credentials from environment variable (preferred for Vercel)
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON)))
	} else {
		// Use credentials file (for local development)
		opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCredentialsPath))
	}

	// Initialize Firebase Storage client
	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Initialize Firestore client
	firestoreClient, err := firestore.NewClient(ctx, cfg.FirebaseProjectID, opts...)
	if err != nil {
		return nil, err
	}

	// Initialize core services
	cacheService := services.NewCacheService(cfg.CacheTTL, cfg.CacheCleanupInterval)
	storageService := services.NewStorageService(storageClient, cfg.FirebaseBucketName)
	firestoreService := services.NewFirestoreService(firestoreClient, cfg.FirestoreCollection)
	imageService := services.NewImageService(storageService, cacheService, firestoreService)

	svcs := &Services{
		Cache:     cacheService,
		Storage:   storageService,
		Firestore: firestoreService,
		Image:     imageService,
	}

	// Initialize Google Drive sync if enabled
	if cfg.DriveSyncInterval > 0 {
		if cfg.GoogleDriveFolderID == "" {
			log.Println("⚠️  Drive sync enabled but GOOGLE_DRIVE_FOLDER_ID not set, skipping Drive sync")
		} else if cfg.GoogleAPIKey == "" && cfg.FirebaseCredentialsPath == "" && cfg.FirebaseCredentialsJSON == "" {
			log.Println("⚠️  Drive sync enabled but no credentials available, skipping Drive sync")
		} else {
			log.Println("🔄 Initializing Google Drive sync service...")

			var driveClient *drive.Service
			if cfg.GoogleAPIKey != "" {
				var err error
				driveClient, err = drive.NewService(context.Background(), option.WithAPIKey(cfg.GoogleAPIKey))
				if err != nil {
					log.Printf("⚠️  Failed to create Drive API client: %v", err)
				}
			}

			// Wrap Drive client in DriveFileService
			driveFileService := services.NewDriveClient(driveClient)

			// Create the DriveService using the new constructor
			driveService := services.NewDriveService(
				driveFileService,
				storageService,
				firestoreService,
				services.NewGeocodingService(),
				cfg.GoogleDriveFolderID,
			)

			svcs.Drive = driveService
		}
	}

	return svcs, nil
}

// CreateHandler creates an HTTP handler with all middleware applied
func CreateHandler(imageService *services.ImageService, allowedOrigins []string) http.Handler {
	// Initialize handlers
	h := handlers.New(imageService)

	// Setup router with middleware
	mux := router.Setup(h)

	// Apply global middleware
	wrappedHandler := middleware.Logger(mux)
	wrappedHandler = middleware.CORS(wrappedHandler, allowedOrigins)

	return wrappedHandler
}

// StartDriveSync starts the Google Drive sync service with optional backfill.
// If backfillOnStartup is true, runs a one-time backfill before starting the watch.
// Returns a cancel function to stop the sync gracefully.
func StartDriveSync(ctx context.Context, driveService *services.DriveService, interval time.Duration, backfillOnStartup bool) context.CancelFunc {
	if driveService == nil {
		log.Println("⚠️  Cannot start Drive sync: driveService is nil")
		return func() {} // Return no-op cancel function
	}

	if interval <= 0 {
		log.Printf("⚠️  Invalid Drive sync interval: %v (must be positive)", interval)
		return func() {} // Return no-op cancel function
	}

	driveCtx, cancel := context.WithCancel(ctx)

	go func() {
		// Run backfill if enabled
		if backfillOnStartup {
			log.Println("📦 Running one-time backfill from Google Drive...")
			if err := driveService.BackfillFromDrive(driveCtx); err != nil {
				if err != context.Canceled {
					log.Printf("⚠️  Backfill completed with errors: %v", err)
				} else {
					log.Println("⚠️  Backfill canceled")
					return
				}
			} else {
				log.Println("✅ Backfill completed successfully")
			}
		}

		// Start continuous watch
		log.Printf("🚀 Starting Drive watch (interval: %v)", interval)
		if err := driveService.WatchForChanges(driveCtx, interval); err != nil {
			if err != context.Canceled {
				log.Printf("❌ Drive watch error: %v", err)
			}
		}
	}()

	return cancel
}
