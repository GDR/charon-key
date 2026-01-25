package ssh

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Manager handles SSH authorized_keys operations
type Manager struct {
	authorizedKeysPath string
}

// NewManager creates a new SSH manager
// If username is empty, uses current user
func NewManager(username string) (*Manager, error) {
	var homeDir string

	if username == "" {
		// Use current user
		currentUser, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get current user: %w", err)
		}
		homeDir = currentUser.HomeDir
	} else {
		// Look up specified user
		u, err := user.Lookup(username)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup user %q: %w", username, err)
		}
		homeDir = u.HomeDir
	}

	authorizedKeysPath := filepath.Join(homeDir, ".ssh", "authorized_keys")

	return &Manager{
		authorizedKeysPath: authorizedKeysPath,
	}, nil
}

// NewManagerWithPath creates a new SSH manager with a custom authorized_keys path
// Useful for testing
func NewManagerWithPath(path string) *Manager {
	return &Manager{
		authorizedKeysPath: path,
	}
}

// GetAuthorizedKeysPath returns the path to the authorized_keys file
func (m *Manager) GetAuthorizedKeysPath() string {
	return m.authorizedKeysPath
}

// ReadExistingKeys reads existing keys from the authorized_keys file
// Returns empty slice if file doesn't exist (not an error)
// Returns error only if file exists but cannot be read
func (m *Manager) ReadExistingKeys() ([]string, error) {
	file, err := os.Open(m.authorizedKeysPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // File doesn't exist, return empty slice
		}
		return nil, fmt.Errorf("failed to open authorized_keys file: %w", err)
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keys = append(keys, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read authorized_keys file: %w", err)
	}

	return keys, nil
}

// MergeKeys merges GitHub keys with existing authorized_keys
// Deduplicates keys and returns them in a consistent format
func (m *Manager) MergeKeys(githubKeys []string, existingKeys []string) []string {
	// Use map to deduplicate (key content as key)
	keyMap := make(map[string]bool)
	var result []string

	// Add existing keys first (preserve order)
	for _, key := range existingKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		// Normalize key (remove comments, extra whitespace)
		normalized := normalizeKey(key)
		if normalized != "" && !keyMap[normalized] {
			keyMap[normalized] = true
			result = append(result, key) // Keep original format
		}
	}

	// Add GitHub keys (avoid duplicates)
	for _, key := range githubKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		normalized := normalizeKey(key)
		if normalized != "" && !keyMap[normalized] {
			keyMap[normalized] = true
			result = append(result, key)
		}
	}

	return result
}

// normalizeKey normalizes a key for comparison (removes comments and extra whitespace)
// This helps with deduplication
func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}

	// SSH keys typically have format: "key-type key-data [comment]"
	// We extract just the key-type and key-data for comparison
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return key // Malformed, return as-is
	}

	// Return key-type and key-data (first two parts)
	return strings.Join(parts[:2], " ")
}

// FormatKeys formats keys for SSH daemon output (one key per line)
func FormatKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return strings.Join(keys, "\n") + "\n"
}

// GetAllKeys reads existing keys and merges with GitHub keys
// Returns formatted output ready for SSH daemon
func (m *Manager) GetAllKeys(githubKeys []string) (string, error) {
	existingKeys, err := m.ReadExistingKeys()
	if err != nil {
		// If we can't read existing keys, still return GitHub keys
		// Log error but don't fail completely
		existingKeys = []string{}
	}

	mergedKeys := m.MergeKeys(githubKeys, existingKeys)
	return FormatKeys(mergedKeys), nil
}

