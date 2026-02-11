# Media App

A simple Go web application to upload media files to Google Cloud Storage and track uploads in a local SQLite database.

## Features

- Web UI for uploading files
- Uploads files to a GCS bucket (configurable)
- Stores file metadata (filename, public URL, timestamp) in a local SQLite database
- Health endpoint to verify GCS connectivity
- Dockerfile for containerized deployment (Cloud Run)

## File layout

- `main.go` — application server and handlers
- `Dockerfile` — multi-stage build for production
- `templates/index.html` — upload page
- `static/style.css` — simple styling
- `media.db` — local SQLite DB (created at runtime)
- `.dockerignore` / `.gcloudignore` — excludes local artifacts from builds

## Prerequisites

- Go (locally, for development)
- Git
- Google Cloud SDK (`gcloud`, `gsutil`) and access to the target GCP project
- A Google Cloud Storage bucket (the app uses `my-test-media-99` by default)

## Configuration

Edit `main.go` to change the bucket name if needed:

```go
const (
	bucketName   = "my-test-media-99" // replace with your bucket
	uploadFolder = "uploads"
	dbFile       = "media.db"
)
```

## Local development

1. Authenticate with Google Cloud Application Default Credentials:

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

2. (Optional) Create a service account key for local development and set `GOOGLE_APPLICATION_CREDENTIALS`.

3. Run the server:

```bash
go run main.go
```

4. Open the app: `http://localhost:8080`

5. Health check: `http://localhost:8080/health` — returns JSON describing connectivity to the GCS bucket.

## Upload flow

- Use the web UI to select a file and submit.
- The server uploads the file to `gs://<bucket>/<uploadFolder>/<timestamp>-<filename>` and stores a public URL in the SQLite DB.

## Notes & Caveats

- On Cloud Run, the app runs with the service account attached to the service; you do not need to set local ADC there.
- `media.db` is a local file and is not persistent across container restarts. For production use, migrate metadata storage to Cloud SQL or Firestore.
- Ensure the Cloud Run service account (or the machine you run locally) has `roles/storage.objectAdmin` or appropriate permissions on the bucket.

## Docker / Cloud Run

To deploy from source using Cloud Run (Cloud Build will be used to build the image):

```bash
gcloud services enable run.googleapis.com cloudbuild.googleapis.com storage.googleapis.com
gcloud run deploy media-app --source . --region asia-south1 --allow-unauthenticated
```

If you prefer to build and push the image manually:

```bash
# Build and push image
gcloud builds submit --tag gcr.io/PROJECT_ID/media-app .
# Deploy using the image
gcloud run deploy media-app --image gcr.io/PROJECT_ID/media-app --region asia-south1 --allow-unauthenticated
```

## Troubleshooting

- `Upload failed` errors: check server logs — missing credentials or permission errors are common. Run `gcloud auth application-default login` locally or grant roles to the service account on GCP.
- `Bucket not found`: verify bucket name and that it exists with `gsutil ls gs://<bucket>`.
- Port already in use: if `listen tcp :8080: bind: Only one usage of each socket address` appears, stop the process using port 8080 or set `PORT` env var before running.

## Committing README and pushing

After reviewing the README, commit and push it to your repo:

```bash
git add README.md
git commit -m "Add project README"
git push origin main
```

If the remote rejects because it contains remote changes, integrate them first:

```bash
git fetch origin
git pull --rebase origin main
# resolve conflicts if any
git push origin main
```

## License & Contact

MIT License — feel free to reuse and adapt. For questions contact the repository owner.
