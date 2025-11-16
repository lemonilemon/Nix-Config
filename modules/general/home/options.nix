{
  lib,
  config,
  pkgs,
  osConfig,
  ...
}:
{
  options = {
    home.general = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default =
          if osConfig == null then
            config.home.enable && config.general.enable
          else
            osConfig.home.general.enable;
        description = "Enable my home-manager general settings";
      };

      fonts = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default =
            if osConfig == null then config.home.general.enable else osConfig.home.general.fonts.enable;
          description = "Enable my font settings";
        };
      };

      pdf = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = if osConfig == null then config.home.general.enable else osConfig.home.general.pdf.enable;

          description = "Enable my PDF settings";
        };
      };

      programlangs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default =
            if osConfig == null then config.home.general.enable else osConfig.home.general.programlangs.enable;
          description = "Enable my programming languages settings";
        };
        packages = lib.mkOption {
          type = lib.types.listOf lib.types.package;
          default =
            if osConfig == null then
              lib.mkDefault [
                pkgs.gcc
                pkgs.python3
              ]
            else
              osConfig.home.general.programlangs.packages;
          description = "Packages for programming languages";
        };
      };

      secrets = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default =
            if osConfig == null then config.home.general.enable else osConfig.home.general.secrets.enable;
          description = "Enable my settings of secrets";
        };
      };

      utils = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default =
            if osConfig == null then config.home.general.enable else osConfig.home.general.utils.enable;
          description = "Enable my settings of utilities";
        };
      };
    };
  };
}
