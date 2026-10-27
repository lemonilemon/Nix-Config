{
  lib,
  config,
  ...
}:
{
  config = lib.mkMerge [
    (lib.mkIf config.home.desktop.hyprland.enable {
      services.network-manager-applet.enable = true;
      home.packages = [ config.services.network-manager-applet.package ];
    })

    (lib.mkIf config.home.desktop.hyprland.tailscale.enable {
      services.tailscale-systray = {
        enable = true;
        theme = "dark:nobg";
      };
    })
  ];
}
