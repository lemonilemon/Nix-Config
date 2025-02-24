{
  lib,
  config,
  ...
}:
{
  options = {
    gui-settings.desktop.hyprland.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.gui-settings.desktop.enable;
      description = "Enable gnome for desktop environment";
    };
  };
  config = lib.mkIf config.gui-settings.desktop.hyprland.enable {
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
