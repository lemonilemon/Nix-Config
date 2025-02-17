{
  lib,
  pkgs,
  config,
  ...
}:
{
  options = {
    gui-settings.desktop.gnome.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.gui-settings.desktop.gnome.enable;
      description = "Enable gnome for desktop environment";
    };
  };
  config = lib.mkIf config.gui-settings.desktop.gnome.enable {
  };
}
