{
  pkgs,
  lib,
  ...
}:
{
  home.packages = with pkgs; [
    imv
  ];
  # xdg.desktopEntries.nemo = {
  #   name = "Nemo";
  #   exec = "${pkgs.nemo-with-extensions}/bin/nemo";
  # };
  xdg.mimeApps = {
    enable = true;
    defaultApplications = {
      "image/png" = [ "imv.desktop" ];
      "image/jpeg" = [ "imv.desktop" ];
      "image/jpg" = [ "imv.desktop" ];
      "image/gif" = [ "imv.desktop" ];
      "image/bmp" = [ "imv.desktop" ];
      "image/webp" = [ "imv.desktop" ];
    };
  };
}
