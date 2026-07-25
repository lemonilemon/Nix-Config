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

      power = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable && config.formFactor != "wsl";
          description = "Enable my power management settings";
        };

        governor = lib.mkOption {
          type = lib.types.enum [
            "performance"
            "powersave"
            "schedutil"
            "ondemand"
            "conservative"
          ];
          default = if config.formFactor == "laptop" then "powersave" else "performance";
          description = "CPU frequency governor, applied only when auto-cpufreq is not managing it";
        };

        autoCpufreq = {
          enable = lib.mkOption {
            type = lib.types.bool;
            default = config.formFactor == "laptop";
            description = "Enable auto-cpufreq; it owns the governor at runtime when on";
          };

          settings = lib.mkOption {
            type = (pkgs.formats.ini { }).type;
            default =
              if config.formFactor == "laptop" then
                {
                  battery = {
                    governor = "powersave";
                    turbo = "never";
                  };
                  charger = {
                    governor = "powersave";
                    turbo = "auto";
                  };
                }
              else
                { };
            description = "Settings passed through to services.auto-cpufreq";
          };
        };

        powertop.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor == "laptop";
          description = "Enable powertop tunings; battery-oriented, off on desktops";
        };

        thermald.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.power.enable;
          description = "Enable thermald; Intel-only, inert on AMD hardware";
        };

        upower.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor == "laptop";
          description = "Enable UPower battery reporting";
        };
      };
    };
  };
}
