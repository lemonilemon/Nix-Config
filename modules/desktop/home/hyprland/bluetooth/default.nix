{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    services.mpris-proxy.enable = true;
  };
}
