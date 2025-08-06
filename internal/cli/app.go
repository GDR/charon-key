package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charon-key/internal/client"
	"charon-key/internal/config"

	"github.com/urfave/cli/v3"
)

// App creates and configures the CLI application
func NewApp() *cli.Command {
	return &cli.Command{
		Name:  "charon-key",
		Usage: "Fetch SSH public keys from GitHub usernames",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "usernames",
				Aliases: []string{"u"},
				Usage:   "GitHub usernames to fetch keys for",
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "File containing GitHub usernames (one per line)",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Only output keys without usernames",
			},
		},
		Action: run,
	}
}

// run is the main action function for the CLI command
func run(ctx context.Context, c *cli.Command) error {
	cfg := &config.Config{
		Quiet: c.Bool("quiet"),
	}

	// Collect usernames from various sources
	var flagUsernames, fileUsernames []string

	if c.StringSlice("usernames") != nil {
		flagUsernames = c.StringSlice("usernames")
	}

	if filename := c.String("file"); filename != "" {
		var err error
		fileUsernames, err = config.ReadUsernamesFromFile(filename)
		if err != nil {
			return fmt.Errorf("error reading file %s: %v", filename, err)
		}
	}

	argUsernames := c.Args().Slice()
	cfg.MergeUsernames(flagUsernames, fileUsernames, argUsernames)

	if !cfg.HasUsernames() {
		return fmt.Errorf("no GitHub usernames provided. Use --usernames, --file, or provide usernames as arguments")
	}

	return processUsernames(cfg)
}

// processUsernames fetches and outputs SSH keys for all configured usernames
func processUsernames(cfg *config.Config) error {
	githubClient := client.NewGitHubClient()

	for _, username := range cfg.Usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}

		keys, err := githubClient.FetchUserKeys(username)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching keys for %s: %v\n", username, err)
			continue
		}

		if len(keys) == 0 {
			if !cfg.Quiet {
				fmt.Fprintf(os.Stderr, "No SSH keys found for user: %s\n", username)
			}
			continue
		}

		outputKeys(username, keys, cfg.Quiet)
	}

	return nil
}

// outputKeys formats and prints the SSH keys
func outputKeys(username string, keys []string, quiet bool) {
	for _, key := range keys {
		if quiet {
			fmt.Println(key)
		} else {
			fmt.Printf("# %s\n%s\n", username, key)
		}
	}
} 