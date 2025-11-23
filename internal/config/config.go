package config

import (
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
		CacheTTL:                getDurationEnv("CACHE_TTL", 12*time.Hour),
		CacheCleanupInterval:    getDurationEnv("CACHE_CLEANUP_INTERVAL", 10*time.Minute),
		AllowedOrigins:          getList("ALLOWED_ORIGINS", []string{"*"}),
	}

	return cfg, nil
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
