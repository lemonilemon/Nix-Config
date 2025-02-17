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
      default = config.gui-settings.desktop.enable;
      description = "Enable gnome for desktop environment";
    };
  };
  config = lib.mkIf config.gui-settings.desktop.gnome.enable {
    services.xserver = {
      enable = true;
      displayManager.gdm.enable = true;
      desktopManager.gnome.enable = true;
    };
    environment.gnome.excludePackages = (
      with pkgs;
      [
        atomix # puzzle game
        cheese # webcam tool
        epiphany # web browser
        evince # document viewer
        geary # email reader
        gedit # text editor
        gnome-characters
        gnome-music
        gnome-photos
        gnome-terminal
        gnome-tour
        hitori # sudoku game
        iagno # go game
        tali # poker game
        totem # video player
      ]
    );
  };
}
