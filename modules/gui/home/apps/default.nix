{
  inputs,
  pkgs,
  config,
  lib,
  ...
}:
{
  config = lib.mkIf config.home.gui.apps.enable {
    home.packages = with pkgs; [
      webcord
      spotify
      vlc
      tradingview
      gopeed
      podman-desktop
      obsidian
      # libreoffice
      # inputs.claude-desktop.packages.${system}.claude-desktop
    ];
    programs.ssh = {
      enable = true;
      enableDefaultConfig = false;
      includes = [
        "~/.ssh/1Password/config"
        # Host inventory (Host aliases, IPs), decrypted by sops-nix from
        # secrets/ssh-hosts.conf. Edit with `sops secrets/ssh-hosts.conf`,
        # then rebuild. On a machine without the age key the file is simply
        # absent and OpenSSH ignores the Include.
        "~/.ssh/config.inventory"
        # Hand-edited scratch hosts (unversioned). For entries that churn too
        # often to deserve a sops edit + rebuild, or that should stay off git
        # entirely; OpenSSH ignores this too while the file does not exist.
        "~/.ssh/config.local"
      ];
      settings = {
        "*" = {
          ForwardAgent = true;
          IdentityAgent = "~/.1password/agent.sock";
        };
      };
    };

    sops = {
      age.keyFile = "${config.home.homeDirectory}/.config/sops/age/keys.txt";
      secrets."ssh-hosts" = {
        sopsFile = ../../../../secrets/ssh-hosts.conf;
        format = "binary";
        path = "${config.home.homeDirectory}/.ssh/config.inventory";
      };
    };
  };
}
