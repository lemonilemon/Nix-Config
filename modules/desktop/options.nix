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

      hyprland.awww.enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = "Enable the awww (formerly swww) wallpaper engine and picker for Hyprland";
      };

      hyprland.hyprpaper.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.hyprland.enable && !config.home.desktop.hyprland.awww.enable;
        description = "Enable hyprpaper static wallpaper (fallback engine when awww is off)";
      };

      hyprland.wlogout.enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.desktop.hyprland.enable;
        description = "Enable wlogout power menu for Hyprland";
      };

      hyprland.idle = {
        lockTimeout = lib.mkOption {
          type = lib.types.int;
          default = 900;
          description = "Seconds of idle before the session locks";
        };

        dpmsTimeout = lib.mkOption {
          type = lib.types.int;
          default = 1200;
          description = "Seconds of idle before the displays are switched off";
        };

        suspend = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.formFactor == "laptop";
            description = "Suspend the machine after suspendTimeout of idle";
          };
        };

        suspendTimeout = lib.mkOption {
          type = lib.types.int;
          default = 1800;
          description = "Seconds of idle before suspending, when suspend is enabled";
        };
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
