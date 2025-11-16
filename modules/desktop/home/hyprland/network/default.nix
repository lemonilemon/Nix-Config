{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    services.network-manager-applet.enable = true;
  };
}
