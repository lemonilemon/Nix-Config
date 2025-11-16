{
  lib,
  ...
}:
{
  imports = [
    ./gui
    ./general
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
