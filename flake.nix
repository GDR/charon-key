{
  description = "Golang development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
      in
      {
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
            
            # Set up Go environment
            export GOPATH="$HOME/go"
            export PATH="$GOPATH/bin:$PATH"
            
            # Enable Go modules if not already set
            export GO111MODULE=on
          '';
        };
      }
    );
}

