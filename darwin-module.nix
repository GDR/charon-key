self:
{ config, lib, pkgs, ... }:

let
  cfg = config.services.charon-key;
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
        "*" = [ "dgarifullin" ];
      }'';
    };
  };

  config = lib.mkIf cfg.enable (
    let
      charon-key = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      userMapStr = lib.concatStringsSep "," (
        lib.concatLists (lib.mapAttrsToList (sshUser: githubUsers:
          map (ghUser: "${sshUser}:${ghUser}") githubUsers
        ) cfg.userMap)
      );
    in
    {
      # All Darwin activation goes through postActivation.text
      # (nix-darwin does not support named activation scripts like NixOS)
      system.activationScripts.postActivation.text = ''
        # --- charon-key: wrapper script ---
        # Real file, not store symlink (sshd auth_secure_path requires
        # the binary and all parent dirs to be root-owned, not group-writable;
        # /nix/store is drwxrwxr-t root nixbld)
        mkdir -p /etc/ssh
        cat > /etc/ssh/charon-key-wrapper << 'EOF'
#!/bin/sh
exec ${charon-key}/bin/charon-key --user-map ${lib.escapeShellArg userMapStr} "$@"
EOF
        chmod 0755 /etc/ssh/charon-key-wrapper

        # --- charon-key: sshd config drop-in ---
        # %u is required — charon-key expects the SSH username as a positional
        # argument. When AuthorizedKeysCommand has its own args (--user-map),
        # sshd does NOT auto-append the username.
        mkdir -p /etc/ssh/sshd_config.d
        cat > /etc/ssh/sshd_config.d/50-charon-key.conf << 'SSHCONF'
AuthorizedKeysCommand /etc/ssh/charon-key-wrapper %u
AuthorizedKeysCommandUser root
SSHCONF
        chmod 0644 /etc/ssh/sshd_config.d/50-charon-key.conf

        # --- charon-key: ensure Include directive ---
        if ! grep -q 'Include /etc/ssh/sshd_config.d/\*' /etc/ssh/sshd_config 2>/dev/null; then
          # Prepend so it takes priority (sshd uses first match)
          printf '%s\n' 'Include /etc/ssh/sshd_config.d/*' | cat - /etc/ssh/sshd_config > /etc/ssh/sshd_config.tmp
          mv /etc/ssh/sshd_config.tmp /etc/ssh/sshd_config
        fi

        # --- charon-key: restart sshd ---
        launchctl kickstart -k system/com.openssh.sshd 2>/dev/null || true
      '';
    }
  );
}
