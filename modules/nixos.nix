{
  lib,
  ...
}:
{
  imports = [
    ./cli
    ./gui
    ./general
    ./desktop
  ];
  config = {
    home-manager.backupFileExtension = "backup";
    environment.pathsToLink = [
      "/share/applications"
      "/share/xdg-desktop-portal-gtk"
    ];
    catppuccin.enable = lib.mkDefault true;
    catppuccin.flavor = lib.mkDefault "mocha";
  };
}
