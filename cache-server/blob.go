package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// handleBlobUpload handles Azure Blob-compatible upload requests.
// PUT /upload/{version}/{key}?comp=block&blockid=...  → stage a block
// PUT /upload/{version}/{key}?comp=blocklist           → commit all blocks to GCS
func (s *Server) handleBlobUpload(w http.ResponseWriter, r *http.Request) {
	// Parse version/key from path: /upload/{version}/{key}
	version, key := s.parseVersionKey(r.URL.Path, "/upload/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	comp := r.URL.Query().Get("comp")

	switch comp {
	case "block":
		s.handleStageBlock(w, r, version, key)
	case "blocklist":
		s.handleCommitBlockList(w, r, version, key)
	default:
		// Direct PUT without block staging — single blob upload
		s.handleDirectUpload(w, r, version, key)
	}
}

// handleStageBlock buffers a single block to a temp file.
func (s *Server) handleStageBlock(w http.ResponseWriter, r *http.Request, version, key string) {
	blockID := r.URL.Query().Get("blockid")
	log.Printf("StageBlock: key=%s blockid=%s", key, blockID)

	tmpFile, err := os.CreateTemp("", "cache-block-*")
	if err != nil {
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(tmpFile, r.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		http.Error(w, "failed to write block", http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	cacheKey := version + "/" + key
	s.uploads.AddBlock(cacheKey, blockID, tmpFile.Name())

	w.WriteHeader(http.StatusCreated)
}

// handleCommitBlockList assembles staged blocks and streams to GCS.
func (s *Server) handleCommitBlockList(w http.ResponseWriter, r *http.Request, version, key string) {
	log.Printf("CommitBlockList: key=%s", key)

	cacheKey := version + "/" + key
	blockPaths := s.uploads.GetBlocks(cacheKey)
	if len(blockPaths) == 0 {
		http.Error(w, "no blocks staged", http.StatusBadRequest)
		return
	}

	// Clean up temp files when done
	defer func() {
		for _, p := range blockPaths {
			os.Remove(p)
		}
	}()

	// Create a multi-reader that concatenates all block files
	readers := make([]io.Reader, 0, len(blockPaths))
	closers := make([]io.Closer, 0, len(blockPaths))
	for _, p := range blockPaths {
		f, err := os.Open(p)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to open block: %v", err), http.StatusInternalServerError)
			// Close any already opened
			for _, c := range closers {
				c.Close()
			}
			return
		}
		readers = append(readers, f)
		closers = append(closers, f)
	}
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()

	objectPath := s.objectPath(version, key)
	if err := s.storage.Write(r.Context(), objectPath, io.MultiReader(readers...)); err != nil {
		log.Printf("Failed to write to GCS: %v", err)
		http.Error(w, "storage write failed", http.StatusInternalServerError)
		return
	}

	log.Printf("Committed cache: %s (%d blocks)", objectPath, len(blockPaths))
	w.WriteHeader(http.StatusCreated)
}

// handleDirectUpload handles a single PUT without block staging.
func (s *Server) handleDirectUpload(w http.ResponseWriter, r *http.Request, version, key string) {
	log.Printf("DirectUpload: key=%s", key)

	objectPath := s.objectPath(version, key)
	if err := s.storage.Write(r.Context(), objectPath, r.Body); err != nil {
		log.Printf("Failed to write to GCS: %v", err)
		http.Error(w, "storage write failed", http.StatusInternalServerError)
		return
	}

	log.Printf("Uploaded cache: %s", objectPath)
	w.WriteHeader(http.StatusCreated)
}

// handleBlobDownload streams a cached object from GCS.
func (s *Server) handleBlobDownload(w http.ResponseWriter, r *http.Request) {
	version, key := s.parseVersionKey(r.URL.Path, "/download/")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	objectPath := s.objectPath(version, key)

	if r.Method == http.MethodHead {
		exists, size, err := s.storage.Exists(r.Context(), objectPath)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
		w.WriteHeader(http.StatusOK)
		return
	}

	reader, err := s.storage.Read(r.Context(), objectPath)
	if err != nil {
		log.Printf("Failed to read from GCS: %v", err)
		http.NotFound(w, r)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, reader)
}

// parseVersionKey extracts version and key from a URL path after the given prefix.
// Path format: {prefix}{version}/{key} where key can contain slashes.
func (s *Server) parseVersionKey(path, prefix string) (string, string) {
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
