{
  pkgs,
  config,
  lib,
  system,
  inputs,
  ...
}:
{
  config = lib.mkIf config.home.gui-settings.apps.enable {
    home.packages = with pkgs; [
      webcord
      spotify
      vlc
      tradingview
      inputs.claude-desktop.packages.${system}.claude-desktop
    ];
    programs.ssh = {
      enable = true;
      forwardAgent = true;
      extraConfig = ''
        Host *
            IdentityAgent ~/.1password/agent.sock
        Include ~/.ssh/1Password/config
      '';
    };
  };
}
