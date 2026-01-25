package resolver

import (
	"fmt"

	"github.com/dgarifullin/charon-key/internal/cache"
	"github.com/dgarifullin/charon-key/internal/config"
	"github.com/dgarifullin/charon-key/internal/github"
	"github.com/dgarifullin/charon-key/internal/logger"
)

// Resolver handles the key resolution logic
type Resolver struct {
	config  *config.Config
	fetcher *github.Fetcher
	cache   *cache.Manager
	logger  *logger.Logger
}

// NewResolver creates a new resolver with the given components
func NewResolver(cfg *config.Config, fetcher *github.Fetcher, cacheManager *cache.Manager, log *logger.Logger) *Resolver {
	return &Resolver{
		config:  cfg,
		fetcher: fetcher,
		cache:   cacheManager,
		logger:  log,
	}
}

// ResolveKeys resolves SSH keys for the given SSH username
// Returns all authorized keys (merged from all GitHub users)
func (r *Resolver) ResolveKeys(sshUsername string) ([]string, error) {
	if sshUsername == "" {
		return nil, fmt.Errorf("SSH username cannot be empty")
	}

	r.logger.Debug("resolving keys", "ssh_username", sshUsername)

	// Step 1: Look up GitHub user(s) from mapping
	githubUsers := r.config.GetGitHubUsers(sshUsername)
	if len(githubUsers) == 0 {
		r.logger.Error("no GitHub users mapped", "ssh_username", sshUsername)
		return nil, fmt.Errorf("no GitHub users mapped for SSH user %q", sshUsername)
	}

	r.logger.Debug("found GitHub users", "ssh_username", sshUsername, "github_users", githubUsers)

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
		r.logger.Error("failed to resolve keys for all GitHub users", "ssh_username", sshUsername, "errors", joinErrors(errors))
		return nil, fmt.Errorf("failed to resolve keys for all GitHub users: %s", joinErrors(errors))
	}

	if len(errors) > 0 {
		r.logger.Warn("partial failure resolving keys", "ssh_username", sshUsername, "errors", joinErrors(errors), "keys_resolved", len(result))
	}

	r.logger.Debug("resolved keys", "ssh_username", sshUsername, "total_keys", len(result))

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
		r.logger.Debug("cache read error", "github_user", githubUser, "error", err)
		// We'll try to fetch fresh keys
	}

	// Step 2: If cache exists and not expired, return cached keys
	if cachedKeys != nil && len(cachedKeys) > 0 && !isExpired {
		r.logger.Debug("cache hit", "github_user", githubUser, "keys_count", len(cachedKeys))
		return cachedKeys, nil
	}

	if cachedKeys != nil && len(cachedKeys) > 0 && isExpired {
		r.logger.Debug("cache expired", "github_user", githubUser)
	} else {
		r.logger.Debug("cache miss", "github_user", githubUser)
	}

	// Step 3: Fetch from GitHub (cache expired or missing)
	r.logger.Info("fetching keys from GitHub", "github_user", githubUser)
	keys, err := r.fetcher.FetchKeys(githubUser)
	if err != nil {
		r.logger.Warn("failed to fetch keys from GitHub", "github_user", githubUser, "error", err)
		// Network error - try to use expired cache if available
		if cachedKeys != nil && len(cachedKeys) > 0 {
			// Use expired cache as fallback (offline mode)
			r.logger.Info("using expired cache as fallback", "github_user", githubUser, "keys_count", len(cachedKeys))
			return cachedKeys, nil
		}
		// No cache available, return error
		return nil, fmt.Errorf("failed to fetch keys from GitHub and no cache available: %w", err)
	}

	r.logger.Info("fetched keys from GitHub", "github_user", githubUser, "keys_count", len(keys))

	// Step 4: Update cache with fresh keys
	if err := r.cache.Write(githubUser, keys); err != nil {
		// Cache write error - log but don't fail the request
		r.logger.Warn("failed to write cache", "github_user", githubUser, "error", err)
		// Keys are still valid, just not cached
	} else {
		r.logger.Debug("cache updated", "github_user", githubUser)
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
func NewResolverWithOptions(cfg *config.Config, fetcher *github.Fetcher, cacheManager *cache.Manager, log *logger.Logger, opts ResolverOptions) *Resolver {
	resolver := NewResolver(cfg, fetcher, cacheManager, log)
	// Options can be applied here if needed in the future
	_ = opts
	return resolver
}

