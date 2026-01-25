package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dgarifullin/charon-key/internal/config"
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

	// Parse configuration
	cfg, err := parseConfig(userMapStr, cacheDir, cacheTTLMinutes, logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Use --help for usage information\n")
		os.Exit(1)
	}

	// Get SSH username from positional arguments (passed by SSH daemon)
	args := flag.Args()
	if len(args) > 0 {
		cfg.SSHUsername = args[0]
	}

	// For now, just validate and print config (implementation continues in next milestones)
	fmt.Fprintf(os.Stderr, "Configuration loaded successfully\n")
	fmt.Fprintf(os.Stderr, "SSH Username: %s\n", cfg.SSHUsername)
	fmt.Fprintf(os.Stderr, "User Map: %v\n", cfg.UserMap)
	fmt.Fprintf(os.Stderr, "Cache Dir: %s\n", cfg.CacheDir)
	fmt.Fprintf(os.Stderr, "Cache TTL: %v\n", cfg.CacheTTL)
	fmt.Fprintf(os.Stderr, "Log Level: %s\n", cfg.LogLevel)

	// TODO: Implement key resolution in Milestone 5
	os.Exit(0)
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

