self:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.charon-key;
  charon-key = self.packages.${pkgs.system}.default;

  userMapStr = lib.concatStringsSep "," (
    lib.concatLists (lib.mapAttrsToList (sshUser: githubUsers:
      map (ghUser: "${sshUser}:${ghUser}") githubUsers
    ) cfg.userMap)
  );
in
{
  options.services.charon-key = {
    enable = lib.mkEnableOption "charon-key SSH AuthorizedKeysCommand for GitHub SSH keys";

    userMap = lib.mkOption {
      type = lib.types.attrsOf (lib.types.listOf lib.types.str);
      description = ''
        Mapping of SSH usernames to lists of GitHub usernames.
        Each SSH user maps to one or more GitHub users whose public keys
        will be authorized for login. Use "*" as a wildcard to match any
        SSH user.
      '';
      example = lib.literalExpression ''{
        root = [ "octocat" ];
        deploy = [ "alice" "bob" ];
        "*" = [ "dgarifullin" ];
      }'';
    };
  };

  config = lib.mkIf cfg.enable {
    # Write the wrapper as a real file (not a Nix store symlink) because sshd
    # requires AuthorizedKeysCommand and all parent directories to be owned by
    # root and not group/other-writable.  /nix/store is drwxrwxr-t root nixbld
    # (group-writable), so any binary there fails sshd's auth_secure_path
    # check with "Unsafe AuthorizedKeysCommand".
    system.activationScripts.charon-key-wrapper = lib.stringAfter [ "etc" ] ''
      mkdir -p /etc/ssh
      cat > /etc/ssh/charon-key-wrapper << 'EOF'
#!/bin/sh
exec ${charon-key}/bin/charon-key --user-map ${lib.escapeShellArg userMapStr} "$@"
EOF
      chmod 0755 /etc/ssh/charon-key-wrapper
    '';

    # %u is required because charon-key expects the SSH username as a
    # positional argument (flag.Args()[0]).  When AuthorizedKeysCommand has
    # its own arguments (like --user-map), sshd does NOT auto-append the
    # username — %u must be explicit.
    services.openssh.extraConfig = ''
      AuthorizedKeysCommand /etc/ssh/charon-key-wrapper %u
      AuthorizedKeysCommandUser root
    '';
  };
}
