package api

import (
	"context"
	"log"
	"net/http"
	"sync"

	"trekka-api/internal/config"
	"trekka-api/internal/server"
)

var (
	handler http.Handler
	initErr error
	once    sync.Once
)

// Initializes the HTTP handler once and reuses it across invocations.
// Uses sync.Once to ensure thread-safe single initialization without data races.
func initHandler() error {
	once.Do(func() {
		ctx := context.Background()

		// Load and validate configuration
		cfg, err := config.Load()
		if err != nil {
			log.Printf("Failed to load configuration: %v", err)
			initErr = err
			return
		}

		// Initialize all services
		svcs, err := server.InitServices(ctx, cfg)
		if err != nil {
			log.Printf("Failed to initialize services: %v", err)
			initErr = err
			return
		}

		// Create HTTP handler
		wrappedHandler := server.CreateHandler(svcs.Image, cfg.AllowedOrigins, cfg.APIKeys)

		// Start Google Drive background sync if enabled
		// Note: In serverless environments, this goroutine persists across requests
		// within the same container instance
		if svcs.Drive != nil {
			server.StartDriveSync(
				context.Background(),
				svcs.Drive,
				cfg.DriveSyncInterval,
				cfg.DriveBackfillOnStartup,
			)
		}

		// Only set handler after full successful initialization
		handler = wrappedHandler
		log.Println("Handler initialized successfully")
	})

	return initErr
}

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	// Attempt initialization (will succeed immediately if already initialized)
	if err := initHandler(); err != nil {
		log.Printf("Handler initialization failed: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delegate to the initialized handler
	handler.ServeHTTP(w, r)
}
