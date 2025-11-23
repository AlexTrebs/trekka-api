package handler

import (
	"context"
	"log"
	"net/http"
	"sync"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"trekka-api/internal/config"
	"trekka-api/internal/handlers"
	"trekka-api/internal/middleware"
	"trekka-api/internal/router"
	"trekka-api/internal/services"
)

var (
	handler     http.Handler
	handlerOnce sync.Once
	initErr     error
)

// Initialize handler once and reuse across invocations
func initHandler() {
	handlerOnce.Do(func() {
		ctx := context.Background()

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			initErr = err
			log.Printf("Failed to load configuration: %v", err)
			return
		}

		// Configure Firebase credentials
		var opts []option.ClientOption
		if cfg.FirebaseCredentialsJSON != "" {
			// Use JSON credentials from environment variable (preferred for Vercel)
			opts = append(opts, option.WithCredentialsJSON([]byte(cfg.FirebaseCredentialsJSON)))
		} else if cfg.FirebaseCredentialsPath != "" {
			// Use credentials file (for local development)
			opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCredentialsPath))
		}

		// Initialize Firebase Storage client
		storageClient, err := storage.NewClient(ctx, opts...)
		if err != nil {
			initErr = err
			log.Printf("Failed to create Firebase Storage client: %v", err)
			return
		}

		// Initialize Firestore client
		firestoreClient, err := firestore.NewClient(ctx, cfg.FirebaseProjectID, opts...)
		if err != nil {
			initErr = err
			log.Printf("Failed to create Firestore client: %v", err)
			return
		}

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
		handler = middleware.Logger(mux)
		handler = middleware.CORS(handler, cfg.AllowedOrigins)

		log.Println("Handler initialized successfully")
	})
}

// Handler is the Vercel serverless function entry point
func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize handler on first request
	initHandler()

	// Check if initialization failed
	if initErr != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Serve the request
	handler.ServeHTTP(w, r)
}
