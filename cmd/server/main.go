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

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"trekka-api/internal/config"
	"trekka-api/internal/handlers"
	"trekka-api/internal/middleware"
	"trekka-api/internal/router"
	"trekka-api/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	// Initialize Firebase Storage client
	storageClient, err := storage.NewClient(
		ctx,
		option.WithCredentialsFile(cfg.FirebaseCredentialsPath),
	)

	if err != nil {
		log.Fatalf("Failed to create Firebase Storage client: %v", err)
	}
	defer storageClient.Close()

	// Initialize Firestore client
	firestoreClient, err := firestore.NewClient(
		ctx,
		cfg.FirebaseProjectID,
		option.WithCredentialsFile(cfg.FirebaseCredentialsPath),
	)

	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	// Initialize services
	cacheService := services.NewCacheService(cfg.CacheTTL, cfg.CacheCleanupInterval)
	storageService := services.NewStorageService(storageClient, cfg.FirebaseBucketName)
	firestoreService := services.NewFirestoreService(firestoreClient, cfg.FirestoreCollection)
	imageService := services.NewImageService(storageService, cacheService, firestoreService)

	// Initialize handlers
	h := handlers.New(imageService)

	// Setup router with middleware
	mux := router.Setup(h)

	// Apply global middleware
	handler := middleware.Logger(mux)
	handler = middleware.CORS(handler, cfg.AllowedOrigins)

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
