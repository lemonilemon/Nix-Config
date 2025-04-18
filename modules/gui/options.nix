{
  lib,
  config,
  ...
}:
{
  options = {
    home.gui-settings = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable;
        description = "Enable my GUI settings for home-manager modules";
      };

      browsers = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui-settings.enable;
          description = "Enable the browsers module";
        };

        firefox = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.gui-settings.browsers.enable;
            description = "Enable firefox for browsing";
          };
        };
        zen = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.gui-settings.browsers.enable;
            description = "Enable zen for browsing";
          };
        };
      };

      apps = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui-settings.enable;
          description = "Enable the apps module";
        };
      };

      kitty = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui-settings.enable;
          description = "Enable kitty terminal";
        };
      };
    };

    nixos.gui-settings = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable;
        description = "Enable my GUI settings for NixOS modules";
      };
    };
  };
}
