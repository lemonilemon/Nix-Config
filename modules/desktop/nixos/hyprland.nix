{
  pkgs,
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.nixos.desktop.hyprland.enable {
    hardware = {
      graphics.enable = true;
      nvidia.modesetting.enable = true;
    };
    programs.hyprland = {
      enable = true; # enable Hyprland
      xwayland.enable = true;
      withUWSM = true; # launch Hyprland as a systemd user session via UWSM
    };
    services.xserver.desktopManager.runXdgAutostartIfNone = true;
    xdg.portal = {
      enable = true;
      extraPortals = with pkgs; [
        xdg-desktop-portal-hyprland
        xdg-desktop-portal-gtk
      ];
      config = {
        common = {
          default = [
            "hyprland"
            "gtk"
          ];
          "org.freedesktop.impl.portal.Settings" = [ "gtk" ];
        };
      };
    };

  };
}
