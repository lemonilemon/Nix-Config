{
  username,
  ...
}:
{
  home.file.".config/hypr/wallpaper.png" = {
    enable = true;
    source = ./wallpaper.png;
  };
  services.hyprpaper = {
    enable = true;
    settings = {
      ipc = "on";
      splash = false;
      splash_offset = 2.0;

      preload = [
        "/home/${username}/.config/hypr/wallpaper.png"
      ];

      wallpaper = [
        ",/home/${username}/.config/hypr/wallpaper.png"
      ];
    };
  };
}
