{
  lib,
  config,
  pkgs,
  ...
}:
{
  options = {
    home.general = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.home.enable && config.general.enable;
        description = "Enable my home-manager general settings";
      };

      fonts = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general.enable;
          description = "Enable my font settings";
        };
      };

      rime = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general.enable;
          description = "Enable Rime IME configuration (臺灣字形 default)";
        };
      };

      pdf = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general.enable;
          description = "Enable my PDF settings";
        };
      };

      programlangs = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general.enable;
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
          default = config.home.general.enable;
          description = "Enable my settings of secrets";
        };
      };

      utils = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.home.general.enable;
          description = "Enable my settings of utilities";
        };
      };
    };

    nixos.general = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = config.nixos.enable && config.general.enable;
        description = "Enable my NixOS general settings";
      };

      nixld = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable;
          description = "Enable nix-ld";
        };
      };

      nix = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable;
          description = "Enable my Nix settings";
        };
      };
    };
  };
}
