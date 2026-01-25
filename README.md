# charon-key

SSH AuthorizedKeysCommand that fetches SSH public keys from GitHub and merges them with existing authorized_keys files.

## Overview

`charon-key` is a Go application designed to be used as an `AuthorizedKeysCommand` in SSH daemon configuration. It:

- Maps SSH usernames to GitHub usernames (many-to-many mapping)
- Fetches SSH public keys from GitHub's public API
- Caches keys with configurable TTL to reduce API calls
- Merges GitHub keys with existing `~/.ssh/authorized_keys` files
- Provides graceful fallback to cached keys when GitHub is unreachable

## Features

- **Many-to-many user mapping**: Multiple SSH users can map to multiple GitHub users
- **Caching**: Configurable cache with TTL to minimize GitHub API calls
- **Offline support**: Falls back to cached keys when GitHub is unreachable
- **Cross-platform**: Works on macOS and Linux
- **Nix packaging**: Built and distributed via Nix

## Installation

### Using Nix

```bash
nix build
```

## Usage

### Basic Usage

```bash
charon-key --user-map alice:alice-github,bob:bob-github
```

### With Options

```bash
charon-key \
  --user-map alice:alice-github,bob:bob-github \
  --cache-dir /var/cache/charon-key \
  --cache-ttl 10 \
  --log-level debug
```

### SSH Configuration

Add to `/etc/ssh/sshd_config`:

```
AuthorizedKeysCommand /path/to/charon-key --user-map <mapping>
AuthorizedKeysCommandUser root
```

## User Mapping Format

The `--user-map` argument accepts comma-separated pairs in the format `sshuser:githubuser`:

- Single mapping: `alice:alice-github`
- Multiple mappings: `alice:alice-github,alice:shared-github,bob:bob-github`
- Wildcard (all SSH users): `*:dgarifullin`

When multiple GitHub users are mapped to the same SSH user, their keys are merged.

## Options

- `--user-map <mapping>` (required): User mapping in format `sshuser:githubuser`
- `--cache-dir <dir>` (optional): Cache directory path (default: OS temp directory)
- `--cache-ttl <minutes>` (optional): Cache TTL in minutes (default: 5)
- `--log-level <level>` (optional): Log level: debug|info|warn|error (default: info)
- `-h, --help`: Show help information
- `-v, --version`: Show version information

## Development

### Prerequisites

- Nix with flakes enabled
- Go 1.21+ (provided by Nix flake)

### Setup

```bash
# Enter development environment
nix develop

# Build the application
go build -o bin/charon-key ./cmd/charon-key

# Run tests
go test ./...
```

## License

[Add license information]

## Status

🚧 **In Development** - This project is currently under active development.

