{
  lib,
  config,
  ...
}:
{
  options = {
    home.desktop = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable;
        description = "Enable desktop environment configuration for home-manager modules";
      };

      hyprland.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.enable;
        description = "Enable hyprland for desktop environment";
      };
    };

    nixos.desktop = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable;
        description = "Enable desktop environment configuration";
      };
      gnome.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.desktop.enable;
        description = "Enable gnome for desktop environment";
      };
      hyprland.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.desktop.enable;
        description = "Enable hyprland for desktop environment";
      };
      displayManager = lib.mkOption {
        type = lib.types.str;
        default = "sddm";
        description = "The display manager to use for the desktop environment";
      };
    };
  };
}
