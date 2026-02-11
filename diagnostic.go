package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/storage"
)

func main() {
	fmt.Println("=== GCS Diagnostic Tool ===\n")

	ctx := context.Background()

	fmt.Println("1. Attempting to create GCS client with default credentials...")
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("❌ ERROR: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("- Run: gcloud auth application-default login")
		fmt.Println("- Or set: $env:GOOGLE_APPLICATION_CREDENTIALS='path/to/key.json'")
		return
	}
	defer client.Close()
	fmt.Println("✅ Successfully created GCS client\n")

	bucketName := "my-test-media-99"
	fmt.Printf("2. Attempting to access bucket: %s\n", bucketName)

	bucket := client.Bucket(bucketName)
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		log.Printf("❌ ERROR accessing bucket: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("- Verify bucket exists: gsutil ls -b gs://my-test-media-99")
		fmt.Println("- Check permissions: gsutil iam get gs://my-test-media-99")
		return
	}
	fmt.Printf("✅ Successfully accessed bucket\n")
	fmt.Printf("   Location: %s\n", attrs.Location)
	fmt.Printf("   StorageClass: %s\n", attrs.StorageClass)
	fmt.Printf("   Created: %v\n\n", attrs.Created)

	fmt.Println("3. Attempting to write test file...")
	obj := bucket.Object("test-diagnostic.txt")
	wc := obj.NewWriter(ctx)
	wc.ContentType = "text/plain"

	if _, err := wc.Write([]byte("Test upload from diagnostic tool")); err != nil {
		log.Printf("❌ ERROR writing file: %v\n", err)
		return
	}

	if err := wc.Close(); err != nil {
		log.Printf("❌ ERROR finalizing upload: %v\n", err)
		return
	}
	fmt.Println("✅ Successfully uploaded test file\n")

	fmt.Println("4. Attempting to list bucket contents...")
	query := &storage.Query{Prefix: ""}
	it := bucket.Objects(ctx, query)

	count := 0
	for {
		attrs, err := it.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			log.Printf("❌ ERROR listing objects: %v\n", err)
			return
		}
		count++
		if count <= 5 {
			fmt.Printf("   - %s (size: %d bytes)\n", attrs.Name, attrs.Size)
		}
	}
	fmt.Printf("✅ Found %d objects in bucket\n", count)

	fmt.Println("\n=== All diagnostics passed! ===")
	fmt.Println("Your app should be able to upload to GCS")
}
