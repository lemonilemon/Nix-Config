{
  lib,
  config,
  pkgs,
  ...
}:
{
  options = {
    general-settings = {
      programlangs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.general-settings.enable;
          description = "Enable my programming languages settings";
        };
        packages = lib.mkOption {
          type = lib.types.listOf lib.types.package;
          default = lib.mkDefault [
            pkgs.gcc
            pkgs.python3
          ];
          description = "Packages for programming languages";
        };
      };
    };
  };

  config = lib.mkIf config.general-settings.programlangs.enable {
    home.packages = config.general-settings.programlangs.packages;
  };
}
