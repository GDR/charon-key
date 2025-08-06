package config

import (
	"os"
	"strings"
)

// Config holds application configuration
type Config struct {
	Usernames []string
	FilePath  string
	Quiet     bool
}

// ReadUsernamesFromFile reads usernames from a file, one per line
// Skips empty lines and comments (lines starting with #)
func ReadUsernamesFromFile(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var usernames []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			usernames = append(usernames, line)
		}
	}

	return usernames, nil
}

// MergeUsernames combines usernames from multiple sources
func (c *Config) MergeUsernames(flagUsernames, fileUsernames, argUsernames []string) {
	c.Usernames = append(c.Usernames, flagUsernames...)
	c.Usernames = append(c.Usernames, fileUsernames...)
	c.Usernames = append(c.Usernames, argUsernames...)
}

// HasUsernames returns true if at least one username is configured
func (c *Config) HasUsernames() bool {
	return len(c.Usernames) > 0
} 