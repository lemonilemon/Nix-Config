{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    services.blueman-applet.enable = true;
    services.mpris-proxy.enable = true;
  };
}
