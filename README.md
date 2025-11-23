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

### Vercel (Serverless)

This project is optimized for Vercel deployment with serverless functions.

#### Prerequisites

- Vercel account
- Firebase service account JSON credentials

#### Deployment Steps

1. **Install Vercel CLI** (optional):

   ```bash
   npm i -g vercel
   ```

2. **Set up environment variables in Vercel**:

   Go to your Vercel project settings → Environment Variables and add:

   ```
   FIREBASE_PROJECT_ID=your-project-id
   FIREBASE_BUCKET_NAME=your-bucket-name
   FIREBASE_CREDENTIALS_JSON={"type":"service_account",...}
   FIRESTORE_COLLECTION=images
   CACHE_TTL=12h
   CACHE_CLEANUP_INTERVAL=10m
   ALLOWED_ORIGINS=https://yourdomain.com
   ```

   **Important**: For `FIREBASE_CREDENTIALS_JSON`, paste your entire Firebase service account JSON as a single-line string.

3. **Deploy**:

   ```bash
   vercel --prod
   ```

   Or connect your GitHub repository to Vercel for automatic deployments.

#### Vercel Configuration

The `vercel.json` file configures:

- Go runtime for the serverless function
- Route all requests to `api/index.go`

#### Performance Notes

- The serverless function uses singleton pattern with double-checked locking to reuse Firebase clients across invocations
- First request may be slower due to cold start
- Subsequent requests within the same instance are fast

For more details, see [VERCEL_DEPLOYMENT.md](VERCEL_DEPLOYMENT.md).

### Docker / Traditional Hosting

Deploy as a containerized application to any platform:

1. **Build the Docker image**:

   ```bash
   docker build -t trekka-api .
   ```

2. **Push to container registry**:

   ```bash
   docker tag trekka-api your-registry/trekka-api:latest
   docker push your-registry/trekka-api:latest
   ```

3. **Deploy to your platform**:
   - **Google Cloud Run**: `gcloud run deploy`
   - **AWS ECS/Fargate**: Create task definition and service
   - **Kubernetes**: Use provided Docker image in deployment

4. **Configure secrets**:
   - Mount `firebase-service-account.json` as a secret volume
   - Or use `FIREBASE_CREDENTIALS_JSON` environment variable
   - Set all required environment variables from `.env.example`

#### Example: Google Cloud Run

```bash
gcloud run deploy trekka-api \
  --image your-registry/trekka-api:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars FIREBASE_PROJECT_ID=your-project-id \
  --set-env-vars FIREBASE_BUCKET_NAME=your-bucket \
  --set-env-vars FIRESTORE_COLLECTION=images
```

## Security Considerations

### Credentials Management

- **Never commit Firebase service account credentials to version control**
- Add `firebase-service-account.json` to `.gitignore`
- Use environment variables or secret management systems for sensitive configuration
- For Vercel: Use `FIREBASE_CREDENTIALS_JSON` environment variable
- For Docker: Mount credentials as read-only secrets

### API Security

- **CORS**: Restrict `ALLOWED_ORIGINS` in production (avoid using `*`)
  ```
  ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
  ```
- **Path Traversal Protection**: Implemented in image handlers to prevent directory traversal attacks
- **Input Validation**: All user inputs are validated for type, format, and length
- **File Size Limits**: Maximum file size is capped at 50MB to prevent memory exhaustion
- **Rate Limiting**: Consider adding rate limiting middleware for production use
- **Authentication**: This API currently has no authentication - add authentication/authorization for production

### Production Recommendations

1. Implement authentication (Firebase Auth, JWT, API keys)
2. Add rate limiting to prevent abuse
3. Enable logging and monitoring
4. Use HTTPS only (enforced by Vercel and most cloud platforms)
5. Regularly rotate service account credentials
6. Set up alerts for unusual access patterns

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
