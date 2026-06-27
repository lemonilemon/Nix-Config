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
      includes = [ "~/.ssh/1Password/config" ];
      settings = {
        "*" = {
          ForwardAgent = true;
          IdentityAgent = "~/.1password/agent.sock";
        };
      };
    };
  };
}
