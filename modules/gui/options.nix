{
  lib,
  config,
  ...
}:
{
  options = {
    home.gui = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable && config.gui.enable;
        description = "Enable my GUI settings for home-manager modules";
      };
      development = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui.enable;
          description = "Enable development tools";
        };
        web = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.gui.development.enable;
            description = "Enable web development tools";
          };
        };
      };
      browsers = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui.enable;
          description = "Enable the browsers module";
        };

        firefox = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.gui.browsers.enable;
            description = "Enable firefox for browsing";
          };
        };
        zen = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.gui.browsers.enable;
            description = "Enable zen for browsing";
          };
        };
      };

      apps = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui.enable;
          description = "Enable the apps module";
        };
        libreoffice = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.home.gui.apps.enable;
            description = "Enable the LibreOffice office suite";
          };
        };
      };

      kitty = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.gui.enable;
          description = "Enable kitty terminal";
        };
      };
    };

    nixos.gui = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable && config.gui.enable;
        description = "Enable my GUI settings for NixOS modules";
      };
      development = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.gui.enable;
          description = "Enable development tools for NixOS modules";
        };
        web = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.nixos.gui.development.enable;
            description = "Enable web development tools for NixOS modules";
          };
        };
      };
      apps = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.gui.enable;
          description = "Enable the apps module for NixOS modules";
        };
      };
    };
  };
}
