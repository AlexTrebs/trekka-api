# Trekka API

A high-performance Go API for serving and managing images stored in Firebase Storage with automatic HEIC/HEIF to JPEG conversion, intelligent caching, and metadata management via Firestore.

## Features

- **Image Serving**: Fetch and serve images from Firebase Storage
- **HEIC/HEIF Conversion**: Automatic conversion of HEIC/HEIF images to JPEG format
- **Intelligent Caching**: In-memory LRU cache with configurable TTL to reduce storage API calls
- **Metadata Management**: Store and retrieve image metadata including geolocation data via Firestore
- **Pagination Support**: List images with configurable page size and pagination
- **CORS Support**: Configurable CORS middleware for cross-origin requests
- **Health Checks**: Built-in health check endpoint for monitoring
- **Graceful Shutdown**: Proper cleanup of resources on server termination
- **Docker Support**: Multi-stage Docker build for optimized production deployments

## Tech Stack

- **Language**: Go 1.25+
- **Storage**: Firebase Cloud Storage
- **Database**: Cloud Firestore
- **Image Processing**: goheif for HEIC/HEIF conversion
- **Containerization**: Docker & Docker Compose

## Prerequisites

- Go 1.25 or higher
- Firebase project with Cloud Storage and Firestore enabled
- Firebase service account credentials JSON file

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
FIRESTORE_COLLECTION=images

# Cache Configuration
CACHE_TTL=12h
CACHE_CLEANUP_INTERVAL=10m

# CORS origins
ALLOWED_ORIGINS="*"
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

The binary will be created at `bin/server`.

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
  -v $(pwd)/firebase-service-account.json:/root/firebase-service-account.json:ro \
  trekka-api
```

## API Endpoints

### Health Check

```
GET /health
```

Returns the health status of the API.

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

Retrieves and serves an image from Firebase Storage. Automatically converts HEIC/HEIF images to JPEG.

**Query Parameters:**

- `fileName` (required): Name of the image file

**Response:**

- Returns the image binary data
- Headers include:
  - `Content-Type`: Image MIME type
  - `Content-Disposition`: Inline with filename
  - `X-Geo-Location`: Geographic location metadata (if available)
  - `Cache-Control`: public, max-age=3600

**Example:**

```bash
curl http://localhost:8080/image?fileName=photo.heic
```

### List Images

```
GET /images/list?limit=<limit>&page=<page>
```

Retrieves a paginated list of image metadata from Firestore.

**Query Parameters:**

- `limit` (optional): Number of items per page (default: all)
- `page` (optional): Page number (0-indexed, default: 0)

**Response:**

```json
[
  {
    "id": "doc-id",
    "fileName": "photo.jpg",
    "contentType": "image/jpeg",
    "size": 1024000,
    "geoLocation": "37.7749,-122.4194",
    "uploadedAt": "2025-01-15T10:30:00Z"
  }
]
```

**Example:**

```bash
curl http://localhost:8080/images/list?limit=20&page=0
```

## Project Structure

```
trekka-api/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration loading
│   ├── handlers/
│   │   ├── handler.go        # Handler initialization
│   │   ├── health.go         # Health check handler
│   │   └── image.go          # Image handlers
│   ├── middleware/
│   │   ├── cors.go           # CORS middleware
│   │   └── logger.go         # Request logging middleware
│   ├── models/
│   │   └── image.go          # Data models
│   ├── router/
│   │   └── router.go         # Route definitions
│   ├── services/
│   │   ├── cache.go          # In-memory cache service
│   │   ├── firestore.go      # Firestore operations
│   │   ├── image.go          # Image processing service
│   │   └── storage.go        # Firebase Storage operations
│   └── utils/
│       └── heicFunctions.go  # HEIC/HEIF conversion utilities
├── .env.example              # Example environment variables
├── docker-compose.yml        # Docker Compose configuration
├── Dockerfile                # Multi-stage Docker build
├── go.mod                    # Go module dependencies
├── go.sum                    # Dependency checksums
├── Makefile                  # Build automation
└── README.md                 # This file
```

## Available Make Commands

```bash
make help              # Show all available commands
make build             # Build the application binary
make run               # Run the application
make dev               # Run with live reload (requires air)
make test              # Run tests
make test-coverage     # Run tests with coverage report
make clean             # Clean build artifacts
make install-deps      # Install dependencies
make tidy              # Tidy go.mod
make fmt               # Format code
make lint              # Run linter (requires golangci-lint)
```

## Testing

Run tests:

```bash
make test
```

Run tests with coverage:

```bash
make test-coverage
```

This generates `coverage.html` which you can open in a browser.

## Performance Features

### Caching Strategy

The API implements an in-memory LRU cache with:

- Configurable TTL (Time To Live)
- Automatic cleanup of expired entries
- Cached converted JPEG images to avoid repeated conversions
- Significant reduction in Firebase Storage API calls

### HEIC/HEIF Conversion

- Automatic detection of HEIC/HEIF images
- On-the-fly conversion to JPEG
- Converted images are cached to improve subsequent requests
- Graceful fallback if conversion fails

## Deployment

### Vercel

This project includes Vercel deployment configuration. See [VERCEL_DEPLOYMENT.md](VERCEL_DEPLOYMENT.md) for details.

### Traditional Hosting

1. Build the Docker image
2. Push to your container registry
3. Deploy to your preferred hosting platform (GCP Cloud Run, AWS ECS, etc.)
4. Ensure Firebase credentials are securely mounted or provided via secrets

## Security Considerations

- Firebase service account credentials should never be committed to version control
- Use environment variables or secret management systems for sensitive configuration
- CORS origins should be restricted in production environments
- Consider implementing authentication/authorization for production use

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[Add your license here]

## Support

For issues and questions, please open an issue in the GitHub repository.
