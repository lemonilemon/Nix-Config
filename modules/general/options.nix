{
  lib,
  config,
  pkgs,
  ...
}:
let
  # WSL borrows the host's hardware: nothing to power-manage, no NIC of its own
  # to manage, no firewall to run.
  isPhysical = config.formFactor != "wsl";
in
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
          default = config.nixos.general.enable && isPhysical;
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

      network = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable && isPhysical;
          description = "Enable my networking settings";
        };

        manager.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.network.enable;
          description = "Manage networking with NetworkManager";
        };

        wifi.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.formFactor == "laptop";
          description = "This host has wireless hardware for NetworkManager to manage";
        };

        firmware.enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.network.enable;
          description = "Install redistributable firmware (wifi, GPU, microcode)";
        };
      };

      firewall = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = config.nixos.general.enable && isPhysical;
          description = "Enable my firewall settings";
        };

        allowedTCPPorts = lib.mkOption {
          type = lib.types.listOf lib.types.port;
          default = [
            22
            80
            443
          ];
          description = "TCP ports accepted from anywhere";
        };

        allowedUDPPorts = lib.mkOption {
          type = lib.types.listOf lib.types.port;
          default = [ ];
          description = "UDP ports accepted from anywhere";
        };

        trustedSubnets = lib.mkOption {
          type = lib.types.listOf lib.types.str;
          default = [ ];
          example = [ "192.168.0.0/24" ];
          description = ''
            IPv4 subnets accepted on the input chain. This is an all-ports
            bypass, not an addition to allowedTCPPorts: every host in a listed
            subnet reaches anything bound to 0.0.0.0, including services added
            later that never opened a port of their own.
          '';
        };
      };

      smb = {
        enable = lib.mkOption {
          type = lib.types.bool;
          # Off by default, unlike the rest of nixos.general, because the
          # shares are not a neutral default: media is served to guests with
          # no authentication, and samba's own openFirewall binds 139/445 and
          # 137/138 on every interface rather than the local subnet. Opt in
          # per host that should genuinely serve files.
          default = false;
          description = "Enable the Samba file server (shares defined in profiles/smb.nix)";
        };
      };
    };
  };
}
