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
      (spotify.overrideAttrs (old: {
        postFixup = ''
          wrapProgram $out/bin/spotify \
            --set XMODIFIERS "@im=fcitx" \
            --unset ELECTRON_OZONE_PLATFORM_HINT \
            --unset NIXOS_OZONE_WL
        '';
      }))
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
