# Quick Start: Google Drive Sync

Get started syncing photos from Google Drive to Firebase in 5 minutes.

## Prerequisites

- ✅ Go 1.25+ installed
- ✅ Firebase project set up
- ✅ Firebase service account JSON file
- ✅ Google Drive folder with photos

## Step 1: Choose Authentication Method

### Option A: Google API Key (Easier)

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Select your Firebase project
3. Enable Google Drive API:
   - Go to **APIs & Services** → **Library**
   - Search for "Google Drive API" → **Enable**
4. Create API Key:
   - Go to **APIs & Services** → **Credentials**
   - Click **Create Credentials** → **API Key**
   - Copy the key
   - (Optional) Restrict to Drive API only

### Option B: Firebase Service Account

Use your existing Firebase credentials (no extra setup needed).

## Step 2: Prepare Your Drive Folder

### If using API Key:

1. Open your Google Drive folder with photos
2. Right-click → **Share** → **Get link**
3. Set to "Anyone with the link can view"
4. Copy the folder ID from URL:
   - URL: `https://drive.google.com/drive/folders/1a2b3c4d5e6f7g8h9i0j`
   - Folder ID: `1a2b3c4d5e6f7g8h9i0j`

### If using Service Account:

1. Open your Google Drive folder with photos
2. Click **Share**
3. Add your Firebase service account email:
   - Find in `firebase-service-account.json` → `client_email`
   - Example: `firebase-adminsdk-xxxxx@your-project.iam.gserviceaccount.com`
4. Give **Viewer** permissions
5. Copy the folder ID from URL

## Step 3: Configure Environment

Add to your `.env` file:

### If using API Key:

```env
# Google Drive Sync
ENABLE_DRIVE_SYNC=true
DRIVE_SYNC_INTERVAL=5m
DRIVE_BACKFILL_ON_STARTUP=true  # Enable for first run
GOOGLE_DRIVE_FOLDER_ID=1a2b3c4d5e6f7g8h9i0j
GOOGLE_API_KEY=AIzaSyXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

# Firebase (for Storage and Firestore)
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_BUCKET_NAME=your-bucket-name
FIREBASE_CREDENTIALS_PATH=firebase-service-account.json
FIRESTORE_COLLECTION=images
```

### If using Service Account:

```env
# Google Drive Sync
ENABLE_DRIVE_SYNC=true
DRIVE_SYNC_INTERVAL=5m
DRIVE_BACKFILL_ON_STARTUP=true  # Enable for first run
GOOGLE_DRIVE_FOLDER_ID=1a2b3c4d5e6f7g8h9i0j

# Firebase (used for both Drive and Firebase services)
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_BUCKET_NAME=your-bucket-name
FIREBASE_CREDENTIALS_PATH=firebase-service-account.json
FIRESTORE_COLLECTION=images
```

## Step 4: Start Server with Sync

```bash
make run
```

You'll see output like:

```
🔄 Initializing Google Drive background sync...
📦 Running one-time backfill from Google Drive...
[DriveSync] Found 25 files in Google Drive folder
[DriveSync] Processing file: photo1.heic
[DriveSync] Converting HEIC to JPEG: photo1.heic
[DriveSync] Successfully synced file: photo1.jpg
...
✅ Backfill completed successfully
🚀 Starting Drive watch (interval: 5m0s)
Server starting on port 8080
```

**What's happening:**

1. ✅ Server initializes all services
2. ✅ Backfill syncs all existing Drive photos
3. ✅ Watch mode starts checking every 5 minutes for new files
4. ✅ API server is ready to serve requests

## Step 5: Verify

Check that your images are synced:

1. **Firebase Console**: Go to Storage and verify files are uploaded
2. **Firestore Console**: Go to Firestore and check the `images` collection
3. **API**: Test the API endpoint:
   ```bash
   curl http://localhost:8080/images/list?limit=10
   curl http://localhost:8080/image?fileName=photo1.jpg
   ```

## Step 6: Disable Backfill (After First Run)

After the initial sync completes, update your `.env`:

```env
DRIVE_BACKFILL_ON_STARTUP=false  # Disable for faster startups
```

Now the server will:

- ✅ Start instantly (no backfill delay)
- ✅ Only sync new files added to Drive
- ✅ Check every 5 minutes for updates

## What Happens During Sync?

1. ✅ Lists all images in your Drive folder
2. ✅ Checks which files already exist in Firestore (skips duplicates)
3. ✅ Downloads new images from Drive
4. ✅ Converts HEIC/HEIF to JPEG automatically
5. ✅ Uploads to Firebase Storage
6. ✅ Extracts GPS data from EXIF
7. ✅ Reverse geocodes to "City, Country" format
8. ✅ Creates metadata in Firestore
9. ✅ Images are now available via the API

## Common Issues

### "Permission denied" error

- ❌ **Problem**: Service account doesn't have access to folder
- ✅ **Solution**: Make sure you shared the folder with the service account email

### "Drive API has not been used"

- ❌ **Problem**: Drive API not enabled
- ✅ **Solution**: Enable it in Google Cloud Console → APIs & Services → Library

### "Folder not found"

- ❌ **Problem**: Wrong folder ID
- ✅ **Solution**: Double-check the ID from the Drive URL

### HEIC conversion fails

- ⚠️ **Warning**: Original HEIC will be uploaded instead
- ✅ **Not critical**: The API can still serve HEIC files

### Rate limit errors (403)

- ⚠️ **Problem**: Too many Drive API requests - Google has temporarily blocked your API key/IP
- ✅ **Automatic handling**: Tool detects persistent 403 errors and pauses for 5 minutes
- ⏰ **Wait it out**: If errors persist, stop the process and wait 6-12 hours
- 🔑 **Long-term fix**: Switch from API Key to Service Account credentials (much higher quotas)
- 📊 **Performance**:
  - Normal: ~20 files/minute with automatic retry
  - Rate limited: Pauses automatically, resumes after cooldown
  - API Keys: Limited to 10-20 requests per 100 seconds
  - Service Accounts: Much higher limits, better for bulk operations

## Example: Complete Setup

```bash
# 1. Clone and setup
git clone <your-repo>
cd trekka-api

# 2. Install dependencies
make install-deps

# 3. Configure environment
cp .env.example .env
# Edit .env with your settings:
#   - Set ENABLE_DRIVE_SYNC=true
#   - Set DRIVE_BACKFILL_ON_STARTUP=true
#   - Add your Google Drive folder ID
#   - Add Google API Key or use service account

# 4. Add service account credentials
# Place firebase-service-account.json in project root

# 5. Start server (backfill runs automatically)
make run

# 6. Wait for backfill to complete, then test
curl http://localhost:8080/images/list
curl http://localhost:8080/image?fileName=photo1.jpg

# 7. After first run, disable backfill in .env
#    Set DRIVE_BACKFILL_ON_STARTUP=false
```

## Configuration Options

```env
# Enable background sync
ENABLE_DRIVE_SYNC=true

# How often to check Drive for new files
DRIVE_SYNC_INTERVAL=5m  # Options: 5m, 10m, 30m, 1h

# Run backfill on startup (only enable for first run or re-sync)
DRIVE_BACKFILL_ON_STARTUP=true
```

## Advanced: Update Metadata Only

If you already have files in Firestore but they're missing GPS data:

```bash
# Preview what would be updated
make sync-update-metadata-dry-run

# Update all files missing GPS data
make sync-update-metadata
```

This extracts GPS from existing files in Firebase Storage and adds reverse geocoded location data.

## Next Steps

- 📖 Read [GOOGLE_DRIVE_SYNC.md](GOOGLE_DRIVE_SYNC.md) for detailed documentation
- 🚀 Deploy to Vercel or Docker with background sync enabled
- 🔍 Monitor server logs for sync activity
- ⚙️ Adjust `DRIVE_SYNC_INTERVAL` based on your upload frequency

## Deployment

The background sync works seamlessly in all environments:

**Docker:**

```bash
docker build -t trekka-api .
docker run -p 8080:8080 --env-file .env trekka-api
```

**Vercel:**

- Set environment variables in Vercel dashboard
- Deploy normally - sync runs in serverless functions

**Traditional Server:**

- Run with systemd or process manager
- Single process handles both API and sync

---

**Questions?** See the full documentation in [GOOGLE_DRIVE_SYNC.md](GOOGLE_DRIVE_SYNC.md)
