# Integration Examples for System Dots Flake

This document shows how to integrate `charon-key` into your system dots flake.

## Option 1: Add as Flake Input (Recommended)

In your system dots `flake.nix`, add `charon-key` as an input:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    home-manager.url = "github:nix-community/home-manager";
    
    # Add charon-key flake
    charon-key.url = "path:/path/to/charon-key";  # Local path
    # OR if it's in a git repo:
    # charon-key.url = "github:yourusername/charon-key";
    # charon-key.url = "git+https://github.com/yourusername/charon-key";
  };

  outputs = { self, nixpkgs, home-manager, charon-key, ... }:
    let
      system = "aarch64-darwin";  # or "x86_64-linux", etc.
      pkgs = import nixpkgs { inherit system; };
    in
    {
      # Use in home-manager
      homeConfigurations.youruser = home-manager.lib.homeManagerConfiguration {
        inherit pkgs;
        modules = [
          {
            home.packages = [
              charon-key.packages.${system}.default
            ];
          }
        ];
      };

      # Or use in NixOS configuration
      nixosConfigurations.yourserver = nixpkgs.lib.nixosSystem {
        inherit system;
        modules = [
          {
            environment.systemPackages = [
              charon-key.packages.${system}.default
            ];
          }
        ];
      };
    };
}
```

## Option 2: Use in NixOS SSH Configuration

If you're using NixOS, you can configure SSH directly:

```nix
{
  services.openssh = {
    enable = true;
    authorizedKeysCommand = "${charon-key.packages.${system}.default}/bin/charon-key";
    authorizedKeysCommandUser = "root";
    extraConfig = ''
      AuthorizedKeysCommand ${charon-key.packages.${system}.default}/bin/charon-key --user-map alice:alice-github,bob:bob-github
      AuthorizedKeysCommandUser root
    '';
  };
}
```

## Option 3: Use in Darwin (macOS) Configuration

For macOS with `nix-darwin`:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    darwin.url = "github:lnl7/nix-darwin";
    charon-key.url = "path:/path/to/charon-key";
  };

  outputs = { self, nixpkgs, darwin, charon-key, ... }:
    let
      system = "aarch64-darwin";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      darwinConfigurations."yourmac" = darwin.lib.darwinConfiguration {
        inherit system;
        modules = [
          {
            environment.systemPackages = [
              charon-key.packages.${system}.default
            ];
          }
        ];
      };
    };
}
```

## Option 4: Use in Home Manager (macOS/Linux)

For home-manager configuration:

```nix
{
  home.packages = [
    charon-key.packages.${pkgs.system}.default
  ];
}
```

## Option 5: Reference from Git Repository

If your charon-key is in a git repository:

```nix
{
  inputs = {
    charon-key = {
      url = "github:yourusername/charon-key";
      # Optional: pin to a specific branch/commit
      # ref = "main";
      # rev = "abc123...";
    };
  };
}
```

Or use a local path (useful for development):

```nix
{
  inputs = {
    charon-key.url = "path:/Users/dgarifullin/Workspaces/gdr/charon-key";
  };
}
```

## Option 6: Override or Extend

You can also override the package if needed:

```nix
{
  charon-key-override = charon-key.packages.${system}.default.overrideAttrs (old: {
    # Add custom build flags, etc.
  });
}
```

## Complete Example: System Dots Flake

Here's a complete example for a macOS system:

```nix
{
  description = "My system configuration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    darwin.url = "github:lnl7/nix-darwin";
    darwin.inputs.nixpkgs.follows = "nixpkgs";
    home-manager.url = "github:nix-community/home-manager";
    home-manager.inputs.nixpkgs.follows = "nixpkgs";
    
    # Your charon-key flake
    charon-key.url = "path:/Users/dgarifullin/Workspaces/gdr/charon-key";
  };

  outputs = { self, nixpkgs, darwin, home-manager, charon-key, ... }:
    let
      system = "aarch64-darwin";
      pkgs = import nixpkgs { inherit system; };
    in
    {
      darwinConfigurations."yourmac" = darwin.lib.darwinConfiguration {
        inherit system;
        modules = [
          home-manager.darwinModules.home-manager
          {
            environment.systemPackages = [
              charon-key.packages.${system}.default
            ];
            
            home-manager.users.youruser = {
              # Or install via home-manager
              home.packages = [
                charon-key.packages.${system}.default
              ];
            };
          }
        ];
      };
    };
}
```

## Usage After Integration

Once integrated, you can:

1. **Build it**: `nix build .#charon-key` (from your dots repo)
2. **Use in configuration**: Reference `charon-key.packages.${system}.default` in your configs
3. **Update**: Run `nix flake update charon-key` to update the input
4. **Lock**: The flake.lock will track the exact version

## Notes

- The flake outputs packages for all default systems (aarch64-darwin, aarch64-linux, x86_64-darwin, x86_64-linux)
- Use `${system}` to reference the current system
- The package includes the binary at `bin/charon-key`
- You can also use `charon-key.apps.${system}.default` if you want to use `nix run` functionality

