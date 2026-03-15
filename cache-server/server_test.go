package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// MockStorage implements Storage for testing.
type MockStorage struct {
	objects map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{objects: make(map[string][]byte)}
}

func (m *MockStorage) Write(_ context.Context, path string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.objects[path] = data
	return nil
}

func (m *MockStorage) Read(_ context.Context, path string) (io.ReadCloser, error) {
	data, ok := m.objects[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *MockStorage) Exists(_ context.Context, path string) (bool, int64, error) {
	data, ok := m.objects[path]
	if !ok {
		return false, 0, nil
	}
	return true, int64(len(data)), nil
}

func (m *MockStorage) ListPrefix(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func newTestServer() (*Server, *MockStorage) {
	store := NewMockStorage()
	srv := NewServer(8787, "testowner", "testrepo", store)
	return srv, store
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := newTestServer()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected 'ok', got %q", w.Body.String())
	}
}

func TestCreateCacheEntry(t *testing.T) {
	srv, _ := newTestServer()

	body, _ := json.Marshal(CreateCacheEntryRequest{Key: "my-cache-key", Version: "abc123"})
	req := httptest.NewRequest("POST", "/twirp/github.actions.results.api.v1.CacheService/CreateCacheEntry", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp CreateCacheEntryResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.OK {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(resp.SignedUploadURL, "/upload/abc123/my-cache-key") {
		t.Fatalf("unexpected upload URL: %s", resp.SignedUploadURL)
	}
}

func TestGetCacheEntryDownloadURL_NotFound(t *testing.T) {
	srv, _ := newTestServer()

	body, _ := json.Marshal(GetCacheEntryDownloadURLRequest{Key: "nonexistent", Version: "v1"})
	req := httptest.NewRequest("POST", "/twirp/github.actions.results.api.v1.CacheService/GetCacheEntryDownloadURL", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp GetCacheEntryDownloadURLResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.OK {
		t.Fatal("expected ok=false for nonexistent key")
	}
}

func TestGetCacheEntryDownloadURL_ExactMatch(t *testing.T) {
	srv, store := newTestServer()

	// Pre-populate storage
	store.objects["testowner/testrepo/v1/my-key"] = []byte("cached data")

	body, _ := json.Marshal(GetCacheEntryDownloadURLRequest{Key: "my-key", Version: "v1"})
	req := httptest.NewRequest("POST", "/twirp/github.actions.results.api.v1.CacheService/GetCacheEntryDownloadURL", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp GetCacheEntryDownloadURLResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.OK {
		t.Fatal("expected ok=true")
	}
	if resp.MatchedKey != "my-key" {
		t.Fatalf("expected matched_key=my-key, got %s", resp.MatchedKey)
	}
	if !strings.Contains(resp.SignedDownloadURL, "/download/v1/my-key") {
		t.Fatalf("unexpected download URL: %s", resp.SignedDownloadURL)
	}
}

func TestGetCacheEntryDownloadURL_PrefixMatch(t *testing.T) {
	srv, store := newTestServer()

	store.objects["testowner/testrepo/v1/node-modules-abc123"] = []byte("cached data")

	body, _ := json.Marshal(GetCacheEntryDownloadURLRequest{
		Key:         "node-modules-def456",
		Version:     "v1",
		RestoreKeys: []string{"node-modules-"},
	})
	req := httptest.NewRequest("POST", "/twirp/github.actions.results.api.v1.CacheService/GetCacheEntryDownloadURL", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp GetCacheEntryDownloadURLResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.OK {
		t.Fatal("expected ok=true for prefix match")
	}
	if resp.MatchedKey != "node-modules-abc123" {
		t.Fatalf("expected matched key node-modules-abc123, got %s", resp.MatchedKey)
	}
}

func TestFinalizeCacheEntryUpload(t *testing.T) {
	srv, _ := newTestServer()

	body, _ := json.Marshal(FinalizeCacheEntryUploadRequest{Key: "k", Version: "v", SizeBytes: 100})
	req := httptest.NewRequest("POST", "/twirp/github.actions.results.api.v1.CacheService/FinalizeCacheEntryUpload", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp FinalizeCacheEntryUploadResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
}

func TestDirectUploadAndDownload(t *testing.T) {
	srv, store := newTestServer()

	// Upload
	data := "hello world cache"
	req := httptest.NewRequest("PUT", "/upload/v1/my-key", strings.NewReader(data))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d", w.Code)
	}

	// Verify in storage
	if string(store.objects["testowner/testrepo/v1/my-key"]) != data {
		t.Fatal("data not in storage")
	}

	// Download
	req = httptest.NewRequest("GET", "/download/v1/my-key", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download: expected 200, got %d", w.Code)
	}
	if w.Body.String() != data {
		t.Fatalf("expected %q, got %q", data, w.Body.String())
	}
}

func TestHeadDownload(t *testing.T) {
	srv, store := newTestServer()

	store.objects["testowner/testrepo/v1/my-key"] = []byte("12345")

	req := httptest.NewRequest("HEAD", "/download/v1/my-key", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Length") != "5" {
		t.Fatalf("expected Content-Length=5, got %s", w.Header().Get("Content-Length"))
	}
}

func TestBlockUpload(t *testing.T) {
	srv, store := newTestServer()

	// Stage block 1
	req := httptest.NewRequest("PUT", "/upload/v1/my-key?comp=block&blockid=block1", strings.NewReader("hello "))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("stage block 1: expected 201, got %d", w.Code)
	}

	// Stage block 2
	req = httptest.NewRequest("PUT", "/upload/v1/my-key?comp=block&blockid=block2", strings.NewReader("world"))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("stage block 2: expected 201, got %d", w.Code)
	}

	// Commit
	req = httptest.NewRequest("PUT", "/upload/v1/my-key?comp=blocklist", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("commit: expected 201, got %d", w.Code)
	}

	// Verify assembled data in storage
	if string(store.objects["testowner/testrepo/v1/my-key"]) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(store.objects["testowner/testrepo/v1/my-key"]))
	}
}

func TestDownloadNotFound(t *testing.T) {
	srv, _ := newTestServer()

	req := httptest.NewRequest("GET", "/download/v1/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
