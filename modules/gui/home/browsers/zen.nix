{
  config,
  lib,
  inputs,
  system,
  ...
}:
{

  config = lib.mkIf config.home.gui-settings.browsers.zen.enable {
    home.packages = [
      inputs.zen-browser.packages."${system}".default
    ];
    xdg.mimeApps = {
      enable = true;
      defaultApplications = {
        "text/html" = [ "zen-beta.desktop" ];
        "x-scheme-handler/http" = [ "zen-beta.desktop" ];
        "x-scheme-handler/https" = [ "zen-beta.desktop" ];
        "x-scheme-handler/about" = [ "zen-beta.desktop" ];
        "x-scheme-handler/unknown" = [ "zen-beta.desktop" ];
      };
    };
  };
}
