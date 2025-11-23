package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trekka-api/internal/config"
	"trekka-api/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	// Initialize all services
	svcs, err := server.InitServices(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Create HTTP handler
	handler := server.CreateHandler(svcs.Image, cfg.AllowedOrigins)

	// Start Google Drive background sync if enabled
	var driveCancelFunc context.CancelFunc
	if svcs.Drive != nil {
		driveCancelFunc = server.StartDriveSync(
			context.Background(),
			svcs.Drive,
			cfg.DriveSyncInterval,
			cfg.DriveBackfillOnStartup,
		)
	}

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Stop Drive sync if running
	if driveCancelFunc != nil {
		log.Println("🛑 Stopping Drive sync...")
		driveCancelFunc()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
