{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    home.file.".config/hypr/wallpaper/pixel_sunset.jxl" = {
      enable = true;
      source = ./wallpaper/pixel_sunset.jxl;
    };
    services.hyprpaper = {
      enable = true;
      settings = {
        ipc = "on";
        splash = false;
        splash_offset = 2;
        splash_opacity = 0.4;

        wallpaper = {
          monitor = "";
          path = "${config.home.homeDirectory}/.config/hypr/wallpaper/pixel_sunset.jxl";
          fit_mode = "cover";
        };
      };
    };
  };
}
