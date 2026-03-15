package main

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// Storage is the interface for cache blob storage.
type Storage interface {
	Write(ctx context.Context, path string, r io.Reader) error
	Read(ctx context.Context, path string) (io.ReadCloser, error)
	Exists(ctx context.Context, path string) (bool, int64, error) // exists, size, err
	ListPrefix(ctx context.Context, prefix string) ([]string, error)
}

// GCSStorage implements Storage backed by Google Cloud Storage.
type GCSStorage struct {
	client *storage.Client
	bucket string
}

func NewGCSStorage(ctx context.Context, bucket string) (*GCSStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	return &GCSStorage{client: client, bucket: bucket}, nil
}

func (g *GCSStorage) Write(ctx context.Context, path string, r io.Reader) error {
	w := g.client.Bucket(g.bucket).Object(path).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return fmt.Errorf("write to GCS: %w", err)
	}
	return w.Close()
}

func (g *GCSStorage) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return g.client.Bucket(g.bucket).Object(path).NewReader(ctx)
}

func (g *GCSStorage) Exists(ctx context.Context, path string) (bool, int64, error) {
	attrs, err := g.client.Bucket(g.bucket).Object(path).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, attrs.Size, nil
}

func (g *GCSStorage) ListPrefix(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		keys = append(keys, attrs.Name)
	}
	return keys, nil
}
