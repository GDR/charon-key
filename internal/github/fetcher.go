package github

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// BaseURL is the base URL for GitHub's SSH keys API
	BaseURL = "https://github.com"
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 10 * time.Second
	// MaxRetries is the maximum number of retries for transient failures
	MaxRetries = 3
	// RetryDelay is the delay between retries
	RetryDelay = 1 * time.Second
)

// Fetcher handles fetching SSH keys from GitHub
type Fetcher struct {
	client  *http.Client
	baseURL string
}

// NewFetcher creates a new GitHub fetcher with default settings
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL: BaseURL,
	}
}

// NewFetcherWithClient creates a new GitHub fetcher with a custom HTTP client
// Useful for testing with mock clients
func NewFetcherWithClient(client *http.Client) *Fetcher {
	return &Fetcher{
		client:  client,
		baseURL: BaseURL,
	}
}

// FetchKeys fetches SSH public keys for a GitHub username
// Returns the keys as a slice of strings (one key per line)
// Returns error if the request fails or the user doesn't exist
func (f *Fetcher) FetchKeys(username string) ([]string, error) {
	if username == "" {
		return nil, fmt.Errorf("GitHub username cannot be empty")
	}

	url := fmt.Sprintf("%s/%s.keys", f.baseURL, username)

	var keys []string
	var lastErr error

	// Retry logic for transient failures
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(RetryDelay * time.Duration(attempt))
		}

		keys, lastErr = f.fetchKeysOnce(url)
		if lastErr == nil {
			return keys, nil
		}

		// Don't retry on 404 (user not found) or other client errors
		if httpErr, ok := lastErr.(*HTTPError); ok {
			if httpErr.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("GitHub user %q not found", username)
			}
			// Retry on 5xx errors (server errors)
			if httpErr.StatusCode >= 500 && attempt < MaxRetries {
				continue
			}
			// Don't retry on 4xx errors (client errors)
			return nil, lastErr
		}

		// Retry on network errors/timeouts if we have retries left
		if attempt < MaxRetries {
			continue
		}
	}

	return nil, fmt.Errorf("failed to fetch keys after %d attempts: %w", MaxRetries+1, lastErr)
}

// fetchKeysOnce performs a single HTTP request to fetch keys
func (f *Fetcher) fetchKeysOnce(url string) ([]string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to identify our tool
	req.Header.Set("User-Agent", "charon-key/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
		}
	}

	// Parse keys from response body
	keys, err := parseKeys(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keys: %w", err)
	}

	return keys, nil
}

// parseKeys parses SSH keys from the response body (one key per line)
func parseKeys(body io.Reader) ([]string, error) {
	var keys []string
	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines
		if line == "" {
			continue
		}

		// Basic validation: check if line looks like an SSH key
		if !isValidKeyFormat(line) {
			continue // Skip invalid lines (comments, etc.)
		}

		keys = append(keys, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return keys, nil
}

// isValidKeyFormat performs basic validation of SSH key format
// SSH keys typically start with: ssh-rsa, ssh-ed25519, ecdsa-sha2-nistp256, etc.
func isValidKeyFormat(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	// Check for common SSH key prefixes
	validPrefixes := []string{
		"ssh-rsa",
		"ssh-ed25519",
		"ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384",
		"ecdsa-sha2-nistp521",
		"ssh-dss", // DSA (deprecated but still seen)
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// FetchKeysForUsers fetches SSH keys for multiple GitHub users and merges them
// Returns all unique keys from all users
func (f *Fetcher) FetchKeysForUsers(usernames []string) ([]string, error) {
	if len(usernames) == 0 {
		return nil, fmt.Errorf("no usernames provided")
	}

	allKeys := make(map[string]bool) // Use map to deduplicate keys
	var errors []string

	for _, username := range usernames {
		keys, err := f.FetchKeys(username)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", username, err))
			continue // Continue fetching from other users even if one fails
		}

		for _, key := range keys {
			allKeys[key] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(allKeys))
	for key := range allKeys {
		result = append(result, key)
	}

	// If all requests failed, return error
	if len(result) == 0 && len(errors) == len(usernames) {
		return nil, fmt.Errorf("all requests failed: %s", strings.Join(errors, "; "))
	}

	// If some requests failed, we still return the keys we got
	// (errors are logged but don't prevent returning partial results)

	return result, nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	URL        string
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

