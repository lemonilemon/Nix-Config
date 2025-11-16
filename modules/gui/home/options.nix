{
  lib,
  config,
  osConfig,
  ...
}:
{
  options = {
    home.gui = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default =
          if osConfig == null then config.home.enable && config.gui.enable else osConfig.home.gui.enable;
        description = "Enable my GUI settings for home-manager modules";
      };
      development = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.gui.enable else osConfig.home.gui.development.enable;
          description = "Enable development tools";
        };
        web = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default =
              if osConfig == null then
                config.home.gui.development.enable
              else
                osConfig.home.gui.development.web.enable;
            description = "Enable web development tools";
          };
        };
      };
      browsers = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.gui.enable else osConfig.home.gui.browsers.enable;
          description = "Enable the browsers module";
        };

        firefox = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default =
              if osConfig == null then
                config.home.gui.browsers.enable
              else
                osConfig.home.gui.browsers.firefox.enable;
            description = "Enable firefox for browsing";
          };
        };
        zen = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default =
              if osConfig == null then config.home.gui.browsers.enable else osConfig.home.gui.browsers.zen.enable;
            description = "Enable zen for browsing";
          };
        };
      };

      apps = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.gui.enable else osConfig.home.gui.apps.enable;
          description = "Enable the apps module";
        };
      };

      kitty = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.gui.enable else osConfig.home.gui.kitty.enable;
          description = "Enable kitty terminal";
        };
      };
    };
  };
}
