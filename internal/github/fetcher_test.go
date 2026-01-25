package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewFetcher(t *testing.T) {
	fetcher := NewFetcher()
	if fetcher == nil {
		t.Fatal("NewFetcher() returned nil")
	}
	if fetcher.client == nil {
		t.Error("Fetcher client is nil")
	}
	if fetcher.baseURL != BaseURL {
		t.Errorf("Fetcher baseURL = %q, want %q", fetcher.baseURL, BaseURL)
	}
}

func TestFetcher_FetchKeys(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		responseBody   string
		statusCode     int
		wantKeys       []string
		wantError      bool
		errorContains  string
	}{
		{
			name:         "successful fetch single key",
			username:     "testuser",
			responseBody: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com\n",
			statusCode:   http.StatusOK,
			wantKeys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com"},
			wantError:    false,
		},
		{
			name: "successful fetch multiple keys",
			username: "testuser",
			responseBody: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com
`,
			statusCode: http.StatusOK,
			wantKeys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com",
			},
			wantError: false,
		},
		{
			name:         "user not found",
			username:     "nonexistent",
			responseBody: "Not Found",
			statusCode:   http.StatusNotFound,
			wantKeys:     nil,
			wantError:    true,
			errorContains: "not found",
		},
		{
			name:         "server error",
			username:     "testuser",
			responseBody: "Internal Server Error",
			statusCode:   http.StatusInternalServerError,
			wantKeys:     nil,
			wantError:    true,
		},
		{
			name:         "empty username",
			username:     "",
			responseBody: "",
			statusCode:   http.StatusOK,
			wantKeys:     nil,
			wantError:    true,
			errorContains: "cannot be empty",
		},
		{
			name: "empty response",
			username: "testuser",
			responseBody: "",
			statusCode: http.StatusOK,
			wantKeys: []string{},
			wantError: false,
		},
		{
			name: "skips invalid lines",
			username: "testuser",
			responseBody: `# This is a comment
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB valid@example.com
invalid line
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI valid2@example.com
`,
			statusCode: http.StatusOK,
			wantKeys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB valid@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI valid2@example.com",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != fmt.Sprintf("/%s.keys", tt.username) {
					t.Errorf("Request path = %q, want %q", r.URL.Path, fmt.Sprintf("/%s.keys", tt.username))
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create fetcher with test server URL
			fetcher := NewFetcher()
			fetcher.baseURL = server.URL

			keys, err := fetcher.FetchKeys(tt.username)

			if (err != nil) != tt.wantError {
				t.Errorf("FetchKeys() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("FetchKeys() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
				return
			}

			if len(keys) != len(tt.wantKeys) {
				t.Errorf("FetchKeys() returned %d keys, want %d", len(keys), len(tt.wantKeys))
				return
			}

			// Check keys match (order may vary)
			keysMap := make(map[string]bool)
			for _, key := range keys {
				keysMap[key] = true
			}
			for _, wantKey := range tt.wantKeys {
				if !keysMap[wantKey] {
					t.Errorf("FetchKeys() missing key: %q", wantKey)
				}
			}
		})
	}
}

func TestFetcher_FetchKeysForUsers(t *testing.T) {
	tests := []struct {
		name          string
		usernames     []string
		responses     map[string]string // username -> response body
		wantKeys      []string
		wantError     bool
		errorContains string
	}{
		{
			name:      "single user",
			usernames: []string{"user1"},
			responses: map[string]string{
				"user1": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com\n",
			},
			wantKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com"},
			wantError: false,
		},
		{
			name:      "multiple users",
			usernames: []string{"user1", "user2"},
			responses: map[string]string{
				"user1": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com\n",
				"user2": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user2@example.com\n",
			},
			wantKeys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI user2@example.com",
			},
			wantError: false,
		},
		{
			name:      "deduplicates keys",
			usernames: []string{"user1", "user2"},
			responses: map[string]string{
				"user1": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB same@example.com\n",
				"user2": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB same@example.com\n",
			},
			wantKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB same@example.com"},
			wantError: false,
		},
		{
			name:      "partial failure",
			usernames: []string{"user1", "nonexistent"},
			responses: map[string]string{
				"user1": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com\n",
			},
			wantKeys:  []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB user1@example.com"},
			wantError: false, // Partial results are acceptable
		},
		{
			name:          "all users fail",
			usernames:     []string{"nonexistent1", "nonexistent2"},
			responses:     map[string]string{},
			wantKeys:      nil,
			wantError:     true,
			errorContains: "all requests failed",
		},
		{
			name:          "empty usernames",
			usernames:     []string{},
			responses:     map[string]string{},
			wantKeys:      nil,
			wantError:     true,
			errorContains: "no usernames provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Extract username from path
				path := strings.TrimPrefix(r.URL.Path, "/")
				username := strings.TrimSuffix(path, ".keys")

				if body, ok := tt.responses[username]; ok {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(body))
				} else {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("Not Found"))
				}
			}))
			defer server.Close()

			// Create fetcher with test server URL
			fetcher := NewFetcher()
			fetcher.baseURL = server.URL

			keys, err := fetcher.FetchKeysForUsers(tt.usernames)

			if (err != nil) != tt.wantError {
				t.Errorf("FetchKeysForUsers() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("FetchKeysForUsers() error = %q, want error containing %q", err.Error(), tt.errorContains)
				}
				return
			}

			if len(keys) != len(tt.wantKeys) {
				t.Errorf("FetchKeysForUsers() returned %d keys, want %d", len(keys), len(tt.wantKeys))
				return
			}

			// Check keys match (order may vary)
			keysMap := make(map[string]bool)
			for _, key := range keys {
				keysMap[key] = true
			}
			for _, wantKey := range tt.wantKeys {
				if !keysMap[wantKey] {
					t.Errorf("FetchKeysForUsers() missing key: %q", wantKey)
				}
			}
		})
	}
}

func TestIsValidKeyFormat(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"ssh-rsa", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com", true},
		{"ssh-ed25519", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test@example.com", true},
		{"ecdsa-sha2-nistp256", "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAI test@example.com", true},
		{"ecdsa-sha2-nistp384", "ecdsa-sha2-nistp384 AAAAE2VjZHNhLXNoYTItbmlzdHAzODQAAAAI test@example.com", true},
		{"ecdsa-sha2-nistp521", "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAI test@example.com", true},
		{"ssh-dss", "ssh-dss AAAAB3NzaC1kc3MAAACBA test@example.com", true},
		{"comment", "# This is a comment", false},
		{"empty", "", false},
		{"whitespace", "   ", false},
		{"invalid", "not-a-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidKeyFormat(tt.key)
			if got != tt.want {
				t.Errorf("isValidKeyFormat(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestHTTPError(t *testing.T) {
	err := &HTTPError{
		StatusCode: http.StatusNotFound,
		URL:        "https://github.com/test.keys",
		Message:    "Not Found",
	}

	if err.Error() != "Not Found" {
		t.Errorf("HTTPError.Error() = %q, want %q", err.Error(), "Not Found")
	}
}

func TestFetcher_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Return server error for first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Success on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com\n"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	fetcher.baseURL = server.URL

	keys, err := fetcher.FetchKeys("testuser")
	if err != nil {
		t.Errorf("FetchKeys() error = %v, want nil", err)
	}
	if len(keys) != 1 {
		t.Errorf("FetchKeys() returned %d keys, want 1", len(keys))
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestFetcher_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com\n"))
	}))
	defer server.Close()

	fetcher := NewFetcher()
	fetcher.client.Timeout = 100 * time.Millisecond // Very short timeout
	fetcher.baseURL = server.URL

	_, err := fetcher.FetchKeys("testuser")
	if err == nil {
		t.Error("FetchKeys() expected timeout error, got nil")
	}
}

