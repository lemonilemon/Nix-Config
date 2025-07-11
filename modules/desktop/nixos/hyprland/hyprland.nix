{
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
    };
    services.xserver.desktopManager.runXdgAutostartIfNone = true;
  };
}
