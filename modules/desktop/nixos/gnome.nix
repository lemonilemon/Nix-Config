{
  lib,
  pkgs,
  config,
  ...
}:
{
  config = lib.mkIf config.nixos.desktop.gnome.enable {
    services.xserver = {
      enable = true;
    };
    services.desktopManager.gnome.enable = true;
    environment.gnome.excludePackages = (
      with pkgs;
      [
        atomix # puzzle game
        cheese # webcam tool
        epiphany # web browser
        simple-scan # document scanner
        evince # document viewer
        geary # email reader
        gedit # text editor
        gnome-characters
        gnome-music
        gnome-photos
        gnome-terminal
        gnome-tour
        gnome-calculator
        gnome-contacts
        gnome-logs
        gnome-maps
        gnome-screenshot
        gnome-connections
        gnome-console
        hitori # sudoku game
        iagno # go game
        tali # poker game
        totem # video player
      ]
    );
  };
}
