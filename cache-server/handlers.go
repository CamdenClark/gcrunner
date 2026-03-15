package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// handleCreateCacheEntry handles the CreateCacheEntry Twirp method.
func (s *Server) handleCreateCacheEntry(w http.ResponseWriter, r *http.Request) {
	var req CreateCacheEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"msg": "invalid request"})
		return
	}

	log.Printf("CreateCacheEntry: key=%s version=%s", req.Key, req.Version)

	uploadURL := fmt.Sprintf("http://localhost:%d/upload/%s/%s", s.port, req.Version, req.Key)

	writeJSON(w, http.StatusOK, CreateCacheEntryResponse{
		OK:              true,
		SignedUploadURL: uploadURL,
	})
}

// handleGetCacheEntryDownloadURL handles the GetCacheEntryDownloadURL Twirp method.
func (s *Server) handleGetCacheEntryDownloadURL(w http.ResponseWriter, r *http.Request) {
	var req GetCacheEntryDownloadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"msg": "invalid request"})
		return
	}

	log.Printf("GetCacheEntryDownloadURL: key=%s version=%s restore_keys=%v", req.Key, req.Version, req.RestoreKeys)

	ctx := r.Context()
	objectPath := s.objectPath(req.Version, req.Key)

	// Try exact match first
	exists, _, err := s.storage.Exists(ctx, objectPath)
	if err != nil {
		log.Printf("Error checking cache existence: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"msg": "storage error"})
		return
	}
	if exists {
		downloadURL := fmt.Sprintf("http://localhost:%d/download/%s/%s", s.port, req.Version, req.Key)
		writeJSON(w, http.StatusOK, GetCacheEntryDownloadURLResponse{
			OK:                true,
			SignedDownloadURL: downloadURL,
			MatchedKey:        req.Key,
		})
		return
	}

	// Try restore key prefix matching
	for _, restoreKey := range req.RestoreKeys {
		prefix := s.objectPrefix(req.Version, restoreKey)
		keys, err := s.storage.ListPrefix(ctx, prefix)
		if err != nil {
			log.Printf("Error listing prefix %s: %v", prefix, err)
			continue
		}
		if len(keys) > 0 {
			// Use the last (most recently written) match
			matched := keys[len(keys)-1]
			// Extract the cache key from the object path: {owner}/{repo}/{version}/{key}
			parts := strings.SplitN(matched, "/", 4)
			if len(parts) == 4 {
				matchedKey := parts[3]
				downloadURL := fmt.Sprintf("http://localhost:%d/download/%s/%s", s.port, req.Version, matchedKey)
				writeJSON(w, http.StatusOK, GetCacheEntryDownloadURLResponse{
					OK:                true,
					SignedDownloadURL: downloadURL,
					MatchedKey:        matchedKey,
				})
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, GetCacheEntryDownloadURLResponse{OK: false})
}

// handleFinalizeCacheEntryUpload handles the FinalizeCacheEntryUpload Twirp method.
func (s *Server) handleFinalizeCacheEntryUpload(w http.ResponseWriter, r *http.Request) {
	var req FinalizeCacheEntryUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"msg": "invalid request"})
		return
	}

	log.Printf("FinalizeCacheEntryUpload: key=%s version=%s size=%d", req.Key, req.Version, req.SizeBytes)

	writeJSON(w, http.StatusOK, FinalizeCacheEntryUploadResponse{OK: true})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
