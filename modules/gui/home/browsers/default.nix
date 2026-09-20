{
  config,
  lib,
  ...
}:
let
  cfg = config.home.gui.browsers;

  browsers = {
    zen = {
      inherit (cfg.zen) enable;
      # The zen-browser flake ships its entry as zen-beta.desktop.
      entry = "zen-beta.desktop";
    };
    firefox = {
      inherit (cfg.firefox) enable;
      entry = "firefox.desktop";
    };
  };

  chosen = browsers.${cfg.default};

  anyEnabled = lib.any (b: b.enable) (lib.attrValues browsers);
in
{
  imports = [
    ./firefox.nix
    ./zen.nix
  ];

  config = {
    assertions = [
      {
        assertion = anyEnabled -> chosen.enable;
        message = ''
          home.gui.browsers.default is "${cfg.default}", but
          home.gui.browsers.${cfg.default}.enable is false, which would leave
          http, https and text/html with no handler.
        '';
      }
    ];

    xdg.mimeApps = lib.mkIf chosen.enable {
      enable = true;
      defaultApplications = lib.genAttrs [
        "text/html"
        "x-scheme-handler/http"
        "x-scheme-handler/https"
        "x-scheme-handler/about"
        "x-scheme-handler/unknown"
      ] (_: [ chosen.entry ]);
    };
  };
}
