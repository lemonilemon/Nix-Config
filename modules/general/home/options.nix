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

      rime = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.rime.enable";
          default = config.home.general.enable;
          description = "Enable Rime IME configuration (臺灣字形 default)";
        };
      };

      obsidian = {
        enable = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.obsidian.enable";
          default = config.home.general.enable;
          description = "Enable the Obsidian vault: the app, nvim's workspace, and its Syncthing folder";
        };

        vaultPath = helpers.mkHomeOpt {
          inherit osConfig;
          path = "home.general.obsidian.vaultPath";
          type = lib.types.str;
          default = "${config.home.homeDirectory}/Documents/notes";
          description = "Absolute path of the Obsidian vault";
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
