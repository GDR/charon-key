package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var showVersion bool
	var showHelp bool

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.BoolVar(&showHelp, "h", false, "Show help information (shorthand)")

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

	// For now, just print a message that the application is not yet implemented
	fmt.Fprintf(os.Stderr, "charon-key: AuthorizedKeysCommand for GitHub SSH keys\n")
	fmt.Fprintf(os.Stderr, "Use --help for usage information\n")
	os.Exit(1)
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

