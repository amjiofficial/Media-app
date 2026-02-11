# Media Upload App - Complete Setup & Troubleshooting Guide

## Problem: Images Not Uploading to Google Cloud Storage

The issue is likely **Google Cloud authentication is not configured on your local machine**.

---

## Solution: Setup Google Cloud Credentials

### Step 1: Authenticate with gcloud (REQUIRED)

Run this command in PowerShell:

```powershell
gcloud auth application-default login
```

This will:
- Open a browser window
- Ask you to sign in with your Google account
- Grant the necessary permissions
- Save credentials locally to: `%APPDATA%\gcloud\application_default_credentials.json`

### Step 2: Set your project

```powershell
gcloud config set project starter-app-487010
```

### Step 3: Verify bucket exists and is accessible

```powershell
gsutil ls -b gs://my-test-media-99
```

You should see output like:
```
gs://my-test-media-99/
```

### Step 4: Test upload manually with gsutil

```powershell
echo "test" | gsutil cp - gs://my-test-media-99/test.txt
gsutil ls gs://my-test-media-99/test.txt
```

If this works, then credentials are properly configured.

### Step 5: Restart the Go app

After setting up credentials, restart the server:

```powershell
# Kill any running instances
Get-Process | Where-Object {$_.Name -like "*go*"} | Stop-Process -Force

# Start fresh
cd "C:\Users\EHSAN LAPTOP\Desktop\media-app1"
go run main.go
```

### Step 6: Check server status

Visit: http://localhost:8080/health

Expected response:
```json
{"status":"OK","message":"Connected to GCS","bucket":"my-test-media-99","location":"..."}
```

If you see an error like:
```json
{"status":"ERROR","message":"GCS not initialized: ..."}
```

Then credentials are still not set up. Return to Step 1.

---

## Alternative: Use Service Account Key (Advanced)

If `gcloud auth application-default login` doesn't work, create a service account:

### Create Service Account in GCP Console

1. Go to: https://console.cloud.google.com/iam-admin/serviceaccounts
2. Click "Create Service Account"
3. Name: `media-app-local`
4. Grant role: `Storage Admin`
5. Create a JSON key file
6. Download and save it locally

### Set credentials environment variable

```powershell
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\path\to\service-account-key.json"

# Then restart the app
go run main.go
```

---

## How the App Works

1. **On Startup**: The app tries to initialize a GCS client using Application Default Credentials
2. **Health Check** (`/health`): Verifies GCS connectivity and bucket access
3. **Upload** (`/upload`): 
   - Receives file from form
   - Uploads to GCS bucket at: `gs://my-test-media-99/uploads/TIMESTAMP-FILENAME`
   - Saves file info to SQLite database
   - Redirects back to homepage showing the upload

---

## Key Files

- `main.go` - Server logic with improved GCS initialization
- `Dockerfile` - For Cloud Run deployment  
- `templates/index.html` - Upload form UI
- `static/style.css` - Styling
- `media.db` - Local SQLite database

---

## Debugging

### Check logs while app is running

The server prints detailed logs:
- `INFO:` - Successful operations
- `WARNING:` - Issues that don't stop execution
- `ERROR:` - Problems that cause failure

Example:
```
✅ Successfully initialized GCS client
✅ Successfully verified access to bucket: my-test-media-99
Server running at http://localhost:8080
```

### Enable verbose error messages

Visit http://localhost:8080/health to see detailed error information if something is wrong.

---

## Common Errors & Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| `GCS not initialized` | No Google Cloud credentials | Run `gcloud auth application-default login` |
| `Cannot access bucket` | Wrong bucket name or no permissions | Verify bucket name and run `gsutil ls gs://my-test-media-99` |
| `Upload failed` | Network issue or bucket error | Check CloudStorage logs in GCP Console |
| `Port 8080 already in use` | Another app using the port | Change PORT: `$env:PORT=3000; go run main.go` |

---

## Testing Upload Manually

1. Go to http://localhost:8080
2. Click "Choose File" and select an image
3. Click "Upload"
4. You should see the image listed below
5. Click the link to verify it opens from GCS

If it doesn't work:
1. Check http://localhost:8080/health
2. Check the terminal logs for error messages
3. Verify credentials with: `gcloud auth list`

---

## For Production (Cloud Run)

The app is ready to deploy to Cloud Run:

```bash
gcloud run deploy media-app \
  --source . \
  --region asia-south1 \
  --allow-unauthenticated
```

On Cloud Run:
- Authentication is automatic (uses service account)
- No need for local credentials
- Data persists in SQLite (but lost on container restart, so use Cloud SQL or Firestore for production)
