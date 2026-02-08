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

        pkgVersion = "1.0.4";

        # Pre-built binaries from GitHub releases.
        # To update: bump pkgVersion and replace the hashes from checksums.txt
        # (convert with: nix-hash --type sha256 --to-sri <hex>)
        assets = {
          x86_64-linux   = { name = "charon-key-linux-amd64";  hash = "sha256-cegmLO1ZLgd0sYYwaceDtTTiC8seBcavwjuGZsrT6aA="; };
          aarch64-linux  = { name = "charon-key-linux-arm64";  hash = "sha256-UzOz3d11ZrCNBv2zzdzfKuexERsi8QP3sDX4bGwQw/g="; };
          x86_64-darwin  = { name = "charon-key-darwin-amd64"; hash = "sha256-3JgFxRynqID5qx6dQ7h1SotkYJVOxRq86JXRz6+n1sY="; };
          aarch64-darwin = { name = "charon-key-darwin-arm64"; hash = "sha256-U+pdVwgH5MWDon/3ulZ4mqxZasuq8NwnSuKo60sCa+k="; };
        };
        asset = assets.${system} or (throw "charon-key: unsupported system ${system}");

        charon-key = pkgs.stdenv.mkDerivation {
          pname = "charon-key";
          version = pkgVersion;

          src = pkgs.fetchurl {
            url = "https://github.com/GDR/charon-key/releases/download/v${pkgVersion}/${asset.name}";
            hash = asset.hash;
          };

          dontUnpack = true;

          installPhase = ''
            mkdir -p $out/bin
            cp $src $out/bin/charon-key
            chmod +x $out/bin/charon-key
          '';

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

