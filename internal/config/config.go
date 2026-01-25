package config

import (
	"fmt"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	// UserMap maps SSH usernames to GitHub usernames
	// Key: SSH username (or "*" for wildcard)
	// Value: List of GitHub usernames
	UserMap map[string][]string

	// CacheDir is the directory for caching keys
	CacheDir string

	// CacheTTL is the cache time-to-live in minutes
	CacheTTL time.Duration

	// LogLevel is the logging level (debug, info, warn, error)
	LogLevel string

	// SSHUsername is the SSH username passed by the SSH daemon
	SSHUsername string
}

// ParseUserMap parses the user mapping string into a map
// Format: "sshuser1:githubuser1,sshuser1:githubuser2,sshuser2:githubuser1"
// Returns error if format is invalid
func ParseUserMap(userMapStr string) (map[string][]string, error) {
	if userMapStr == "" {
		return nil, fmt.Errorf("user-map cannot be empty")
	}

	result := make(map[string][]string)

	// Split by comma to get individual mappings
	pairs := strings.Split(userMapStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by colon to get sshuser:githubuser
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping format: %q (expected sshuser:githubuser)", pair)
		}

		sshUser := strings.TrimSpace(parts[0])
		githubUser := strings.TrimSpace(parts[1])

		if sshUser == "" {
			return nil, fmt.Errorf("SSH username cannot be empty in mapping: %q", pair)
		}
		if githubUser == "" {
			return nil, fmt.Errorf("GitHub username cannot be empty in mapping: %q", pair)
		}

		// Add to map (append if SSH user already exists)
		result[sshUser] = append(result[sshUser], githubUser)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid mappings found in user-map")
	}

	return result, nil
}

// ValidateLogLevel validates the log level
func ValidateLogLevel(level string) error {
	validLevels := []string{"debug", "info", "warn", "error"}
	levelLower := strings.ToLower(level)
	for _, valid := range validLevels {
		if levelLower == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid log level: %q (valid: %s)", level, strings.Join(validLevels, ", "))
}

// GetGitHubUsers returns the GitHub users for a given SSH username
// Returns empty slice if SSH user not found
// Handles wildcard "*" mapping
func (c *Config) GetGitHubUsers(sshUsername string) []string {
	// Check for exact match first
	if users, ok := c.UserMap[sshUsername]; ok {
		return users
	}

	// Check for wildcard match
	if users, ok := c.UserMap["*"]; ok {
		return users
	}

	return []string{}
}

