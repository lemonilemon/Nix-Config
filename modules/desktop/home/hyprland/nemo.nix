{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.desktop.hyprland.enable {
    home.packages = with pkgs; [
      nemo-with-extensions
    ];
    xdg.desktopEntries.nemo = {
      name = "Nemo";
      exec = "${pkgs.nemo-with-extensions}/bin/nemo";
    };
    xdg.mimeApps = {
      enable = true;
      defaultApplications = {
        "inode/directory" = [ "nemo.desktop" ];
        "application/x-gnome-saved-search" = [ "nemo.desktop" ];
        "application/zip" = [ "org.gnome.FileRoller.desktop" ];
        "application/rar" = [ "org.gnome.FileRoller.desktop" ];
      };
    };
    dconf = {
      settings = {
        "org/cinnamon/desktop/applications/terminal" = {
          exec = "kitty";
          # exec-arg = ""; # argument
        };
      };
    };
  };
}
