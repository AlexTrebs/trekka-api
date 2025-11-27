# Google Drive Sync

This document explains how to set up and use the Google Drive sync functionality to automatically sync photos from a Google Drive folder to Firebase Storage and Firestore.

## Overview

The Google Drive sync tool:

- Monitors a specified Google Drive folder for image files
- Automatically converts HEIC/HEIF images to JPEG
- Uploads images to Firebase Storage
- Creates metadata entries in Firestore
- Supports both one-time backfill and continuous monitoring

## Features

- **Automatic HEIC Conversion**: Converts Apple HEIC/HEIF images to JPEG format
- **Backfill Support**: Sync all existing files in the Drive folder
- **Watch Mode**: Continuously monitor for new files
- **Duplicate Detection**: Skips files that already exist in Firestore
- **Metadata Extraction**: Extracts and stores image metadata
- **Error Handling**: Continues processing even if individual files fail

## Setup

### Authentication Methods

You can authenticate with Google Drive using **two methods**:

#### Method 1: Google API Key (Recommended for simplicity)

**Pros**: Simpler setup, no folder sharing needed if folder is public
**Cons**: Read-only access only, folder must be public or shared with link

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select your Firebase project (or create a new one)
3. Navigate to **APIs & Services** → **Credentials**
4. Click **Create Credentials** → **API Key**
5. Copy the API key
6. (Optional but recommended) Click **Restrict Key**:
   - Under "API restrictions", select "Restrict key"
   - Enable only "Google Drive API"
   - Click **Save**

#### Method 2: Firebase Service Account

**Pros**: Full access, can use same credentials as Firebase
**Cons**: Requires folder sharing setup

Use your existing Firebase service account credentials.

### 1. Google Drive API Access

Enable Google Drive API in your Google Cloud project:

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select your Firebase project
3. Navigate to **APIs & Services** → **Library**
4. Search for "Google Drive API" and click **Enable**

### 2. Configure Drive Folder Access

#### If using API Key:

**Option A: Public folder**

1. Right-click your Drive folder → **Share** → **Get link**
2. Set to "Anyone with the link" can view
3. Copy the folder ID from the URL

**Option B: Shared with specific people**

1. Share the folder with the email addresses that need access
2. Ensure the folder has "Viewer" or higher permissions
3. Copy the folder ID from the URL

#### If using Service Account:

1. Open Google Drive and navigate to the folder you want to sync
2. Click **Share** on the folder
3. Add your Firebase service account email (found in `firebase-service-account.json`, field `client_email`)
4. Give it **Viewer** permissions
5. Copy the folder ID from the URL

**Getting Folder ID:**

- URL format: `https://drive.google.com/drive/folders/FOLDER_ID_HERE`
- Example: `https://drive.google.com/drive/folders/1a2b3c4d5e6f7g8h9i0j`
- The folder ID is: `1a2b3c4d5e6f7g8h9i0j`

### 3. Configure Environment

Add to your `.env` file:

**For API Key method:**

```env
GOOGLE_DRIVE_FOLDER_ID=your-folder-id-here
GOOGLE_API_KEY=your-api-key-here
```

**For Service Account method:**

```env
GOOGLE_DRIVE_FOLDER_ID=your-folder-id-here
# Firebase credentials already configured above
```

## Usage

### Background Sync (Automatic with API Server)

The simplest way to use Drive sync is to enable automatic background syncing when the API server starts. This is ideal for production deployments where you want continuous monitoring without running a separate service.

**Configuration:**

Add to your `.env` file:

```env
ENABLE_DRIVE_SYNC=true
DRIVE_SYNC_INTERVAL=5m
DRIVE_BACKFILL_ON_STARTUP=false  # Optional: set to true to sync all existing files on startup
```

**Start the server:**

```bash
# Using Make
make run

# Or run directly
go run cmd/server/main.go
```

When the server starts with `ENABLE_DRIVE_SYNC=true`, you'll see:

```
🔄 Initializing Google Drive background sync...
🚀 Starting Drive watch (interval: 5m0s)
Server starting on port 8080
```

If you also enable `DRIVE_BACKFILL_ON_STARTUP=true`, you'll see:

```
🔄 Initializing Google Drive background sync...
📦 Running one-time backfill from Google Drive...
[DriveSync] Found 150 files in Google Drive folder
[DriveSync] Processing files...
✅ Backfill completed successfully
🚀 Starting Drive watch (interval: 5m0s)
Server starting on port 8080
```

**How it works:**

1. Server starts and initializes all services (Firebase, Firestore, etc.)
2. If `ENABLE_DRIVE_SYNC=true`, the Drive service is initialized
3. If `DRIVE_BACKFILL_ON_STARTUP=true`, runs a one-time sync of all existing Drive files
   - Syncs new files that don't exist in Firestore
   - Updates existing files missing GPS/location data
   - Skips files with complete metadata
4. A background goroutine starts watching for new files at the specified interval
5. The API server handles requests normally while sync runs in the background
6. On shutdown (Ctrl+C or SIGTERM), the sync is gracefully stopped before the server exits

**Benefits:**

- ✅ No separate service to manage
- ✅ Runs automatically with your API server
- ✅ Shares the same Firebase/Drive credentials
- ✅ Graceful shutdown ensures sync completes cleanly
- ✅ Perfect for Vercel, Docker, or traditional deployments

**Configuration options:**

- `ENABLE_DRIVE_SYNC`: Set to `true` to enable (default: `false`)
- `DRIVE_SYNC_INTERVAL`: Check interval (examples: `5m`, `10m`, `1h`, default: `5m`)
- `DRIVE_BACKFILL_ON_STARTUP`: Set to `true` to sync all existing files on startup (default: `false`)

**When to use:**

- Production deployments where continuous sync is needed
- Deployments where you want "set it and forget it" behavior
- When you don't want to manage a separate sync service
- **Initial setup**: Enable `DRIVE_BACKFILL_ON_STARTUP=true` on first run to sync all existing files
- **Ongoing sync**: Keep `DRIVE_BACKFILL_ON_STARTUP=false` for normal operation (only new files)

**When not to use:**

- Updating metadata for existing files only (use update-metadata tool below)
- Custom sync schedules requiring more control (use standalone sync tool below)

**Recommended setup:**

1. **First deployment**: Set `DRIVE_BACKFILL_ON_STARTUP=true` to sync all existing files
2. **After initial sync**: Set `DRIVE_BACKFILL_ON_STARTUP=false` for faster startups
3. **To re-sync**: Temporarily set back to `true` if you manually add many files to Drive

### How Backfill Works

When you enable `DRIVE_BACKFILL_ON_STARTUP=true`, the server performs a one-time sync on startup:

1. Lists all files in the specified Drive folder
2. Checks which files already exist in Firestore
3. For **new files**: Downloads, converts (if needed), uploads to Storage, and creates metadata with GPS data
4. For **existing files without GPS data**: Downloads, extracts GPS from EXIF, updates metadata
5. For **existing files with complete metadata**: Skips
6. After completion, starts the continuous watch mode

**Smart Sync Behavior:**

- ✅ New images → Full sync (upload + create metadata with EXIF)
- ✅ Existing images missing GPS → Download + extract EXIF + update metadata
- ✅ Existing images with GPS → Skip (no download needed)

### Update Metadata for Existing Files

If you have existing files in Firestore missing GPS data, you can update them:

```bash
# Preview what would be updated (dry run)
make sync-update-metadata-dry-run

# Update all files missing GPS data
make sync-update-metadata

# Or run directly
go run cmd/update-metadata/main.go
go run cmd/update-metadata/main.go -dry-run  # Preview mode
```

**How it works:**

1. Fetches all image metadata from Firestore
2. Identifies entries missing GPS data (`geoLocation` is empty)
3. Downloads each file from **Firebase Storage** (not Drive - much faster!)
4. Extracts GPS coordinates from EXIF data
5. **Fallback**: If EXIF extraction fails from Storage (e.g., `EOF` error), automatically tries downloading the original from Google Drive
6. **Reverse geocodes** coordinates to city and country using OpenStreetMap
7. Updates Firestore metadata with location string (e.g., "San Francisco, United States")

**Benefits:**

- ✅ Uses Firebase Storage (already uploaded files) instead of re-downloading from Drive
- ✅ Much faster than downloading from Drive
- ✅ Works on all files in Firestore, not just Drive-synced ones
- ✅ Dry-run mode to preview changes before applying
- ✅ Automatic reverse geocoding to human-readable locations
- ✅ Respects Nominatim API usage policy (1 request/second)
- ✅ **Smart fallback**: Automatically tries Google Drive if EXIF extraction fails from Storage

**Drive Fallback:**

The tool automatically handles cases where Firebase Storage files are missing EXIF data (e.g., converted JPEGs, stripped metadata):

- If `GOOGLE_DRIVE_FOLDER_ID` is configured, the tool will:
  1. Detect EXIF extraction failure from Storage
  2. Search for the original file in your Drive folder
  3. Download the original from Drive
  4. Extract EXIF from the original file

This is particularly useful for images that were converted from HEIC to JPEG during upload, as the JPEG conversion might strip some EXIF data.

**Rate Limiting & Retry Logic:**

To avoid Google Drive API rate limits (403 errors), the tool implements robust protection:

**Base Rate Limiting:**

- 3-second minimum delay between all Drive API calls
- Shared rate limiter across all operations (list, download, search)
- Prevents hitting Google's queries-per-100-seconds quota

**Exponential Backoff Retry:**

- Downloads: Up to 5 retry attempts with exponential backoff (5s, 10s, 20s, 40s, 80s)
- List operations: Up to 3 retry attempts per page
- Search operations: Up to 3 retry attempts
- Automatically detects 403 rate limit errors and retries

**Persistent Rate Limit Detection:**

- If 3 consecutive files fail with 403 errors, pauses for 5 minutes
- Prevents cascading failures when Google has flagged your API key/IP
- Automatically resumes after cooldown period

**Timeout Protection:**

- 5-minute timeout per file download
- Prevents indefinite hangs on large files or network issues
- Context-aware cancellation for clean shutdown

**Performance Impact:**

- Normal operation: ~20 files per minute (3s per file)
- With retries: ~10-15 files per minute
- After rate limit cooldown: Automatically resumes at normal speed

**Important Notes:**

- **API Key vs OAuth**: API keys have much stricter limits than OAuth 2.0 service accounts
- **Persistent rate limits**: Google may block your API key/IP for hours after detecting "automated queries"
- **First-time backfill**: For large folders (100+ files), consider running overnight or in smaller batches
- **Recovery**: If you hit persistent 403 errors, wait 6-12 hours before retrying

For large batches, the process will be slower but more reliable. The tool prioritizes reliability over speed to avoid being blocked by Google.

## File Processing

### Supported Image Formats

The sync tool processes all image files, including:

- JPEG/JPG
- PNG
- GIF
- HEIC/HEIF (automatically converted to JPEG)
- WebP
- Other formats supported by Firebase Storage

### HEIC/HEIF Conversion

When HEIC or HEIF files are detected:

1. File is downloaded from Google Drive
2. Converted to JPEG format using goheif library
3. File extension changed from `.heic` to `.jpg`
4. JPEG version uploaded to Firebase Storage
5. Metadata stored with JPEG content type

If conversion fails, the original file is uploaded instead.

### Metadata Stored

For each image, the following metadata is stored in Firestore:

```json
{
  "fileName": "photo.jpg",
  "contentType": "image/jpeg",
  "storagePath": "photo.jpg",
  "geoLocation": "San Francisco, United States",
  "coordinates": {
    "lat": "37.774900",
    "lng": "-122.419400"
  },
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:30:00Z"
}
```

**GeoLocation Field:**

- Automatically populated using reverse geocoding
- Format: `"City, Country"` (e.g., "San Francisco, United States")
- Uses OpenStreetMap Nominatim API
- Cached to minimize API calls
- Falls back to coordinates if geocoding fails

## Deployment Recommendations

**For most deployments**, use the **Background Sync** mode which runs automatically with the API server:

```env
ENABLE_DRIVE_SYNC=true
DRIVE_SYNC_INTERVAL=5m
DRIVE_BACKFILL_ON_STARTUP=true  # First deployment only
```

This works seamlessly with:

- **Docker**: No additional containers needed
- **Vercel**: Runs in serverless functions (with caveats - see below)
- **Traditional servers**: Single process handles both API and sync
- **Kubernetes**: One deployment, no sidecars needed

### Vercel Serverless Considerations

When using background sync on Vercel:

- The sync goroutine persists within a container instance
- Vercel may spin down containers during low traffic
- Sync interval should be reasonable (5-15 minutes recommended)
- Best for moderate sync loads; high-volume may need dedicated service

### High-Volume Deployments

For very high-volume sync operations, consider running the API server separately from metadata updates:

- Run API server with `ENABLE_DRIVE_SYNC=true` for new file detection
- Run `make sync-update-metadata` periodically via cron for bulk metadata updates
- This separates real-time sync from batch processing

## Monitoring and Logs

### Log Output

The sync tool outputs detailed logs:

```
[DriveSync] Starting backfill from Google Drive folder: 1a2b3c4d5e6f7g8h9i0j
[DriveSync] Found 150 files in Google Drive folder
[DriveSync] Processing file: photo1.heic (ID: 1xyz...)
[DriveSync] Downloading file from Drive: photo1.heic
[DriveSync] Converting HEIC to JPEG: photo1.heic
[DriveSync] Uploading to Firebase Storage: photo1.jpg
[DriveSync] Creating Firestore metadata: photo1.jpg
[DriveSync] Successfully synced file: photo1.jpg (Firestore ID: abc123)
[DriveSync] File already exists in Firestore: photo2.jpg
[DriveSync] Backfill complete: 50 synced, 100 skipped, 0 errors
```

### Error Handling

The sync tool:

- Logs errors but continues processing other files
- Reports summary statistics at the end
- Returns non-zero exit code on critical failures
- Handles network interruptions gracefully

## Troubleshooting

### Permission Denied

**Error**: `Failed to list files: permission denied`

**Solution**: Make sure the service account email is shared with the Drive folder with at least Viewer permissions.

### Drive API Not Enabled

**Error**: `Drive API has not been used in project`

**Solution**: Enable the Google Drive API in Google Cloud Console for your project.

### Invalid Folder ID

**Error**: `Failed to list files: not found`

**Solution**: Double-check the folder ID from the Drive URL. Ensure it's the folder ID, not a file ID.

### Conversion Failures

**Error**: `HEIC conversion failed`

**Solution**: This is a warning. The original HEIC file will be uploaded instead of JPEG. Check that goheif library is properly installed.

### Files Not Syncing

**Possible causes**:

1. File already exists in Firestore (check logs for "already exists")
2. File is not an image (only image MIME types are processed)
3. File was created before the last watch check (in watch mode)

### Rate Limit Errors (403)

**Error**: `googleapi: got HTTP response code 403 with body: ...automated queries...`

**This means**: Google has temporarily blocked your API key or IP address for making too many requests.

**Immediate actions**:

1. **Stop the sync process** - Running it repeatedly will make the ban longer
2. **Check logs** for "Detected persistent rate limiting, pausing for 5m" - the tool automatically backs off
3. **Wait it out** - Google's rate limits typically expire after 6-12 hours

**Long-term solutions**:

1. **Switch to OAuth 2.0 / Service Account** (Recommended)
   - API keys have very strict limits (10-20 requests per 100 seconds)
   - Service accounts have much higher quotas
   - Use `FIREBASE_CREDENTIALS_PATH` instead of `GOOGLE_API_KEY`
   - Share your Drive folder with the service account email

2. **Reduce sync frequency**
   - Increase `DRIVE_SYNC_INTERVAL` from `5m` to `15m` or `30m`
   - Only use `DRIVE_BACKFILL_ON_STARTUP=true` on first run

3. **Batch processing**
   - For large folders, sync in smaller batches
   - Process during off-peak hours
   - Consider manual upload to Firebase Storage as alternative

**Why this happens**:

- Google detects rapid API calls as "automated queries"
- API keys are meant for low-volume, user-facing applications
- Bulk sync operations require OAuth 2.0 credentials

## Performance Considerations

### Backfill Large Folders

For folders with thousands of images:

- Process may take considerable time
- Consider running in batches or during off-peak hours
- Monitor memory usage for very large images

### Watch Interval

Choose an appropriate interval based on:

- **Frequent uploads**: 1-5 minutes
- **Occasional uploads**: 10-30 minutes
- **Daily batches**: 1-6 hours

More frequent polling = higher API usage but faster detection.

### API Quotas

Google Drive API has quota limits:

- **Queries per day**: 1,000,000,000 (1 billion)
- **Queries per 100 seconds per user**: 1,000

For typical usage, you won't hit these limits. If you do:

- Increase watch interval
- Request quota increase in Cloud Console

## Security Best Practices

1. **Service Account Permissions**: Give the service account only the minimum required permissions (Viewer is sufficient)
2. **Folder Access**: Only share specific folders, not your entire Drive
3. **Credentials**: Never commit `firebase-service-account.json` to version control
4. **Environment Variables**: Use secure secret management for production deployments
5. **Network**: Run sync service in a secure environment
