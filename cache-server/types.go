package main

import "sync"

// Twirp request/response types for v2 cache API

type CreateCacheEntryRequest struct {
	Key     string `json:"key"`
	Version string `json:"version"`
}

type CreateCacheEntryResponse struct {
	OK              bool   `json:"ok"`
	SignedUploadURL string `json:"signed_upload_url,omitempty"`
}

type GetCacheEntryDownloadURLRequest struct {
	Key          string   `json:"key"`
	RestoreKeys  []string `json:"restore_keys,omitempty"`
	Version      string   `json:"version"`
}

type GetCacheEntryDownloadURLResponse struct {
	OK                bool   `json:"ok"`
	SignedDownloadURL string `json:"signed_download_url,omitempty"`
	MatchedKey        string `json:"matched_key,omitempty"`
}

type FinalizeCacheEntryUploadRequest struct {
	Key       string `json:"key"`
	Version   string `json:"version"`
	SizeBytes int64  `json:"size_bytes"`
}

type FinalizeCacheEntryUploadResponse struct {
	OK bool `json:"ok"`
}

// In-memory state tracking uploads in progress
type UploadState struct {
	mu     sync.Mutex
	blocks map[string][]string // key -> ordered list of temp file paths for staged blocks
}

func NewUploadState() *UploadState {
	return &UploadState{
		blocks: make(map[string][]string),
	}
}

func (u *UploadState) AddBlock(key, blockID, tempPath string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	// Store block with its ID for ordering
	u.blocks[key] = append(u.blocks[key], tempPath)
}

func (u *UploadState) GetBlocks(key string) []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	paths := u.blocks[key]
	delete(u.blocks, key)
	return paths
}
