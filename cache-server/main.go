package main

import (
	"context"
	"flag"
	"log"
)

func main() {
	bucket := flag.String("bucket", "", "GCS bucket for cache storage (required)")
	owner := flag.String("owner", "", "Repository owner (required)")
	repo := flag.String("repo", "", "Repository name (required)")
	port := flag.Int("port", 8787, "Port to listen on")
	flag.Parse()

	if *bucket == "" || *owner == "" || *repo == "" {
		log.Fatal("Required flags: -bucket, -owner, -repo")
	}

	ctx := context.Background()
	gcs, err := NewGCSStorage(ctx, *bucket)
	if err != nil {
		log.Fatalf("Failed to create GCS storage: %v", err)
	}

	server := NewServer(*port, *owner, *repo, gcs)
	log.Fatal(server.ListenAndServe())
}
