{
  pkgs,
  config,
  lib,
  ...
}:
{
  config = lib.mkIf config.home.gui-settings.apps.enable {
    home.packages = with pkgs; [
      webcord
      spotify
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
