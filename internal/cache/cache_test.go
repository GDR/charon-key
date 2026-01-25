package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name      string
		cacheDir  string
		ttl       time.Duration
		wantError bool
	}{
		{
			name:      "custom cache dir",
			cacheDir:  "/tmp/test-charon-key",
			ttl:       5 * time.Minute,
			wantError: false,
		},
		{
			name:      "empty cache dir (use temp)",
			cacheDir:  "",
			ttl:       5 * time.Minute,
			wantError: false,
		},
		{
			name:      "zero TTL",
			cacheDir:  "/tmp/test-charon-key-2",
			ttl:       0,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.cacheDir, tt.ttl)
			if (err != nil) != tt.wantError {
				t.Errorf("NewManager() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && manager == nil {
				t.Error("NewManager() returned nil manager")
				return
			}
			if !tt.wantError {
				// Cleanup
				if tt.cacheDir != "" {
					os.RemoveAll(tt.cacheDir)
				} else {
					os.RemoveAll(manager.cacheDir)
				}
			}
		})
	}
}

func TestManager_WriteRead(t *testing.T) {
	cacheDir := "/tmp/test-charon-key-write-read"
	defer os.RemoveAll(cacheDir)

	manager, err := NewManager(cacheDir, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	githubUser := "testuser"
	keys := []string{
		"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test2@example.com",
	}

	// Write
	if err := manager.Write(githubUser, keys); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read
	readKeys, isExpired, err := manager.Read(githubUser)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if isExpired {
		t.Error("Read() isExpired = true, want false")
	}
	if len(readKeys) != len(keys) {
		t.Errorf("Read() returned %d keys, want %d", len(readKeys), len(keys))
	}

	// Verify keys match
	keysMap := make(map[string]bool)
	for _, key := range readKeys {
		keysMap[key] = true
	}
	for _, wantKey := range keys {
		if !keysMap[wantKey] {
			t.Errorf("Read() missing key: %q", wantKey)
		}
	}
}

func TestManager_Expiration(t *testing.T) {
	cacheDir := "/tmp/test-charon-key-expiration"
	defer os.RemoveAll(cacheDir)

	// Create manager with very short TTL
	manager, err := NewManager(cacheDir, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	githubUser := "testuser"
	keys := []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com"}

	// Write
	if err := manager.Write(githubUser, keys); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read immediately (should not be expired)
	_, isExpired, err := manager.Read(githubUser)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if isExpired {
		t.Error("Read() isExpired = true immediately after write, want false")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Read after expiration (should be expired but still return keys)
	readKeys, isExpired, err := manager.Read(githubUser)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !isExpired {
		t.Error("Read() isExpired = false after TTL, want true")
	}
	if len(readKeys) == 0 {
		t.Error("Read() returned no keys after expiration, want keys for fallback")
	}
}

func TestManager_CacheMiss(t *testing.T) {
	cacheDir := "/tmp/test-charon-key-miss"
	defer os.RemoveAll(cacheDir)

	manager, err := NewManager(cacheDir, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Read non-existent user
	keys, isExpired, err := manager.Read("nonexistent")
	if err != nil {
		t.Fatalf("Read() error = %v, want nil for cache miss", err)
	}
	if keys != nil {
		t.Errorf("Read() returned keys = %v, want nil for cache miss", keys)
	}
	if isExpired {
		t.Error("Read() isExpired = true for cache miss, want false")
	}
}

func TestManager_Clear(t *testing.T) {
	cacheDir := "/tmp/test-charon-key-clear"
	defer os.RemoveAll(cacheDir)

	manager, err := NewManager(cacheDir, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	githubUser := "testuser"
	keys := []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com"}

	// Write
	if err := manager.Write(githubUser, keys); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Verify it exists
	_, _, err = manager.Read(githubUser)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	// Clear
	if err := manager.Clear(githubUser); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify it's gone
	readKeys, _, err := manager.Read(githubUser)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if readKeys != nil {
		t.Errorf("Read() returned keys after Clear(), want nil")
	}
}

func TestManager_GetCacheDir(t *testing.T) {
	cacheDir := "/tmp/test-charon-key-dir"
	defer os.RemoveAll(cacheDir)

	manager, err := NewManager(cacheDir, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager.GetCacheDir() != cacheDir {
		t.Errorf("GetCacheDir() = %q, want %q", manager.GetCacheDir(), cacheDir)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		notEmpty bool
	}{
		{"simple", "testuser", "testuser", true},
		{"with-dash", "test-user", "test-user", true},
		{"with-underscore", "test_user", "test_user", true},
		{"with-numbers", "user123", "user123", true},
		{"with-special", "user@github", "user_github", true},
		{"with-spaces", "user name", "user_name", true},
		{"empty", "", "default", false},
		{"all-special", "@#$%", "____", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if tt.notEmpty && got == "" {
				t.Errorf("sanitizeFilename(%q) = %q, want non-empty", tt.input, got)
			}
			if !tt.notEmpty && got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestManager_MultipleUsers(t *testing.T) {
	cacheDir := "/tmp/test-charon-key-multi"
	defer os.RemoveAll(cacheDir)

	manager, err := NewManager(cacheDir, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	users := []struct {
		username string
		keys     []string
	}{
		{"user1", []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com"}},
		{"user2", []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user2@example.com"}},
	}

	// Write for multiple users
	for _, u := range users {
		if err := manager.Write(u.username, u.keys); err != nil {
			t.Fatalf("Write(%q) error = %v", u.username, err)
		}
	}

	// Read for each user
	for _, u := range users {
		keys, _, err := manager.Read(u.username)
		if err != nil {
			t.Fatalf("Read(%q) error = %v", u.username, err)
		}
		if len(keys) != len(u.keys) {
			t.Errorf("Read(%q) returned %d keys, want %d", u.username, len(keys), len(u.keys))
		}
	}
}

func TestGetTempDir(t *testing.T) {
	tempDir, err := getTempDir()
	if err != nil {
		t.Fatalf("getTempDir() error = %v", err)
	}
	if tempDir == "" {
		t.Error("getTempDir() returned empty string")
	}

	// Verify it's a valid directory
	info, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", tempDir, err)
	}
	if !info.IsDir() {
		t.Errorf("getTempDir() returned %q which is not a directory", tempDir)
	}
}

func TestManager_EmptyCacheDir(t *testing.T) {
	// Test that empty cacheDir uses temp directory
	manager, err := NewManager("", 5*time.Minute)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	cacheDir := manager.GetCacheDir()
	if cacheDir == "" {
		t.Error("GetCacheDir() returned empty string")
	}

	// Should contain "charon-key" in the path
	baseName := filepath.Base(cacheDir)
	parent := filepath.Dir(cacheDir)
	parentBase := filepath.Base(parent)
	if baseName != "charon-key" && parentBase != "charon-key" {
		t.Errorf("GetCacheDir() = %q, expected to contain 'charon-key' in path", cacheDir)
	}

	// Cleanup
	defer os.RemoveAll(cacheDir)
}

