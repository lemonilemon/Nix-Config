{
  config,
  lib,
  ...
}:
{
  config = lib.mkIf config.home.gui-settings.browsers.firefox.enable {
    programs.firefox.enable = true;
    xdg.mimeApps = {
      enable = true;
      defaultApplications = {
        "text/html" = [ "firefox.desktop" ];
        "x-scheme-handler/http" = [ "firefox.desktop" ];
        "x-scheme-handler/https" = [ "firefox.desktop" ];
        "x-scheme-handler/about" = [ "firefox.desktop" ];
        "x-scheme-handler/unknown" = [ "firefox.desktop" ];
      };

    };
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
    # Use no-remote as default
    nixpkgs.overlays = [
      (final: prev: {
        firefox = prev.firefox.overrideAttrs (oldAttrs: {
          buildCommand = ''
            ${oldAttrs.buildCommand}
            mv $out/bin/firefox $out/bin/firefox-bin
            cat > $out/bin/firefox <<EOF
            #!/bin/sh
            exec $out/bin/firefox-bin -no-remote "\$@"
            EOF
            chmod +x $out/bin/firefox
          '';
        });
      })
    ];
  };
}
