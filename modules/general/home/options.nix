{
  lib,
  config,
  pkgs,
  osConfig ? null,
  helpers,
  ...
}:
{
  options = {
    home.general = {
      enable = helpers.mkHomeOpt {
        inherit osConfig;
        path = "home.general.enable";
        default = config.home.enable && config.general.enable;
        description = "Enable my home-manager general settings";
      };

      fonts = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.fonts.enable";
          default = config.home.general.enable;
          description = "Enable my font settings";
        };
      };

      pdf = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.pdf.enable";
          default = config.home.general.enable;
          description = "Enable my PDF settings";
        };
      };

      programlangs = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.programlangs.enable";
          default = config.home.general.enable;
          description = "Enable my programming languages settings";
        };
        packages = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.programlangs.packages";
          type = lib.types.listOf lib.types.package;
          default = lib.mkDefault [
            pkgs.gcc
            pkgs.python3
          ];
          description = "Packages for programming languages";
        };
      };

      secrets = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.secrets.enable";
          default = config.home.general.enable;
          description = "Enable my settings of secrets";
        };
      };

      utils = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.utils.enable";
          default = config.home.general.enable;
          description = "Enable my settings of utilities";
        };
      };
    };
  };
}
