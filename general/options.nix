{
  lib,
  config,
  pkgs,
  ...
}:
{
  options = {
    home.general-settings = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable;
        description = "Enable my home-manager general settings";
      };

      fonts = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general-settings.enable;
          description = "Enable my font settings";
        };
      };

      pdf = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general-settings.enable;
          description = "Enable my PDF settings";
        };
      };

      programlangs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general-settings.enable;
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

      secrets = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general-settings.enable;
          description = "Enable my settings of secrets";
        };
      };

      utils = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general-settings.enable;
          description = "Enable my settings of utilities";
        };
      };
    };

    nixos.general-settings = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable;
        description = "Enable my NixOS general settings";
      };

      nixld = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general-settings.enable;
          description = "Enable nix-ld";
        };
      };

      nix = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general-settings.enable;
          description = "Enable my Nix settings";
        };
      };
    };
  };
}
