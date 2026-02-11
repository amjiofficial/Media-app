package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// ---------------- CONFIG ----------------
const (
	bucketName   = "my-test-media-99" // <-- Replace with your actual GCP bucket
	uploadFolder = "uploads"          // Optional folder inside bucket
	dbFile       = "media.db"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

// Initialize GCS client globally for reuse
var gcsClient *storage.Client
var gcsInitError error

// Initialize GCS client at startup
func initGCS() {
	ctx := context.Background()
	var err error
	gcsClient, err = storage.NewClient(ctx)
	if err != nil {
		gcsInitError = err
		log.Printf("WARNING: Could not initialize GCS client: %v", err)
		log.Printf("         Check if GOOGLE_APPLICATION_CREDENTIALS is set")
		log.Printf("         Or run: gcloud auth application-default login")
	} else {
		log.Println("✅ Successfully initialized GCS client")
		
		// Verify bucket access
		_, err := gcsClient.Bucket(bucketName).Attrs(ctx)
		if err != nil {
			log.Printf("⚠️  WARNING: Cannot access bucket '%s': %v", bucketName, err)
			gcsInitError = err
		} else {
			log.Printf("✅ Successfully verified access to bucket: %s", bucketName)
		}
	}
}

// ---------------- MAIN ----------------
func main() {	// Initialize GCS
	initGCS()
		// ---------- SQLite Setup ----------
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS media (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		url TEXT NOT NULL,
		uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	// ---------- HTTP Routes ----------
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		showIndex(w, r, db)
	})
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, db)
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checkHealth(w, r)
	})

	// Serve static files (CSS/JS)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// ---------- Cloud Run Dynamic Port ----------
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("\n╔════════════════════════════════════════╗")
	log.Printf("║   MEDIA UPLOAD APP STARTED             ║")
	log.Printf("║   GCS Status: %v                  ║", gcsInitError == nil)
	log.Printf("║   Bucket: %s", bucketName)
	log.Printf("║   Server: http://localhost:%s      ║", port)
	log.Printf("║   Health: http://localhost:%s/health║", port)
	log.Printf("╚════════════════════════════════════════╝\n")
	
	log.Printf("Listening on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ---------------- HANDLERS ----------------
func showIndex(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query("SELECT id, filename, url, uploaded_at FROM media ORDER BY uploaded_at DESC")
	if err != nil {
		http.Error(w, "Database error", 500)
		return
	}
	defer rows.Close()

	var files []map[string]string
	for rows.Next() {
		var id int
		var filename, url, uploadedAt string
		rows.Scan(&id, &filename, &url, &uploadedAt)
		files = append(files, map[string]string{
			"ID":         fmt.Sprint(id),
			"Filename":   filename,
			"URL":        url,
			"UploadedAt": uploadedAt,
		})
	}

	if err := templates.ExecuteTemplate(w, "index.html", files); err != nil {
		http.Error(w, "Template rendering error", 500)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	log.Println("=== UPLOAD REQUEST RECEIVED ===")

	// Check if GCS is initialized
	if gcsInitError != nil {
		log.Printf("ERROR: GCS not initialized: %v", gcsInitError)
		http.Error(w, fmt.Sprintf("Cloud Storage not available: %v. Please check your Google Cloud credentials.", gcsInitError), 500)
		return
	}

	if gcsClient == nil {
		log.Printf("ERROR: GCS client is nil")
		http.Error(w, "Cloud Storage client not available", 500)
		return
	}

	// Parse uploaded file
	log.Println("Parsing multipart form...")
	r.ParseMultipartForm(10 << 20) // 10 MB max
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("ERROR: Cannot read file: %v", err)
		http.Error(w, "Cannot read file", 400)
		return
	}
	defer file.Close()

	fileName := header.Filename
	log.Printf("✅ File received: %s (size: %d bytes)", fileName, header.Size)
	
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	log.Printf("   Content-Type: %s", contentType)

	// ---------- GCP Storage Upload ----------
	ctx := context.Background()
	objectName := fmt.Sprintf("%s/%d-%s", uploadFolder, time.Now().Unix(), fileName)
	log.Printf("📤 Uploading to GCS...")
	log.Printf("   Bucket: %s", bucketName)
	log.Printf("   Path: %s", objectName)
	
	wc := gcsClient.Bucket(bucketName).Object(objectName).NewWriter(ctx)
	wc.ContentType = contentType
	
	log.Println("   Writing file to GCS...")
	bytesWritten, err := io.Copy(wc, file)
	if err != nil {
		log.Printf("❌ ERROR: Upload copy failed: %v", err)
		http.Error(w, fmt.Sprintf("Upload failed: %v", err), 500)
		return
	}
	log.Printf("   ✅ Written %d bytes", bytesWritten)
	
	log.Println("   Closing GCS writer...")
	if err := wc.Close(); err != nil {
		log.Printf("❌ ERROR: Failed to finalize upload: %v", err)
		http.Error(w, fmt.Sprintf("Failed to finalize upload: %v", err), 500)
		return
	}
	log.Println("   ✅ GCS upload completed")

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
	log.Printf("🔗 Public URL: %s", publicURL)

	// Save file info to SQLite
	log.Println("💾 Saving to database...")
	_, err = db.Exec("INSERT INTO media(filename,url) VALUES(?,?)", fileName, publicURL)
	if err != nil {
		log.Printf("❌ ERROR: Database save failed: %v", err)
		http.Error(w, fmt.Sprintf("Database save failed: %v", err), 500)
		return
	}

	log.Printf("✅✅✅ File %s successfully uploaded to GCS and saved to database!", fileName)
	log.Println("=== UPLOAD COMPLETE ===\n")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Health check endpoint
func checkHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if gcsInitError != nil {
		log.Printf("HEALTH CHECK FAILED: %v", gcsInitError)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"ERROR","message":"GCS not initialized: %v","bucket":"%s","fix":"Run: gcloud auth application-default login"}`, gcsInitError, bucketName)
		return
	}

	if gcsClient == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"ERROR","message":"GCS client is nil","bucket":"%s"}`, bucketName)
		return
	}

	// Try to access the bucket
	ctx := context.Background()
	attrs, err := gcsClient.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		log.Printf("HEALTH CHECK FAILED: Cannot access bucket %s: %v", bucketName, err)
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"ERROR","message":"Cannot access bucket: %v","bucket":"%s"}`, err, bucketName)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"OK","message":"Connected to GCS","bucket":"%s","location":"%s"}`, bucketName, attrs.Location)
}
