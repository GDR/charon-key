package resolver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dgarifullin/charon-key/internal/cache"
	"github.com/dgarifullin/charon-key/internal/config"
	"github.com/dgarifullin/charon-key/internal/github"
)

func TestNewResolver(t *testing.T) {
	cfg := &config.Config{
		UserMap: map[string][]string{
			"alice": {"alice-github"},
		},
		CacheTTL: 5 * time.Minute,
	}

	cacheManager, _ := cache.NewManager("/tmp/test-resolver", 5*time.Minute)
	defer cacheManager.Clear("alice-github")

	fetcher := github.NewFetcher()
	resolver := NewResolver(cfg, fetcher, cacheManager)

	if resolver == nil {
		t.Fatal("NewResolver() returned nil")
	}
	if resolver.config != cfg {
		t.Error("Resolver config not set correctly")
	}
	if resolver.fetcher != fetcher {
		t.Error("Resolver fetcher not set correctly")
	}
	if resolver.cache != cacheManager {
		t.Error("Resolver cache not set correctly")
	}
}

func TestResolver_ResolveKeys(t *testing.T) {
	tests := []struct {
		name         string
		sshUsername  string
		userMap      map[string][]string
		githubResp   map[string]string // github user -> keys response
		wantKeys     int
		wantError    bool
		errorContains string
	}{
		{
			name:        "single GitHub user",
			sshUsername: "alice",
			userMap: map[string][]string{
				"alice": {"alice-github"},
			},
			githubResp: map[string]string{
				"alice-github": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB alice@example.com\n",
			},
			wantKeys:  1,
			wantError: false,
		},
		{
			name:        "multiple GitHub users",
			sshUsername: "alice",
			userMap: map[string][]string{
				"alice": {"alice-github", "shared-github"},
			},
			githubResp: map[string]string{
				"alice-github":  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB alice@example.com\n",
				"shared-github": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI shared@example.com\n",
			},
			wantKeys:  2,
			wantError: false,
		},
		{
			name:        "wildcard mapping",
			sshUsername: "unknown",
			userMap: map[string][]string{
				"*": {"wildcard-github"},
			},
			githubResp: map[string]string{
				"wildcard-github": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB wildcard@example.com\n",
			},
			wantKeys:  1,
			wantError: false,
		},
		{
			name:        "no mapping",
			sshUsername: "nonexistent",
			userMap:     map[string][]string{},
			githubResp:  map[string]string{},
			wantKeys:    0,
			wantError:   true,
			errorContains: "no GitHub users mapped",
		},
		{
			name:        "empty SSH username",
			sshUsername: "",
			userMap: map[string][]string{
				"alice": {"alice-github"},
			},
			githubResp: map[string]string{},
			wantKeys:   0,
			wantError:  true,
			errorContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract username from path (format: /username.keys)
				path := strings.TrimPrefix(r.URL.Path, "/")
				username := strings.TrimSuffix(path, ".keys")

				if keys, ok := tt.githubResp[username]; ok {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(keys))
				} else {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("Not Found"))
				}
			}))
			defer server.Close()

			// Create components
			cfg := &config.Config{
				UserMap:  tt.userMap,
				CacheTTL: 5 * time.Minute,
			}

			cacheManager, _ := cache.NewManager("/tmp/test-resolver-"+tt.name, 5*time.Minute)
			defer func() {
				for user := range tt.userMap {
					cacheManager.Clear(user)
				}
			}()

			fetcher := github.NewFetcher()
			fetcher.SetBaseURL(server.URL)

			resolver := NewResolver(cfg, fetcher, cacheManager)

			keys, err := resolver.ResolveKeys(tt.sshUsername)

			if (err != nil) != tt.wantError {
				t.Errorf("ResolveKeys() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				if tt.errorContains != "" && err != nil && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ResolveKeys() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
				return
			}

			if len(keys) != tt.wantKeys {
				t.Errorf("ResolveKeys() returned %d keys, want %d", len(keys), tt.wantKeys)
			}
		})
	}
}

func TestResolver_CacheUsage(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com\n"))
	}))
	defer server.Close()

	cacheDir := "/tmp/test-resolver-cache"
	cacheManager, _ := cache.NewManager(cacheDir, 5*time.Minute)
	defer cacheManager.Clear("test-github")

	cfg := &config.Config{
		UserMap: map[string][]string{
			"alice": {"test-github"},
		},
		CacheTTL: 5 * time.Minute,
	}

	fetcher := github.NewFetcher()
	fetcher.SetBaseURL(server.URL)

	resolver := NewResolver(cfg, fetcher, cacheManager)

	// First call - should fetch from GitHub and cache
	keys1, err := resolver.ResolveKeys("alice")
	if err != nil {
		t.Fatalf("ResolveKeys() error = %v", err)
	}
	if len(keys1) == 0 {
		t.Error("ResolveKeys() returned no keys")
	}

	// Second call - should use cache (not hit GitHub)
	keys2, err := resolver.ResolveKeys("alice")
	if err != nil {
		t.Fatalf("ResolveKeys() error = %v", err)
	}
	if len(keys2) != len(keys1) {
		t.Errorf("ResolveKeys() returned %d keys on second call, want %d", len(keys2), len(keys1))
	}

	// Verify keys match
	if keys1[0] != keys2[0] {
		t.Errorf("Cached keys don't match: %q != %q", keys1[0], keys2[0])
	}
}

func TestResolver_OfflineMode(t *testing.T) {
	// Create server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	}))
	defer server.Close()

	cacheDir := "/tmp/test-resolver-offline"
	cacheManager, _ := cache.NewManager(cacheDir, 5*time.Minute)
	defer cacheManager.Clear("test-github")

	// Pre-populate cache
	cachedKeys := []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB cached@example.com"}
	cacheManager.Write("test-github", cachedKeys)

	cfg := &config.Config{
		UserMap: map[string][]string{
			"alice": {"test-github"},
		},
		CacheTTL: 5 * time.Minute,
	}

	fetcher := github.NewFetcher()
	fetcher.SetBaseURL(server.URL)

	resolver := NewResolver(cfg, fetcher, cacheManager)

	// Should use expired cache when GitHub fails
	keys, err := resolver.ResolveKeys("alice")
	if err != nil {
		t.Errorf("ResolveKeys() error = %v, want nil (should use cache)", err)
	}
	if len(keys) == 0 {
		t.Error("ResolveKeys() returned no keys, want cached keys")
	}
	if keys[0] != cachedKeys[0] {
		t.Errorf("ResolveKeys() returned %q, want %q", keys[0], cachedKeys[0])
	}
}

func TestResolver_Deduplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB duplicate@example.com\n"))
	}))
	defer server.Close()

	cacheDir := "/tmp/test-resolver-dedup"
	cacheManager, _ := cache.NewManager(cacheDir, 5*time.Minute)
	defer func() {
		cacheManager.Clear("user1")
		cacheManager.Clear("user2")
	}()

	cfg := &config.Config{
		UserMap: map[string][]string{
			"alice": {"user1", "user2"}, // Both return same key
		},
		CacheTTL: 5 * time.Minute,
	}

	fetcher := github.NewFetcher()
	fetcher.SetBaseURL(server.URL)

	resolver := NewResolver(cfg, fetcher, cacheManager)

	keys, err := resolver.ResolveKeys("alice")
	if err != nil {
		t.Fatalf("ResolveKeys() error = %v", err)
	}

	// Should have only one key (deduplicated)
	if len(keys) != 1 {
		t.Errorf("ResolveKeys() returned %d keys, want 1 (deduplicated)", len(keys))
	}
}


