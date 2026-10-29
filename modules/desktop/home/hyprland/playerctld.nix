{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.playerctld.enable {
    services.playerctld.enable = true;
  };
}
