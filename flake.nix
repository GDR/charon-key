{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
    {
      packages = {
        default = pkgs.buildGoModule {
          pname = "charon-key";
          version = "0.1.0";
          
          src = ./.;
          
          vendorHash = "sha256-kYbVtbVorI+cCMAqizYfSm2BcSjyKmICkcKahi9Llx0=";
          
          meta = with pkgs.lib; {
            description = "Charon Key Go application";
            homepage = "https://github.com/gdr/charon-key";
            license = licenses.mit;
            maintainers = [ {
              name = "Damir Garifullin";
              github = "gdr";
            } ];
          };
        };
      };

      devShell = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          gopls
          gotools
          go-tools
        ];
      };
    });
}