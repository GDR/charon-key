{
  description = "SSH AuthorizedKeysCommand for GitHub SSH keys";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    (flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        pkgVersion = "1.0.3";

        charon-key = pkgs.buildGoModule {
          pname = "charon-key";
          version = pkgVersion;
          # Use self to include all git-tracked files
          src = self;

          # The subpackage to build
          subPackages = [ "cmd/charon-key" ];

          # No dependencies yet, so vendorHash is null
          # When dependencies are added, this will be automatically calculated
          vendorHash = null;

          # Build flags for version information
          ldflags = [
            "-X main.version=${pkgVersion}"
            "-X main.commit=${self.rev or "unknown"}"
            "-X main.date=${self.lastModifiedDate or "unknown"}"
          ];

          meta = with pkgs.lib; {
            description = "SSH AuthorizedKeysCommand that fetches SSH public keys from GitHub";
            homepage = "https://github.com/gdr/charon-key";
            license = licenses.mit;
            maintainers = [ ];
            platforms = platforms.unix;
          };
        };
      in
      {
        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go compiler and tools
            go
            gopls
            gofumpt
            golangci-lint
            gomodifytags
            gotools
            gopkgs
            gotests
            impl
            reftools

            # Development tools
            git
            gnumake
            just  # Task runner (optional)
          ];

          shellHook = ''
            echo "🐹 Golang development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Available tools:"
            echo "  - go: Go compiler"
            echo "  - gopls: Go language server"
            echo "  - gofumpt: Go formatter"
            echo "  - golangci-lint: Go linter"
            echo ""
            echo "Build commands:"
            echo "  nix build          - Build the application"
            echo "  nix run            - Run the application"
            echo ""
            
            # Set up Go environment
            export GOPATH="$HOME/go"
            export PATH="$GOPATH/bin:$PATH"
            
            # Enable Go modules if not already set
            export GO111MODULE=on
          '';
        };

        # Package output
        packages.default = charon-key;

        # App output (for nix run)
        apps.default = flake-utils.lib.mkApp {
          drv = charon-key;
        };
      }
    ))
    //
    {
      # NixOS module for declarative charon-key configuration
      nixosModules.default = import ./module.nix self;

      # nix-darwin module for macOS
      darwinModules.default = import ./darwin-module.nix self;
    };
}

