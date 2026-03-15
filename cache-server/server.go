package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Server struct {
	port    int
	owner   string
	repo    string
	storage Storage
	uploads *UploadState
	mux     *http.ServeMux
}

func NewServer(port int, owner, repo string, storage Storage) *Server {
	s := &Server{
		port:    port,
		owner:   owner,
		repo:    repo,
		storage: storage,
		uploads: NewUploadState(),
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Twirp cache service endpoints
	twirpBase := "/twirp/github.actions.results.api.v1.CacheService/"
	s.mux.HandleFunc(twirpBase+"CreateCacheEntry", s.handleCreateCacheEntry)
	s.mux.HandleFunc(twirpBase+"GetCacheEntryDownloadURL", s.handleGetCacheEntryDownloadURL)
	s.mux.HandleFunc(twirpBase+"FinalizeCacheEntryUpload", s.handleFinalizeCacheEntryUpload)

	// Azure Blob-compatible endpoints
	s.mux.HandleFunc("/upload/", s.handleBlobUpload)
	s.mux.HandleFunc("/download/", s.handleBlobDownload)

	// Health check
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

// objectPath returns the GCS object path for a given version and key.
func (s *Server) objectPath(version, key string) string {
	return fmt.Sprintf("%s/%s/%s/%s", s.owner, s.repo, version, key)
}

// objectPrefix returns the GCS prefix for prefix matching.
func (s *Server) objectPrefix(version, keyPrefix string) string {
	return fmt.Sprintf("%s/%s/%s/%s", s.owner, s.repo, version, keyPrefix)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.String())

	// Strip _apis/artifactcache prefix if present (v2 API routing)
	if strings.HasPrefix(r.URL.Path, "/_apis/artifactcache/") {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/_apis/artifactcache")
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	log.Printf("Cache server listening on %s (owner=%s, repo=%s)", addr, s.owner, s.repo)
	return http.ListenAndServe(addr, s)
}
