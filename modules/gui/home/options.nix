{
  lib,
  config,
  osConfig ? null,
  helpers,
  ...
}:
{
  options = {
    home.gui = {
      enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.gui.enable";
        default = config.home.enable && config.gui.enable;
        description = "Enable my GUI settings for home-manager modules";
      };
      development = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.gui.development.enable";
          default = config.home.gui.enable;
          description = "Enable development tools";
        };
        web = {
          enable = helpers.mkHomeOpt {
            inherit osConfig;
            path = "home.gui.development.web.enable";
            default = config.home.gui.development.enable;
            description = "Enable web development tools";
          };
        };
      };
      browsers = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.gui.browsers.enable";
          default = config.home.gui.enable;
          description = "Enable the browsers module";
        };

        firefox = {
          enable = helpers.mkHomeOpt {
            inherit osConfig;
            path = "home.gui.browsers.firefox.enable";
            default = config.home.gui.browsers.enable;
            description = "Enable firefox for browsing";
          };
        };
        zen = {
          enable = helpers.mkHomeOpt {
            inherit osConfig;
            path = "home.gui.browsers.zen.enable";
            default = config.home.gui.browsers.enable;
            description = "Enable zen for browsing";
          };
        };
      };

      apps = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.gui.apps.enable";
          default = config.home.gui.enable;
          description = "Enable the apps module";
        };
      };

      kitty = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.gui.kitty.enable";
          default = config.home.gui.enable;
          description = "Enable kitty terminal";
        };
      };
    };
  };
}
