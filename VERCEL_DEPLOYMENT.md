# Vercel Deployment Guide

## Setup Firebase Credentials on Vercel

### Step 1: Get your Firebase service account JSON
1. Go to Firebase Console → Project Settings → Service Accounts
2. Click "Generate New Private Key"
3. Download the JSON file

### Step 2: Add to Vercel as Environment Variable

#### Option A: Using Vercel Dashboard (Recommended)
1. Go to your project on Vercel
2. Navigate to Settings → Environment Variables
3. Add a new variable:
   - **Name**: `FIREBASE_CREDENTIALS_JSON`
   - **Value**: Copy and paste the **entire contents** of your `firebase-service-account.json` file
   - Select environments: Production, Preview, Development

#### Option B: Using Vercel CLI
```bash
# Copy the JSON content first, then:
vercel env add FIREBASE_CREDENTIALS_JSON
# Paste the entire JSON when prompted
# Select: Production, Preview, Development
```

### Step 3: Add Other Environment Variables

Add these in Vercel Dashboard or via CLI:

```bash
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_BUCKET_NAME=your-bucket-name.appspot.com
FIRESTORE_COLLECTION=images
ALLOWED_ORIGINS=https://your-frontend.vercel.app,https://yourdomain.com
CACHE_TTL=12h
CACHE_CLEANUP_INTERVAL=10m
```

## Deployment

### Deploy to Vercel
```bash
# Install Vercel CLI
npm i -g vercel

# Login
vercel login

# Deploy (preview)
vercel

# Deploy to production
vercel --prod
```

## Local vs Vercel

### Local Development
- Uses `cmd/server/main.go`
- Reads from `.env` file
- Uses `FIREBASE_CREDENTIALS_PATH=firebase-service-account.json`
- Run with: `go run cmd/server/main.go`

### Vercel (Serverless)
- Uses `api/index.go`
- Reads from environment variables
- Uses `FIREBASE_CREDENTIALS_JSON` (full JSON content)
- Deployed via `vercel` command

## File Structure

```
.
├── api/
│   └── index.go           # Vercel serverless entry point
├── cmd/
│   └── server/
│       └── main.go        # Local development server
├── internal/              # Shared application code
├── vercel.json            # Vercel configuration
└── .vercelignore         # Files to exclude from deployment
```

## Testing Your Deployment

After deployment:
```bash
# Health check
curl https://your-project.vercel.app/health

# Get image
curl https://your-project.vercel.app/image?fileName=test.jpg

# List images
curl https://your-project.vercel.app/images/list?limit=10&page=1
```

## Troubleshooting

### Check logs
```bash
vercel logs
```

### Common issues:
1. **500 Error**: Check if `FIREBASE_CREDENTIALS_JSON` is set correctly
2. **Authentication failed**: Ensure the JSON is complete and valid
3. **CORS errors**: Add your frontend domain to `ALLOWED_ORIGINS`
