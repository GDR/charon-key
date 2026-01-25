package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry represents a cached entry for a GitHub user's keys
type CacheEntry struct {
	GitHubUser string    `json:"github_user"`
	Keys       []string  `json:"keys"`
	Timestamp  time.Time `json:"timestamp"`
}

// Cache represents the cache structure
type Cache struct {
	Entries []CacheEntry `json:"entries"`
}

// Manager handles cache operations
type Manager struct {
	cacheDir string
	ttl      time.Duration
}

// NewManager creates a new cache manager
func NewManager(cacheDir string, ttl time.Duration) (*Manager, error) {
	// If cacheDir is empty, use OS temp directory
	if cacheDir == "" {
		var err error
		cacheDir, err = getTempDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get temp directory: %w", err)
		}
		// Create charon-key subdirectory in temp
		cacheDir = filepath.Join(cacheDir, "charon-key")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Manager{
		cacheDir: cacheDir,
		ttl:      ttl,
	}, nil
}

// getTempDir returns the OS temp directory (cross-platform)
func getTempDir() (string, error) {
	tempDir := os.TempDir()
	if tempDir == "" {
		return "", fmt.Errorf("temp directory not available")
	}
	return tempDir, nil
}

// getCacheFilePath returns the cache file path for a GitHub username
func (m *Manager) getCacheFilePath(githubUser string) string {
	// Sanitize username for filename (basic sanitization)
	safeName := sanitizeFilename(githubUser)
	return filepath.Join(m.cacheDir, fmt.Sprintf("%s.json", safeName))
}

// sanitizeFilename sanitizes a string for use as a filename
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		} else {
			result += "_"
		}
	}
	if result == "" {
		result = "default"
	}
	return result
}

// Write stores keys for a GitHub user in the cache
func (m *Manager) Write(githubUser string, keys []string) error {
	if githubUser == "" {
		return fmt.Errorf("GitHub username cannot be empty")
	}

	entry := CacheEntry{
		GitHubUser: githubUser,
		Keys:       keys,
		Timestamp:  time.Now(),
	}

	cache := Cache{
		Entries: []CacheEntry{entry},
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cachePath := m.getCacheFilePath(githubUser)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Read retrieves keys for a GitHub user from the cache
// Returns keys, isExpired, error
// isExpired indicates if the cache entry exists but is expired (useful for fallback)
func (m *Manager) Read(githubUser string) ([]string, bool, error) {
	if githubUser == "" {
		return nil, false, fmt.Errorf("GitHub username cannot be empty")
	}

	cachePath := m.getCacheFilePath(githubUser)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil // Cache miss, not an error
		}
		return nil, false, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Find entry for this GitHub user
	for _, entry := range cache.Entries {
		if entry.GitHubUser == githubUser {
			// Check if expired
			age := time.Since(entry.Timestamp)
			isExpired := age > m.ttl

			return entry.Keys, isExpired, nil
		}
	}

	return nil, false, nil // Entry not found
}

// IsExpired checks if the cache entry for a GitHub user is expired
// Returns false if cache doesn't exist
func (m *Manager) IsExpired(githubUser string) (bool, error) {
	if githubUser == "" {
		return false, fmt.Errorf("GitHub username cannot be empty")
	}

	cachePath := m.getCacheFilePath(githubUser)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // Cache doesn't exist, consider it expired
		}
		return false, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return true, nil // Invalid cache, consider expired
	}

	// Find entry for this GitHub user
	for _, entry := range cache.Entries {
		if entry.GitHubUser == githubUser {
			age := time.Since(entry.Timestamp)
			return age > m.ttl, nil
		}
	}

	return true, nil // Entry not found, consider expired
}

// GetCacheDir returns the cache directory path
func (m *Manager) GetCacheDir() string {
	return m.cacheDir
}

// Clear removes the cache entry for a GitHub user
func (m *Manager) Clear(githubUser string) error {
	if githubUser == "" {
		return fmt.Errorf("GitHub username cannot be empty")
	}

	cachePath := m.getCacheFilePath(githubUser)
	if err := os.Remove(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already removed, not an error
		}
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}

