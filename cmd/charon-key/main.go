package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dgarifullin/charon-key/internal/cache"
	"github.com/dgarifullin/charon-key/internal/config"
	"github.com/dgarifullin/charon-key/internal/errors"
	"github.com/dgarifullin/charon-key/internal/github"
	"github.com/dgarifullin/charon-key/internal/logger"
	"github.com/dgarifullin/charon-key/internal/resolver"
	"github.com/dgarifullin/charon-key/internal/ssh"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var showVersion bool
	var showHelp bool
	var userMapStr string
	var cacheDir string
	var cacheTTLMinutes int
	var logLevel string

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.BoolVar(&showHelp, "h", false, "Show help information (shorthand)")
	flag.StringVar(&userMapStr, "user-map", "", "User mapping (required): sshuser1:githubuser1,sshuser1:githubuser2")
	flag.StringVar(&cacheDir, "cache-dir", "", "Cache directory (optional, default: OS temp)")
	flag.IntVar(&cacheTTLMinutes, "cache-ttl", 5, "Cache TTL in minutes (optional, default: 5)")
	flag.StringVar(&logLevel, "log-level", "info", "Log level: debug|info|warn|error (optional, default: info)")

	flag.Parse()

	if showVersion {
		fmt.Printf("charon-key version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("date: %s\n", date)
		os.Exit(0)
	}

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Initialize logger first (for error logging)
	log := logger.NewLogger(logLevel)

	// Parse configuration
	cfg, err := parseConfig(userMapStr, cacheDir, cacheTTLMinutes, logLevel)
	if err != nil {
		log.Error("configuration error", "error", err)
		errors.ExitWithCode(errors.ExitConfigError)
	}

	// Get SSH username from positional arguments (passed by SSH daemon)
	args := flag.Args()
	if len(args) > 0 {
		cfg.SSHUsername = args[0]
	}

	// Log startup configuration
	log.Info("starting charon-key", "version", version, "ssh_username", cfg.SSHUsername)
	log.Debug("configuration", "user_map", cfg.UserMap, "cache_dir", cfg.CacheDir, "cache_ttl", cfg.CacheTTL, "log_level", cfg.LogLevel)

	// Initialize cache manager
	cacheManager, err := cache.NewManager(cfg.CacheDir, cfg.CacheTTL)
	if err != nil {
		log.Error("failed to initialize cache", "error", err)
		errors.ExitWithCode(errors.ExitGeneralError)
	}
	log.Debug("cache initialized", "cache_dir", cacheManager.GetCacheDir())

	// Initialize GitHub fetcher
	fetcher := github.NewFetcher()
	fetcher.SetLogger(log)

	// Initialize resolver
	resolver := resolver.NewResolver(cfg, fetcher, cacheManager, log)

	// Resolve keys
	githubKeys, err := resolver.ResolveKeysForSSHUser()
	if err != nil {
		log.Error("failed to resolve keys", "error", err)
		errors.ExitWithCode(errors.ExitNetworkError)
	}

	// Validate keys (fail secure on invalid keys)
	for _, key := range githubKeys {
		if !isValidKeyFormat(key) {
			log.Error("invalid key format detected", "key", key)
			errors.HandleInvalidKey(key, fmt.Errorf("key does not match valid SSH key format"))
		}
	}

	// Initialize SSH manager
	sshManager, err := ssh.NewManager(cfg.SSHUsername)
	if err != nil {
		log.Warn("failed to initialize SSH manager, using current user", "error", err)
		sshManager, err = ssh.NewManager("")
		if err != nil {
			log.Error("failed to initialize SSH manager with current user", "error", err)
			errors.ExitWithCode(errors.ExitPermissionError)
		}
	}

	// Get all keys (merge with existing authorized_keys)
	output, err := sshManager.GetAllKeys(githubKeys)
	if err != nil {
		log.Warn("failed to read existing authorized_keys, using GitHub keys only", "error", err)
		// Still output GitHub keys even if we can't read existing file
		output = ssh.FormatKeys(githubKeys)
	}

	// Output to stdout (SSH daemon reads from here)
	fmt.Print(output)

	log.Debug("completed successfully", "total_keys", len(githubKeys))
	errors.ExitWithCode(errors.ExitSuccess)
}

// isValidKeyFormat performs basic validation of SSH key format
// This is a duplicate from github package but needed here for validation
func isValidKeyFormat(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	validPrefixes := []string{
		"ssh-rsa",
		"ssh-ed25519",
		"ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384",
		"ecdsa-sha2-nistp521",
		"ssh-dss",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

func parseConfig(userMapStr, cacheDir string, cacheTTLMinutes int, logLevel string) (*config.Config, error) {
	// Validate required user-map
	if userMapStr == "" {
		return nil, fmt.Errorf("--user-map is required")
	}

	// Parse user mapping
	userMap, err := config.ParseUserMap(userMapStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user-map: %w", err)
	}

	// Validate log level
	if err := config.ValidateLogLevel(logLevel); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Validate cache TTL
	if cacheTTLMinutes < 1 {
		return nil, fmt.Errorf("cache-ttl must be at least 1 minute, got %d", cacheTTLMinutes)
	}

	cfg := &config.Config{
		UserMap:  userMap,
		CacheDir: cacheDir, // Empty means use OS temp (handled in cache package)
		CacheTTL: time.Duration(cacheTTLMinutes) * time.Minute,
		LogLevel: logLevel,
	}

	return cfg, nil
}

func printHelp() {
	fmt.Println("charon-key - SSH AuthorizedKeysCommand for GitHub SSH keys")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  charon-key [OPTIONS] [SSH-USERNAME]")
	fmt.Println()
	fmt.Println("Description:")
	fmt.Println("  Fetches SSH public keys from GitHub and merges them with existing")
	fmt.Println("  authorized_keys file. Designed to be used as AuthorizedKeysCommand")
	fmt.Println("  in sshd_config.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --user-map <mapping>     User mapping (required)")
	fmt.Println("                          Format: sshuser1:githubuser1,sshuser1:githubuser2")
	fmt.Println("  --cache-dir <dir>       Cache directory (optional, default: OS temp)")
	fmt.Println("  --cache-ttl <minutes>   Cache TTL in minutes (optional, default: 5)")
	fmt.Println("  --log-level <level>     Log level: debug|info|warn|error (optional, default: info)")
	fmt.Println("  -h, --help              Show this help message")
	fmt.Println("  -v, --version           Show version information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  charon-key --user-map alice:alice-github,bob:bob-github")
	fmt.Println("  charon-key --user-map *:dgarifullin --cache-dir /var/cache/charon-key")
	fmt.Println()
	fmt.Println("SSH Configuration:")
	fmt.Println("  Add to /etc/ssh/sshd_config:")
	fmt.Println("    AuthorizedKeysCommand /path/to/charon-key --user-map <mapping>")
	fmt.Println("    AuthorizedKeysCommandUser root")
}

