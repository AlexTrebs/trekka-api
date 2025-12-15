package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                    string
	FirebaseProjectID       string
	FirebaseBucketName      string
	FirebaseCredentialsPath string
	FirebaseCredentialsJSON string // For Vercel: raw JSON string
	FirestoreCollection     string
	CacheTTL                time.Duration
	CacheCleanupInterval    time.Duration
	AllowedOrigins          []string
	APIKeys                 []string      // API keys for authentication (comma-separated)
	GoogleDriveFolderID     string        // Google Drive folder ID for sync
	GoogleAPIKey            string        // Google API key for Drive access (alternative to service account)
	DriveSyncInterval       time.Duration // How often to check Drive for new files (default: 5 minutes)
	DriveBackfillOnStartup  bool          // Run one-time backfill on server startup before starting watch
	IsVercel                bool          // Detected via VERCEL env var
}

// Load reads configuration from environment variables and .env file.
// It loads the .env file if present, then populates the Config struct.
// Returns an error if required configuration is missing.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Port:                    getEnv("PORT", "8080"),
		FirebaseProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseBucketName:      getEnv("FIREBASE_BUCKET_NAME", ""),
		FirebaseCredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", "firebase-service-account.json"),
		FirebaseCredentialsJSON: getEnv("FIREBASE_CREDENTIALS_JSON", ""),
		FirestoreCollection:     getEnv("FIRESTORE_COLLECTION", "images"),
		CacheTTL:                getDurationEnv("CACHE_TTL", 15*time.Minute),
		CacheCleanupInterval:    getDurationEnv("CACHE_CLEANUP_INTERVAL", 10*time.Minute),
		AllowedOrigins:          getList("ALLOWED_ORIGINS", []string{"*"}),
		APIKeys:                 getList("API_KEYS", []string{}),
		GoogleDriveFolderID:     getEnv("GOOGLE_DRIVE_FOLDER_ID", ""),
		GoogleAPIKey:            getEnv("GOOGLE_API_KEY", ""),
		DriveSyncInterval:       getDurationEnv("DRIVE_SYNC_INTERVAL", 5*time.Minute),
		DriveBackfillOnStartup:  getBoolEnv("DRIVE_BACKFILL_ON_STARTUP", false),
		IsVercel:                getEnv("VERCEL", "") != "",
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	if c.FirebaseProjectID == "" {
		return fmt.Errorf("FIREBASE_PROJECT_ID is required")
	}
	if c.FirebaseBucketName == "" {
		return fmt.Errorf("FIREBASE_BUCKET_NAME is required")
	}
	if c.FirebaseCredentialsJSON == "" && c.FirebaseCredentialsPath == "" {
		return fmt.Errorf("either FIREBASE_CREDENTIALS_JSON or FIREBASE_CREDENTIALS_PATH must be set")
	}
	if c.FirestoreCollection == "" {
		return fmt.Errorf("FIRESTORE_COLLECTION is required")
	}
	if c.CacheTTL <= 0 {
		return fmt.Errorf("CACHE_TTL must be positive")
	}
	if c.CacheCleanupInterval <= 0 {
		return fmt.Errorf("CACHE_CLEANUP_INTERVAL must be positive")
	}
	if len(c.APIKeys) == 0 {
		return fmt.Errorf("API_KEYS is required (comma-separated list of API keys)")
	}
	return nil
}

// Retrieves an environment variable or returns a default value if not set.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// Retrieves a duration from environment variable or returns a default value.
// It supports both time.Duration format (e.g., "10m", "12h") and integer minutes.
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		if minutes, err := strconv.Atoi(value); err == nil {
			return time.Duration(minutes) * time.Minute
		}
	}
	return defaultValue
}

// Retrieves a comma-separated list from environment variable or returns a default value.
func getList(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// Retrieves a boolean from environment variable or returns a default value.
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
