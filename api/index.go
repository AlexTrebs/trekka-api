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
	mu          sync.Mutex
	initErr     error
	initialized bool
)

// initHandler initializes the HTTP handler once and reuses it across invocations.
// Uses double-checked locking for optimal performance in serverless environments.
// Returns an error if initialization fails, allowing retry on next request.
//
// Note: Firebase clients are not explicitly closed as Vercel's serverless
// runtime handles resource cleanup on function termination.
func initHandler() error {
	// Fast path: check without lock (first check)
	if initialized && initErr == nil {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring lock
	if initialized {
		return initErr
	}

	ctx := context.Background()

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		initErr = err
		initialized = true
		return err
	}

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
		log.Printf("Failed to create Firebase Storage client: %v", err)
		initErr = err
		initialized = true
		return err
	}

	// Initialize Firestore client
	firestoreClient, err := firestore.NewClient(ctx, cfg.FirebaseProjectID, opts...)
	if err != nil {
		log.Printf("Failed to create Firestore client: %v", err)
		initErr = err
		initialized = true
		return err
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
	wrappedHandler := middleware.Logger(mux)
	wrappedHandler = middleware.CORS(wrappedHandler, cfg.AllowedOrigins)

	// Only set handler and mark as initialized after full successful initialization
	handler = wrappedHandler
	initialized = true
	initErr = nil

	log.Println("Handler initialized successfully")
	return nil
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
