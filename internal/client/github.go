package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubClient handles interactions with GitHub API
type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewGitHubClient creates a new GitHub client with TLS configuration
func NewGitHubClient() *GitHubClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12, // Enforce minimum TLS 1.2
				InsecureSkipVerify: false,            // Always verify certificates
				ServerName:         "github.com",     // Enforce server name verification
			},
		},
	}

	return &GitHubClient{
		httpClient: client,
		baseURL:    "https://github.com",
	}
}

// FetchUserKeys fetches SSH public keys for a given GitHub username
func (c *GitHubClient) FetchUserKeys(username string) ([]string, error) {
	url := fmt.Sprintf("%s/%s.keys", c.baseURL, username)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keys: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("user not found")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Filter out empty lines
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	var keys []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
} 