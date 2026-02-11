# Google Cloud Authentication Setup for Local Development

## Step 1: Authenticate with gcloud
```powershell
gcloud auth application-default login
```
This will open a browser window. Log in with your Google account.

## Step 2: Set the project
```powershell
gcloud config set project starter-app-487010
```

## Step 3: Create a service account key (optional, for local development)
```powershell
gcloud iam service-accounts create media-app-local --display-name="Media App Local"

gcloud projects add-iam-policy-binding starter-app-487010 `
  --member="serviceAccount:media-app-local@starter-app-487010.iam.gserviceaccount.com" `
  --role="roles/storage.admin"

gcloud iam service-accounts keys create C:\Users\EHSAN LAPTOP\Desktop\media-app1\sa-key.json `
  --iam-account=media-app-local@starter-app-487010.iam.gserviceaccount.com

$env:GOOGLE_APPLICATION_CREDENTIALS="C:\Users\EHSAN LAPTOP\Desktop\media-app1\sa-key.json"
```

## Step 4: Verify the bucket exists and is accessible
```powershell
gsutil ls gs://my-test-media-99
```

## Step 5: Run the Go app
```powershell
go run main.go
```

## Step 6: Test the health endpoint
Open in browser: http://localhost:8080/health

This should return: `{"status":"OK",...}`

## If still getting errors:

1. Check if bucket name matches exactly: `my-test-media-99`
2. Verify credentials are set: `$env:GOOGLE_APPLICATION_CREDENTIALS` 
3. Test manually with gsutil: `gsutil cp test.txt gs://my-test-media-99/`
