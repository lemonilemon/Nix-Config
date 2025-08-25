{
  config,
  lib,
  ...
}:
{
  config = lib.mkIf config.home.gui-settings.browsers.firefox.enable {
    programs.firefox.enable = false;

    # xdg.mimeApps = {
    #   enable = true;
    #   defaultApplications = {
    #     "text/html" = [ "firefox.desktop" ];
    #     "x-scheme-handler/http" = [ "firefox.desktop" ];
    #     "x-scheme-handler/https" = [ "firefox.desktop" ];
    #     "x-scheme-handler/about" = [ "firefox.desktop" ];
    #     "x-scheme-handler/unknown" = [ "firefox.desktop" ];
    #   };
    # };
    xdg.desktopEntries.firefox = {
      name = "Firefox";
      exec = "firefox -no-remote %U";
      icon = "firefox";
      terminal = false;
      categories = [
        "Network"
        "WebBrowser"
      ];
    };
  };
}
