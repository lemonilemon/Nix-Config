{
  lib,
  config,
  osConfig ? null,
  helpers,
  ...
}:
{
  options = {
    home.desktop = {
      enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.enable";
        default = config.home.enable && config.desktop.enable;
        description = "Enable desktop environment configuration for home-manager modules";
      };

      hyprland.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.enable";
        default = config.home.desktop.enable;
        description = "Enable hyprland for desktop environment";
      };

      hyprland.eww.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.eww.enable";
        default = false;
        description = "Enable Eww bar for Hyprland";
      };

      hyprland.eww.laptopControls.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.eww.laptopControls.enable";
        default = false;
        description = "Enable laptop-only Eww controls for clamshell and headless display modes";
      };

      hyprland.waybar.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.waybar.enable";
        default = config.home.desktop.hyprland.enable && !config.home.desktop.hyprland.eww.enable;
        description = "Enable Waybar for Hyprland";
      };

      hyprland.dunst.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.dunst.enable";
        default = config.home.desktop.hyprland.enable;
        description = "Enable Dunst notifications for Hyprland";
      };

      hyprland.awww.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.awww.enable";
        default = false;
        description = "Enable the awww (formerly swww) wallpaper engine and picker for Hyprland";
      };

      hyprland.hyprpaper.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.hyprpaper.enable";
        default = config.home.desktop.hyprland.enable && !config.home.desktop.hyprland.awww.enable;
        description = "Enable hyprpaper static wallpaper (fallback engine when awww is off)";
      };

      hyprland.wlogout.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.wlogout.enable";
        default = config.home.desktop.hyprland.enable;
        description = "Enable wlogout power menu for Hyprland";
      };

      hyprland.idle.lockTimeout = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.lockTimeout";
        type = lib.types.int;
        default = 900;
        description = "Seconds of idle before the session locks";
      };

      hyprland.idle.dpmsTimeout = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.dpmsTimeout";
        type = lib.types.int;
        default = 1200;
        description = "Seconds of idle before the displays are switched off";
      };

      hyprland.idle.suspend.enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.suspend.enable";
        default = false;
        description = "Suspend the machine after suspendTimeout of idle";
      };

      hyprland.idle.suspendTimeout = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.desktop.hyprland.idle.suspendTimeout";
        type = lib.types.int;
        default = 1800;
        description = "Seconds of idle before suspending, when suspend is enabled";
      };
    };
  };
}
