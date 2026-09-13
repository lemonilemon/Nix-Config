{
  config,
  lib,
  ...
}:
{
  config = lib.mkIf config.home.gui.browsers.firefox.enable {
    programs.firefox.enable = false;

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
