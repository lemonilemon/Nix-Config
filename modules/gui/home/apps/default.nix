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
        # Hand-edited host inventory (Host aliases, IPs). Deliberately outside
        # Nix and git: hosts churn too often to deserve a rebuild, and OpenSSH
        # ignores the Include while the file does not exist yet.
        "~/.ssh/config.local"
      ];
      settings = {
        "*" = {
          ForwardAgent = true;
          IdentityAgent = "~/.1password/agent.sock";
        };
      };
    };
  };
}
