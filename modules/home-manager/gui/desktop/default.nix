{
  lib,
  config,
  ...
}:
{
  imports = [
    ./gnome.nix
    ./hyprland
  ];
  options = {
    gui-settings.desktop.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.gui-settings.enable;
      description = "Enable desktop environment configuration";
    };
  };
  config = lib.mkIf config.gui-settings.desktop.enable { };
}
