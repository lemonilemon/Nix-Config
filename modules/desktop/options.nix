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
        default = config.home.enable && config.desktop.enable;
        description = "Enable desktop environment configuration for home-manager modules";
      };

      hyprland.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.enable;
        description = "Enable hyprland for desktop environment";
      };

      hyprland.eww.enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable Eww bar for Hyprland";
      };

      hyprland.eww.laptopControls.enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable laptop-only Eww controls for clamshell and headless display modes";
      };

      hyprland.waybar.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.hyprland.enable && !config.home.desktop.hyprland.eww.enable;
        description = "Enable Waybar for Hyprland";
      };

      hyprland.dunst.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.hyprland.enable;
        description = "Enable Dunst notifications for Hyprland";
      };

      hyprland.swww.enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable the swww wallpaper engine and picker for Hyprland";
      };

      hyprland.hyprpaper.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.hyprland.enable && !config.home.desktop.hyprland.swww.enable;
        description = "Enable hyprpaper static wallpaper (fallback engine when swww is off)";
      };

      hyprland.wlogout.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.hyprland.enable;
        description = "Enable wlogout power menu for Hyprland";
      };
    };

    nixos.desktop = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable && config.desktop.enable;
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
