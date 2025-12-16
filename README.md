# Trekka API

A high-performance Go API for serving and managing photos and videos stored in Firebase Storage with automatic HEIC/HEIF to JPEG conversion, intelligent caching, comprehensive metadata extraction, and reverse geocoding.

## Features

### API Server

- **Image & Video Serving**: Fetch and serve media from Firebase Storage via signed URLs
- **HEIC/HEIF Conversion**: Automatic conversion of HEIC/HEIF images to JPEG format
- **Intelligent Caching**: In-memory LRU cache with configurable TTL to reduce storage API calls
- **Comprehensive Metadata Extraction**:
  - **Images**: EXIF data extraction (GPS coordinates, timestamps, resolution)
  - **Videos**: MP4 metadata extraction using exiftool (GPS, creation date, dimensions)
  - **Reverse Geocoding**: Automatic GPS coordinate to location name conversion using OpenStreetMap Nominatim
- **Pagination Support**: List images with configurable page size and pagination
- **API Key Authentication**: Required for all endpoints except /health
- **Rate Limiting**: Per-IP rate limiting (10 req/sec) to prevent abuse and control costs
- **Swagger/OpenAPI Documentation**: Interactive API documentation at `/swagger/`
- **CORS Support**: Configurable CORS middleware for cross-origin requests
- **Request Tracking**: Request ID middleware for debugging and monitoring
- **Health Checks**: Built-in health check endpoint for monitoring
- **Graceful Shutdown**: Proper cleanup of resources on server termination
- **Docker Support**: Multi-stage Docker build optimized for Cloud Run deployment
- **Cloud Build Caching**: Fast rebuilds with Docker layer caching (1-2 min vs 3-5 min)

### Google Drive Sync (Optional)

- **Automatic Sync**: Monitor Google Drive folder for new images and videos
- **Backfill Support**: Sync all existing media from Drive to Firebase
- **HEIC Conversion**: Automatically converts HEIC/HEIF files to JPEG during sync
- **Metadata Extraction**: Automatic GPS and timestamp extraction during sync
- **Reverse Geocoding**: Converts GPS coordinates to location names (city, country)
- **Duplicate Detection**: Skips files already synced to prevent duplicates
- **Continuous Monitoring**: Watch mode for real-time syncing of new uploads
- **Robust Rate Limiting**: Automatic retry with exponential backoff (5 attempts per file)
- **Smart Error Recovery**: Detects persistent rate limits and pauses automatically
- **Timeout Protection**: 5-minute timeout per download prevents hangs
- **Standalone Tool**: Separate CLI tool for flexible deployment options

### Metadata Management Tools

- **Metadata Update Utilities**: Multiple commands for updating and backfilling metadata
- **Batch Processing**: Update all files or only files with missing data
- **Dry Run Mode**: Preview changes before applying them
- **Google Drive Backfill**: Re-download and re-extract metadata from Google Drive
- **Progress Tracking**: Real-time progress updates during batch operations

## Tech Stack

- **Language**: Go 1.25+
- **Storage**: Firebase Cloud Storage
- **Database**: Cloud Firestore
- **Image Processing**: goheif for HEIC/HEIF conversion
- **Metadata Extraction**:
  - goexif for image EXIF data
  - exiftool for MP4 video metadata
- **Geocoding**: OpenStreetMap Nominatim API
- **Containerization**: Docker & Docker Compose

## Prerequisites

- Go 1.25 or higher
- Firebase project with Cloud Storage and Firestore enabled
- Firebase service account credentials JSON file
- exiftool (for video metadata extraction): `sudo apt-get install libimage-exiftool-perl` or `brew install exiftool`

## Installation

### Clone the repository

```bash
git clone <repository-url>
cd trekka-api
```

### Install dependencies

```bash
make install-deps
```

Or manually:

```bash
go mod download
```

### Install exiftool (required for video metadata)

**Linux:**
```bash
sudo apt-get update
sudo apt-get install libimage-exiftool-perl
```

**macOS:**
```bash
brew install exiftool
```

**Verify installation:**
```bash
exiftool -ver
```

## Configuration

### Environment Variables

Create a `.env` file in the root directory (see `.env.example`):

```env
# Server Configuration
PORT=8080

# Firebase Configuration
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_BUCKET_NAME=your-project-id.appspot.com
FIREBASE_CREDENTIALS_PATH=firebase-service-account.json
# Or use FIREBASE_CREDENTIALS_JSON for raw JSON (e.g., on Vercel)
FIRESTORE_COLLECTION=images

# Cache Configuration
CACHE_TTL=15m
CACHE_CLEANUP_INTERVAL=10m

# Security
API_KEYS=your-secret-api-key-1,your-secret-api-key-2

# CORS origins (comma-separated, use * for all origins)
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# Google Drive Sync (Optional)
GOOGLE_DRIVE_FOLDER_ID=your-drive-folder-id
DRIVE_SYNC_INTERVAL=5m
DRIVE_BACKFILL_ON_STARTUP=false
```

### Firebase Setup

1. Create a Firebase project at [Firebase Console](https://console.firebase.google.com/)
2. Enable Cloud Storage and Firestore
3. Generate a service account key:
   - Go to Project Settings → Service Accounts
   - Click "Generate New Private Key"
   - Save the JSON file as `firebase-service-account.json` in the project root
4. Update the environment variables with your Firebase project details

## Usage

### Development

Run the application locally:

```bash
make run
```

Or with live reload (requires [air](https://github.com/cosmtrek/air)):

```bash
make dev
```

### Build

Build the binary:

```bash
make build
```

The binaries will be created in `bin/`:
- `bin/server` - API server
- `bin/update-metadata` - Metadata update utility

### Docker

Build and run with Docker Compose:

```bash
docker-compose up --build
```

Or manually with Docker:

```bash
docker build -t trekka-api .
docker run -p 8080:8080 \
  -e FIREBASE_BUCKET_NAME=your-bucket \
  -e API_KEYS=your-api-key \
  -v $(pwd)/firebase-service-account.json:/root/firebase-service-account.json:ro \
  trekka-api
```

## API Endpoints

### Health Check

```
GET /health
```

Returns the health status of the API. **No authentication required.**

**Response:**

```json
{
  "status": "healthy"
}
```

### Get Image

```
GET /image?fileName=<filename>
```

Retrieves and serves an image or video from Firebase Storage. Automatically converts HEIC/HEIF images to JPEG.

**Authentication:** Required (API key in `X-API-Key` header)

**Query Parameters:**

- `fileName` (required): Name of the media file

**Response:**

- Returns a redirect (302) to the signed URL for direct download from Firebase Storage
- Headers include:
  - `X-Geo-Location`: Geographic location metadata (if available)
  - `X-Content-Type`: Media MIME type
  - `Cache-Control`: public, max-age=900 (15 minutes)
  - `CDN-Cache-Control`: public, max-age=86400 (24 hours for edge caching)

**Example:**

```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/image?fileName=photo.heic
```

### List Images

```
GET /images/list?limit=<limit>&page=<page>
```

Retrieves a paginated list of image metadata from Firestore.

**Authentication:** Required (API key in `X-API-Key` header)

**Query Parameters:**

- `limit` (optional): Number of items per page (max 1000, default: 1000)
- `page` (optional): Page number (0-indexed, default: 0)

**Response:**

```json
[
  {
    "id": "doc-id",
    "fileName": "photo.jpg",
    "contentType": "image/jpeg",
    "coordinates": {
      "lat": "37.774900",
      "lng": "-122.419400"
    },
    "storagePath": "photo.jpg",
    "geoLocation": "San Francisco, United States",
    "formattedDate": "Wednesday, 15 January 2025, 14:30",
    "resolution": [4032, 3024],
    "takenAt": "2025-01-15T14:30:45Z",
    "createdAt": "2025-01-15T10:30:00Z",
    "updatedAt": "2025-01-15T10:30:00Z"
  }
]
```

**Example:**

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8080/images/list?limit=20&page=0"
```

## Project Structure

```
trekka-api/
├── cmd/
│   ├── server/
│   │   └── main.go              # API server entry point
│   └── update-metadata/
│       ├── main.go              # Metadata update tool
│       └── update-dates.go      # Date/time metadata updater
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── handlers/
│   │   ├── handler.go           # Handler initialization
│   │   ├── health.go            # Health check handler
│   │   └── image.go             # Image/video handlers
│   ├── middleware/
│   │   ├── auth.go              # API key authentication
│   │   ├── cors.go              # CORS middleware
│   │   ├── logger.go            # Request logging
│   │   ├── ratelimit.go         # Rate limiting
│   │   └── requestid.go         # Request ID tracking
│   ├── models/
│   │   └── image.go             # Data models
│   ├── router/
│   │   └── router.go            # Route definitions
│   ├── server/
│   │   └── init.go              # Server initialization
│   ├── services/
│   │   ├── cache.go             # In-memory cache service
│   │   ├── driveClient.go       # Google Drive API client
│   │   ├── driveService.go      # Google Drive sync service
│   │   ├── firestore.go         # Firestore operations
│   │   ├── geocoding.go         # Reverse geocoding service
│   │   ├── image.go             # Image processing service
│   │   ├── metadata.go          # Metadata extraction orchestration
│   │   └── storage.go           # Firebase Storage operations
│   ├── utils/
│   │   ├── drive.go             # Drive utility functions
│   │   ├── exif.go              # EXIF data extraction
│   │   ├── heicFunctions.go     # HEIC/HEIF conversion
│   │   └── mp4.go               # MP4 video metadata extraction
│   └── errors/
│       └── errors.go            # Custom error types
├── docs/
│   ├── docs.go                  # Swagger documentation
│   ├── swagger.json             # OpenAPI spec (JSON)
│   └── swagger.yaml             # OpenAPI spec (YAML)
├── .env.example                 # Example environment variables
├── docker-compose.yml           # Docker Compose configuration
├── Dockerfile                   # Multi-stage Docker build
├── go.mod                       # Go module dependencies
├── go.sum                       # Dependency checksums
├── Makefile                     # Build automation
└── README.md                    # This file
```

## Available Make Commands

### Application Commands

```bash
make help                         # Show all available commands
make build                        # Build the application binaries (server + update-metadata)
make run                          # Run the API server
make dev                          # Run with live reload (requires air)
make test                         # Run tests
make test-coverage                # Run tests with coverage report
make clean                        # Clean build artifacts
make install-deps                 # Install dependencies
make tidy                         # Tidy go.mod
make fmt                          # Format code
make lint                         # Run linter (requires golangci-lint)
```

### Metadata Management Commands

#### Update Metadata from Storage/Drive

```bash
# Update ALL files in Firestore
make sync-update-metadata

# Update ONLY files missing GPS/location data
make sync-update-metadata-empty

# Force re-download from Drive for all files (slower but most accurate)
make sync-update-metadata-backfill

# Force re-download from Drive for files missing GPS/location only
make sync-update-metadata-backfill-empty
```

#### Dry Run (Preview Changes)

```bash
# Preview all updates without making changes
make sync-update-metadata-dry-run

# Preview updates for empty fields only
make sync-update-metadata-empty-dry-run

# Preview backfill updates
make sync-update-metadata-backfill-dry-run
```

**Background Sync (Recommended):** Enable automatic syncing when the API server starts:

```bash
# .env configuration
DRIVE_SYNC_INTERVAL=5m
DRIVE_BACKFILL_ON_STARTUP=true  # First run only

# Start server with sync
make run
```

## Metadata Extraction Features

### Image Metadata (EXIF)

Automatically extracts from JPEG, PNG, HEIC/HEIF images:
- **GPS Coordinates**: Latitude and longitude
- **Timestamps**: Photo capture date and time with timezone
- **Resolution**: Image width and height in pixels
- **Location Names**: Reverse geocoding converts GPS to "City, Country" format

### Video Metadata (MP4)

Automatically extracts from MP4 videos using exiftool:
- **GPS Coordinates**: Latitude and longitude (if embedded)
- **Creation Date**: Video recording timestamp
- **Resolution**: Video dimensions (width × height)
- **Location Names**: Reverse geocoding for videos with GPS data

### Reverse Geocoding

- Uses OpenStreetMap Nominatim API (free, no API key required)
- Converts GPS coordinates to human-readable locations
- In-memory caching to minimize API calls
- Automatic rate limiting (1 request/sec as per Nominatim policy)
- Gracefully handles missing or invalid coordinates

## Deployment

### Google Cloud Run (Recommended)

This project is optimized for Google Cloud Run deployment with Docker containers.

#### Prerequisites

- Google Cloud account with billing enabled
- Firebase service account JSON credentials
- Docker installed locally (for testing)

#### Deployment Steps

1. **Build and deploy**:

   ```bash
   # Build and push to Google Container Registry
   gcloud builds submit --config cloudbuild.yaml

   # Deploy to Cloud Run
   gcloud run deploy trekka-api \
     --image gcr.io/YOUR_PROJECT_ID/trekka-api:latest \
     --platform managed \
     --region us-central1 \
     --allow-unauthenticated \
     --min-instances 0 \
     --max-instances 10 \
     --set-env-vars FIREBASE_PROJECT_ID=your-project-id \
     --set-env-vars FIREBASE_BUCKET_NAME=your-bucket \
     --set-env-vars FIRESTORE_COLLECTION=images \
     --set-secrets FIREBASE_CREDENTIALS_JSON=firebase-credentials:latest \
     --set-secrets API_KEYS=api-keys:latest
   ```

2. **Configure secrets** (one-time setup):

   ```bash
   # Store Firebase credentials in Secret Manager
   gcloud secrets create firebase-credentials \
     --data-file=firebase-service-account.json

   # Store API keys in Secret Manager (comma-separated)
   echo -n "your-api-key-1,your-api-key-2" | \
     gcloud secrets create api-keys --data-file=-
   ```

#### Cloud Build Configuration

The `cloudbuild.yaml` file enables Docker layer caching for faster builds:

- First build: 3-5 minutes
- Subsequent builds: 1-2 minutes (with caching)
- Cost: ~$0.01/month for image storage

## Support

For issues and questions, please open an issue in the GitHub repository.
