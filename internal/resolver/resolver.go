package resolver

import (
	"fmt"

	"github.com/dgarifullin/charon-key/internal/cache"
	"github.com/dgarifullin/charon-key/internal/config"
	"github.com/dgarifullin/charon-key/internal/github"
)

// Resolver handles the key resolution logic
type Resolver struct {
	config  *config.Config
	fetcher *github.Fetcher
	cache   *cache.Manager
}

// NewResolver creates a new resolver with the given components
func NewResolver(cfg *config.Config, fetcher *github.Fetcher, cacheManager *cache.Manager) *Resolver {
	return &Resolver{
		config:  cfg,
		fetcher: fetcher,
		cache:   cacheManager,
	}
}

// ResolveKeys resolves SSH keys for the given SSH username
// Returns all authorized keys (merged from all GitHub users)
func (r *Resolver) ResolveKeys(sshUsername string) ([]string, error) {
	if sshUsername == "" {
		return nil, fmt.Errorf("SSH username cannot be empty")
	}

	// Step 1: Look up GitHub user(s) from mapping
	githubUsers := r.config.GetGitHubUsers(sshUsername)
	if len(githubUsers) == 0 {
		return nil, fmt.Errorf("no GitHub users mapped for SSH user %q", sshUsername)
	}

	// Step 2: Resolve keys for all GitHub users
	allKeys := make(map[string]bool) // Use map to deduplicate
	var errors []string

	for _, githubUser := range githubUsers {
		keys, err := r.resolveKeysForGitHubUser(githubUser)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", githubUser, err))
			continue // Continue with other users even if one fails
		}

		// Merge keys (deduplicate)
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
	if len(result) == 0 && len(errors) == len(githubUsers) {
		return nil, fmt.Errorf("failed to resolve keys for all GitHub users: %s", joinErrors(errors))
	}

	// Return partial results if some succeeded
	return result, nil
}

// resolveKeysForGitHubUser resolves keys for a single GitHub user
// Implements the full flow: cache check -> fetch if needed -> update cache
func (r *Resolver) resolveKeysForGitHubUser(githubUser string) ([]string, error) {
	// Step 1: Check cache
	cachedKeys, isExpired, err := r.cache.Read(githubUser)
	if err != nil {
		// Cache read error (not a cache miss) - log but continue
		// We'll try to fetch fresh keys
	}

	// Step 2: If cache exists and not expired, return cached keys
	if cachedKeys != nil && len(cachedKeys) > 0 && !isExpired {
		return cachedKeys, nil
	}

	// Step 3: Fetch from GitHub (cache expired or missing)
	keys, err := r.fetcher.FetchKeys(githubUser)
	if err != nil {
		// Network error - try to use expired cache if available
		if cachedKeys != nil && len(cachedKeys) > 0 {
			// Use expired cache as fallback (offline mode)
			return cachedKeys, nil
		}
		// No cache available, return error
		return nil, fmt.Errorf("failed to fetch keys from GitHub and no cache available: %w", err)
	}

	// Step 4: Update cache with fresh keys
	if err := r.cache.Write(githubUser, keys); err != nil {
		// Cache write error - log but don't fail the request
		// Keys are still valid, just not cached
	}

	return keys, nil
}

// ResolveKeysForSSHUser resolves keys for the SSH username from config
// This is a convenience method that uses the SSH username from config
func (r *Resolver) ResolveKeysForSSHUser() ([]string, error) {
	if r.config.SSHUsername == "" {
		return nil, fmt.Errorf("SSH username not set in config")
	}
	return r.ResolveKeys(r.config.SSHUsername)
}

// joinErrors joins multiple error messages
func joinErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	if len(errors) == 1 {
		return errors[0]
	}
	result := errors[0]
	for i := 1; i < len(errors); i++ {
		result += "; " + errors[i]
	}
	return result
}

// ResolverOptions allows configuring resolver behavior
type ResolverOptions struct {
	// UseExpiredCache controls whether to use expired cache when GitHub is unreachable
	// Default: true (offline mode support)
	UseExpiredCache bool
}

// NewResolverWithOptions creates a resolver with custom options
func NewResolverWithOptions(cfg *config.Config, fetcher *github.Fetcher, cacheManager *cache.Manager, opts ResolverOptions) *Resolver {
	resolver := NewResolver(cfg, fetcher, cacheManager)
	// Options can be applied here if needed in the future
	_ = opts
	return resolver
}

